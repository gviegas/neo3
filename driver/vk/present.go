// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"sync"
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/wsi"
)

// swapchain implements driver.Swapchain.
type swapchain struct {
	d     *Driver
	win   wsi.Window
	qfam  C.uint32_t
	sf    C.VkSurfaceKHR
	sc    C.VkSwapchainKHR
	pf    driver.PixelFmt
	imgs  []C.VkImage
	views []driver.ImageView
	mu    sync.Mutex

	// The number of images that can be acquired is given by
	// 	1 + len(views) - minImg
	// curImg is incremented/decremented when images are
	// acquired/released.
	minImg int
	curImg int

	// nextSem contains semaphores that are signaled when the
	// presentation engine is done presenting the images.
	// It has 1 + len(views) - minImg elements and is indexed
	// by viewSync[viewIdx], with viewIdx obtained from Next.
	nextSem []C.VkSemaphore

	// presSem contains semaphores that are waited on by the
	// presentation engine before it presents the images.
	// It has len(views) elements and is indexed by the view
	// indices themselves (this is so because cmdBuffer.Commit
	// does not wait for presentation to complete).
	presSem []C.VkSemaphore

	// queSync contain additional data for queue transfers.
	// It has 1 + len(views) - minImg elements and is indexed
	// by viewSync[viewIdx], with viewIdx obtained from Next.
	// If the same queue is used for both rendering and
	// presentation, queSync is nil.
	queSync []queueSync

	// viewSync contains indices in nextSem/queSync indicating
	// the synchronization data held by an image view.
	// If a view is not pending presentation, the index
	// value is undefined.
	// Its indices match those of the views slice.
	viewSync []int

	// syncUsed indicates which indices in nextSem and queSync
	// are currently in use.
	// Its length is equal to the maximum number of images
	// that can be acquired (i.e., 1 + len(views) - minImg).
	syncUsed []bool

	// The swapchain is marked as 'broken' when either
	// suboptimal or out of date errors occur.
	// It is expected that Recreate or Destroy will be
	// called eventually.
	broken bool
}

// queueSync contains additional synchronization data
// necessary to perform queue transfers.
// It is used by swapchain in cases where the rendering
// and presentation queues differ.
type queueSync struct {
	// Signaled by presRel and waited on by the first
	// rendering command buffer that uses the view.
	rendWait C.VkSemaphore
	// Signaled by the last rendering command buffer
	// that uses the view and waited on by presAcq.
	presWait C.VkSemaphore
	// Where the queue transfer from presentation
	// to rendering occurs.
	presRel driver.CmdBuffer
	// Where the queue transfer from rendering to
	// presentation occurs.
	presAcq driver.CmdBuffer
}

// NewSwapchain creates a new swapchain.
func (d *Driver) NewSwapchain(win wsi.Window, imageCount int) (driver.Swapchain, error) {
	if d.exts[extSurface] && d.exts[extSwapchain] {
		s := &swapchain{
			d:   d,
			win: win,
		}
		if err := s.initSurface(); err != nil {
			return nil, err
		}
		if err := s.initSwapchain(imageCount); err != nil {
			C.vkDestroySurfaceKHR(d.inst, s.sf, nil)
			return nil, err
		}
		if err := s.newViews(); err != nil {
			C.vkDestroySwapchainKHR(d.dev, s.sc, nil)
			C.vkDestroySurfaceKHR(d.inst, s.sf, nil)
			return nil, err
		}
		if err := s.syncSetup(); err != nil {
			for _, v := range s.views {
				v.Destroy()
			}
			C.vkDestroySwapchainKHR(d.dev, s.sc, nil)
			C.vkDestroySurfaceKHR(d.inst, s.sf, nil)
			return nil, err
		}
		return s, nil
	}
	return nil, driver.ErrCannotPresent
}

// initSwapchain creates a new swapchain from s.sf.
// It sets the sc, pf, minImg and curImg fields of s.
func (s *swapchain) initSwapchain(imageCount int) error {
	var capab C.VkSurfaceCapabilitiesKHR
	res := C.vkGetPhysicalDeviceSurfaceCapabilitiesKHR(s.d.pdev, s.sf, &capab)
	if err := checkResult(res); err != nil {
		return err
	}

	// Number of backbuffers.
	nimg := C.uint32_t(imageCount)
	if capab.minImageCount > nimg {
		nimg = capab.minImageCount
	} else if capab.maxImageCount != 0 && capab.maxImageCount < nimg {
		nimg = capab.maxImageCount
	}

	// Image size.
	var extent C.VkExtent2D
	if capab.maxImageExtent == extent {
		return driver.ErrWindow
	}
	if capab.currentExtent.width == ^C.uint32_t(0) {
		extent.width = C.uint32_t(s.win.Width())
		extent.height = C.uint32_t(s.win.Height())
	} else {
		extent = capab.currentExtent
	}

	// Pre-transform.
	xform := capab.currentTransform

	// Composite alpha.
	var calpha C.VkCompositeAlphaFlagBitsKHR
	switch ca := capab.supportedCompositeAlpha; true {
	case ca&C.VK_COMPOSITE_ALPHA_INHERIT_BIT_KHR != 0:
		calpha = C.VK_COMPOSITE_ALPHA_INHERIT_BIT_KHR
	case ca&C.VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR != 0:
		calpha = C.VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR
	default:
		// TODO: Consider choosing whichever is available instead.
		return driver.ErrCompositor
	}

	// Image format and color space.
	var nfmt C.uint32_t
	res = C.vkGetPhysicalDeviceSurfaceFormatsKHR(s.d.pdev, s.sf, &nfmt, nil)
	if err := checkResult(res); err != nil {
		return err
	}
	fmts := make([]C.VkSurfaceFormatKHR, nfmt)
	res = C.vkGetPhysicalDeviceSurfaceFormatsKHR(s.d.pdev, s.sf, &nfmt, &fmts[0])
	if err := checkResult(res); err != nil {
		return err
	}
	prefFmts := []struct {
		pf  driver.PixelFmt
		fmt C.VkFormat
	}{
		{driver.RGBA8sRGB, C.VK_FORMAT_R8G8B8A8_SRGB},
		{driver.BGRA8sRGB, C.VK_FORMAT_B8G8R8A8_SRGB},
		{driver.RGBA8un, C.VK_FORMAT_R8G8B8A8_UNORM},
		{driver.BGRA8un, C.VK_FORMAT_B8G8R8A8_UNORM},
		{driver.RGBA16f, C.VK_FORMAT_R16G16B16A16_SFLOAT},
	}
	ifmt := -1
fmtLoop:
	for i := range prefFmts {
		for j := range fmts {
			if prefFmts[i].fmt == fmts[j].format {
				s.pf = prefFmts[i].pf
				ifmt = j
				break fmtLoop
			}
		}
	}
	if ifmt == -1 {
		if len(fmts) == 1 && fmts[0].format == C.VK_FORMAT_UNDEFINED {
			// This is a thing apparently, and it means that we can
			// pick whatever format we want. However, accordingly to
			// v1.3 of the spec, advertising undefined format is not
			// allowed, but here it is just in case.
			fmts[0].format = prefFmts[0].fmt
			fmts[0].colorSpace = C.VK_COLOR_SPACE_SRGB_NONLINEAR_KHR
			s.pf = prefFmts[0].pf
			ifmt = 0
		} else if len(fmts) > 0 {
			// TODO: Check if this format is one of the predefined
			// driver.PixelFmt values.
			s.pf = internalFmt(fmts[0].format)
			ifmt = 0
		}
		return driver.ErrCannotPresent
	}

	// Present mode.
	var nmode C.uint32_t
	res = C.vkGetPhysicalDeviceSurfacePresentModesKHR(s.d.pdev, s.sf, &nmode, nil)
	if err := checkResult(res); err != nil {
		return err
	}
	modes := make([]C.VkPresentModeKHR, nmode)
	res = C.vkGetPhysicalDeviceSurfacePresentModesKHR(s.d.pdev, s.sf, &nmode, &modes[0])
	if err := checkResult(res); err != nil {
		return err
	}
	mode := C.VkPresentModeKHR(C.VK_PRESENT_MODE_FIFO_KHR)
	//for _, m := range modes {
	//	if m == C.VK_PRESENT_MODE_MAILBOX_KHR {
	//		mode = m
	//		break
	//	}
	//}

	// Swapchain.
	// TODO: Additional usage.
	defer C.vkDestroySwapchainKHR(s.d.dev, s.sc, nil)
	info := C.VkSwapchainCreateInfoKHR{
		sType:            C.VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
		surface:          s.sf,
		minImageCount:    nimg,
		imageFormat:      fmts[ifmt].format,
		imageColorSpace:  fmts[ifmt].colorSpace,
		imageExtent:      extent,
		imageArrayLayers: 1,
		imageUsage:       C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
		imageSharingMode: C.VK_SHARING_MODE_EXCLUSIVE,
		preTransform:     xform,
		compositeAlpha:   calpha,
		presentMode:      mode,
		clipped:          C.VK_TRUE,
		oldSwapchain:     s.sc,
	}
	res = C.vkCreateSwapchainKHR(s.d.dev, &info, nil, &s.sc)
	if err := checkResult(res); err != nil {
		var null C.VkSwapchainKHR
		s.sc = null
		return err
	}
	s.minImg = int(capab.minImageCount)
	s.curImg = 0
	return nil
}

// newViews creates new image views from s.sc.
// It sets the imgs and views fields of s.
// If len(s.views) is not zero, it calls Destroy on each view.
func (s *swapchain) newViews() error {
	var nimg C.uint32_t
	res := C.vkGetSwapchainImagesKHR(s.d.dev, s.sc, &nimg, nil)
	if err := checkResult(res); err != nil {
		return err
	}
	if len(s.imgs) != int(nimg) {
		s.imgs = make([]C.VkImage, nimg)
	}
	res = C.vkGetSwapchainImagesKHR(s.d.dev, s.sc, &nimg, &s.imgs[0])
	if err := checkResult(res); err != nil {
		return err
	}
	info := C.VkImageViewCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		viewType: C.VK_IMAGE_VIEW_TYPE_2D,
		format:   convPixelFmt(s.pf),
		components: C.VkComponentMapping{
			r: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			g: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			b: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			a: C.VK_COMPONENT_SWIZZLE_IDENTITY,
		},
		subresourceRange: C.VkImageSubresourceRange{
			aspectMask: C.VK_IMAGE_ASPECT_COLOR_BIT,
			levelCount: 1,
			layerCount: 1,
		},
	}
	for i := range s.views {
		s.views[i].Destroy()
	}
	if len(s.views) != int(nimg) {
		s.views = make([]driver.ImageView, nimg)
	}
	for i := range s.views {
		info.image = s.imgs[i]
		var view C.VkImageView
		res := C.vkCreateImageView(s.d.dev, &info, nil, &view)
		if err := checkResult(res); err != nil {
			for ; i > 0; i-- {
				s.views[i-1].Destroy()
			}
			s.views = nil
			return err
		}
		if s.views[i] == nil {
			s.views[i] = &imageView{
				s:      s,
				view:   view,
				subres: info.subresourceRange,
			}
		} else {
			*s.views[i].(*imageView) = imageView{
				s:      s,
				view:   view,
				subres: info.subresourceRange,
			}
		}
	}
	return nil
}

// syncSetup creates the synchronization data required for
// presentation of s.
// It sets the nextSem, presSem, queSync, viewSync and syncUsed
// fields of s.
// The caller must ensure that no semaphores are in use before
// calling this method.
func (s *swapchain) syncSetup() error {
	// There must be no acquisitions when this method
	// is called so we do not need to clear viewSync.
	if len(s.viewSync) != len(s.views) {
		s.viewSync = make([]int, len(s.views))
	}
	n := 1 + len(s.views) - s.minImg
	// Likewise, syncUsed must have all elements set
	// to false already.
	if len(s.syncUsed) != n {
		s.syncUsed = make([]bool, n)
	}

	createSem := func() (C.VkSemaphore, error) {
		info := C.VkSemaphoreCreateInfo{
			sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO,
		}
		var sem C.VkSemaphore
		res := C.vkCreateSemaphore(s.d.dev, &info, nil, &sem)
		return sem, checkResult(res)
	}
	destroyQueSync := func(qs queueSync) {
		C.vkDestroySemaphore(s.d.dev, qs.rendWait, nil)
		C.vkDestroySemaphore(s.d.dev, qs.presWait, nil)
		if qs.presRel != nil {
			qs.presRel.Destroy()
		}
		if qs.presAcq != nil {
			qs.presAcq.Destroy()
		}
	}

	if s.qfam == s.d.qfam {
		// Single queue. The rendering command buffer
		// waits a nextSem and signals a presSem.
		// queSync is not needed.
		for i := range s.queSync {
			destroyQueSync(s.queSync[i])
		}
		s.queSync = nil
	} else {
		// Different queues. The rendering command buffer
		// waits for a queSync.rendWait, which is signaled
		// by a queSync.presRel, which in turn waits for a
		// nextSem. Then, the rendering command buffer
		// signals a queSync.presWait, which is waited on
		// by a queSync.presAcq, which then signals a
		// presSem and the presentation can execute.
		i := len(s.queSync)
		switch {
		case i < n:
			var qs queueSync
			var err error
			for ; i < n; i++ {
				qs.rendWait, err = createSem()
				if err != nil {
					destroyQueSync(qs)
					return err
				}
				qs.presWait, err = createSem()
				if err != nil {
					destroyQueSync(qs)
					return err
				}
				qs.presRel, err = s.d.newCmdBuffer(s.qfam)
				if err != nil {
					destroyQueSync(qs)
					return err
				}
				qs.presAcq, err = s.d.newCmdBuffer(s.qfam)
				if err != nil {
					destroyQueSync(qs)
					return err
				}
				s.queSync = append(s.queSync, qs)
			}
		case i > n:
			for ; i > n; i-- {
				destroyQueSync(s.queSync[i-1])
			}
			s.queSync = s.queSync[:n]
		}
	}

	// We only need enough nextSem elements to
	// handle max acquisitions, since rendering
	// does wait completion.
	i := len(s.nextSem)
	switch {
	case i < n:
		for ; i < n; i++ {
			sem, err := createSem()
			if err != nil {
				return err
			}
			s.nextSem = append(s.nextSem, sem)
		}
	case i > n:
		for ; i > n; i-- {
			C.vkDestroySemaphore(s.d.dev, s.nextSem[i-1], nil)
		}
		s.nextSem = s.nextSem[:n]
	}

	// We need an element in presSem for each view
	// because presentation does not wait.
	n = len(s.views)
	i = len(s.presSem)
	switch {
	case i < n:
		for ; i < n; i++ {
			sem, err := createSem()
			if err != nil {
				return err
			}
			s.presSem = append(s.presSem, sem)
		}
	case i > n:
		for ; i > n; i-- {
			C.vkDestroySemaphore(s.d.dev, s.presSem[i-1], nil)
		}
		s.presSem = s.presSem[:n]
	}
	return nil
}

// Views returns the list of image views that comprises
// the swapchain.
func (s *swapchain) Views() []driver.ImageView {
	// It is expected that this method will not be
	// called often.
	var views []driver.ImageView
	return append(views, s.views...)
}

// Next returns the index of the next writable image view.
func (s *swapchain) Next(cb driver.CmdBuffer) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.broken {
		return -1, driver.ErrSwapchain
	}
	if s.curImg > len(s.views)-s.minImg {
		return -1, driver.ErrNoBackbuffer
	}
	sync := -1
	for i := range s.syncUsed {
		if !s.syncUsed[i] {
			sync = i
			break
		}
	}
	if sync == -1 {
		// Should never happen.
		panic("no swapchain sync data to use")
	}
	var idx C.uint32_t
	var null C.VkFence
	res := C.vkAcquireNextImageKHR(s.d.dev, s.sc, C.UINT64_MAX, s.nextSem[sync], null, &idx)
	switch res {
	case C.VK_SUCCESS:
		s.curImg++
		s.viewSync[idx] = sync
		s.syncUsed[sync] = true
		c := cb.(*cmdBuffer)
		if c.qfam != s.qfam {
			// TODO
			panic("not implemented")
		}
		c.sc = s
		c.scView = int(idx)
		c.scNext = true
		c.scPres = false
		return int(idx), nil
	case C.VK_SUBOPTIMAL_KHR:
		s.curImg++
		fallthrough
	case C.VK_ERROR_OUT_OF_DATE_KHR:
		s.broken = true
		return -1, driver.ErrSwapchain
	default:
		if err := checkResult(res); err != nil {
			return -1, err
		}
		// Should never happen.
		println(res)
		panic("unexpected result from swapchain's acquisition")
	}
}

// Present presents the image view identified by index.
func (s *swapchain) Present(index int, cb driver.CmdBuffer) error {
	if s.broken {
		return driver.ErrSwapchain
	}
	c := cb.(*cmdBuffer)
	if c.qfam != s.qfam {
		// TODO
		panic("not implemented")
	}
	c.scPres = true
	return nil
}

// present enqueues an image for presentation.
// It assumes that Next and Present were called and that the
// command buffer(s) they target have been submitted for
// execution.
// waitSem must refer to C memory.
func (s *swapchain) present(index int, waitSem *C.VkSemaphore) error {
	psc := (*C.VkSwapchainKHR)(C.malloc(C.sizeof_VkSwapchainKHR))
	defer C.free(unsafe.Pointer(psc))
	*psc = s.sc
	pidx := (*C.uint32_t)(C.malloc(4))
	defer C.free(unsafe.Pointer(pidx))
	*pidx = C.uint32_t(index)
	info := C.VkPresentInfoKHR{
		sType:              C.VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
		waitSemaphoreCount: 1,
		pWaitSemaphores:    waitSem,
		swapchainCount:     1,
		pSwapchains:        psc,
		pImageIndices:      pidx,
	}
	res := C.vkQueuePresentKHR(s.d.ques[s.qfam], &info)
	switch res {
	case C.VK_SUCCESS:
		return nil
	case C.VK_SUBOPTIMAL_KHR, C.VK_ERROR_OUT_OF_DATE_KHR:
		return driver.ErrSwapchain
	default:
		if err := checkResult(res); err != nil {
			return err
		}
	}
	// Should never happen.
	return errUnknown
}

// Recreate recreates the swapchain.
func (s *swapchain) Recreate() error {
	C.vkQueueWaitIdle(s.d.ques[s.qfam])
	if err := s.initSwapchain(len(s.views)); err != nil {
		return err
	}
	if err := s.newViews(); err != nil {
		return err
	}
	if err := s.syncSetup(); err != nil {
		return err
	}
	s.broken = false
	return nil
}

// Format returns the image views' driver.PixelFmt.
func (s *swapchain) Format() driver.PixelFmt { return s.pf }

// Destroy destroys the swapchain.
func (s *swapchain) Destroy() {
	if s == nil {
		return
	}
	if s.d != nil {
		C.vkQueueWaitIdle(s.d.ques[s.d.qfam])
		if s.qfam != s.d.qfam {
			C.vkQueueWaitIdle(s.d.ques[s.qfam])
		}
		for _, x := range s.queSync {
			C.vkDestroySemaphore(s.d.dev, x.rendWait, nil)
			C.vkDestroySemaphore(s.d.dev, x.presWait, nil)
			if x.presRel != nil {
				x.presRel.Destroy()
			}
			if x.presAcq != nil {
				x.presAcq.Destroy()
			}
		}
		for _, x := range s.presSem {
			C.vkDestroySemaphore(s.d.dev, x, nil)
		}
		for _, x := range s.nextSem {
			C.vkDestroySemaphore(s.d.dev, x, nil)
		}
		for _, v := range s.views {
			v.Destroy()
		}
		C.vkDestroySwapchainKHR(s.d.dev, s.sc, nil)
		C.vkDestroySurfaceKHR(s.d.inst, s.sf, nil)
	}
	*s = swapchain{}
}

// presQueueFor returns the index of a queue that supports
// presentation to a given surface.
// It returns driver.ErrCannotPresent if none of the queues
// support presentation. If the query function itself fails
// for any reason, its error is returned instead.
func (d *Driver) presQueueFor(sf C.VkSurfaceKHR) (C.uint32_t, error) {
	n := C.uint32_t(len(d.ques))
	e := driver.ErrCannotPresent
	var sup C.VkBool32
	for i := C.uint32_t(0); i < n; i++ {
		qfam := (i + d.qfam) % n
		err := checkResult(C.vkGetPhysicalDeviceSurfaceSupportKHR(d.pdev, qfam, sf, &sup))
		if err != nil {
			e = err
			continue
		}
		if sup == C.VK_TRUE {
			return qfam, nil
		}
	}
	return ^C.uint32_t(0), e
}
