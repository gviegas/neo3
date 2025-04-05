// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"sync"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

// swapchain implements driver.Swapchain.
type swapchain struct {
	d     *Driver
	win   wsi.Window
	qfam  C.uint32_t
	sf    C.VkSurfaceKHR
	sc    C.VkSwapchainKHR
	pf    driver.PixelFmt
	usg   driver.Usage
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
	// It has 1 + len(views) - minImg elements initially and
	// is indexed by viewSync[viewIdx], with viewIdx obtained
	// from Next.
	nextSem []C.VkSemaphore

	// presSem contains semaphores that are waited on by the
	// presentation engine before it presents the images.
	// It has len(views) elements and is indexed by the view
	// indices themselves (this is so because Present does not
	// wait for presentation to complete).
	presSem []C.VkSemaphore

	// queSync contain additional data for queue transfers.
	// Its elements correspond to nextSem's.
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
	// Its length matches that of nexSem and non-nil queSync.
	syncUsed []bool

	// pendOp is used by cmdBuffer to track synchronization
	// state across multiple command buffers and Commit calls.
	// Its indices match those of the views slice.
	pendOp []bool

	// presInfo contains C-allocated memory used during a call
	// to Present. The pWaitSemaphores, pSwapchains and
	// pImageIndices fields, along with the info structure
	// itself, all refer to C memory. They can hold a single
	// element each.
	presInfo *C.VkPresentInfoKHR

	// The swapchain is marked as 'broken' when either
	// suboptimal or out of date errors occur.
	// It is expected that Recreate or Destroy will be
	// called eventually.
	broken bool

	// If Present fails to enqueue the wait operation,
	// badSem will be set to true. syncSetup will not
	// attempt to reuse any elements of presSem in
	// this case.
	badSem bool
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
	presRel *cmdBuffer
	// Where the queue transfer from rendering to
	// presentation occurs.
	presAcq *cmdBuffer
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
				i := v.(*imageView).i
				v.Destroy()
				i.Destroy()
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
	calpha := C.VkCompositeAlphaFlagBitsKHR(1)
	for range 32 {
		if C.VkFlags(calpha)&capab.supportedCompositeAlpha != 0 {
			break
		}
		calpha <<= 1
	}

	// Image format and color space.
	var nfmt C.uint32_t
	res = C.vkGetPhysicalDeviceSurfaceFormatsKHR(s.d.pdev, s.sf, &nfmt, nil)
	if err := checkResult(res); err != nil {
		return err
	}
	fmts := make([]C.VkSurfaceFormatKHR, nfmt)
	res = C.vkGetPhysicalDeviceSurfaceFormatsKHR(s.d.pdev, s.sf, &nfmt, unsafe.SliceData(fmts))
	if err := checkResult(res); err != nil {
		return err
	}
	// TODO: Refine this.
	prefFmts := []struct {
		pf  driver.PixelFmt
		fmt C.VkFormat
	}{
		{driver.RGBA8SRGB, C.VK_FORMAT_R8G8B8A8_SRGB},
		{driver.BGRA8SRGB, C.VK_FORMAT_B8G8R8A8_SRGB},
		{driver.RGBA8Unorm, C.VK_FORMAT_R8G8B8A8_UNORM},
		{driver.BGRA8Unorm, C.VK_FORMAT_B8G8R8A8_UNORM},
		{driver.RGBA16Float, C.VK_FORMAT_R16G16B16A16_SFLOAT},
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
			// XXX: This is non-conformant behavior - advertising
			// undefined format is disallowed.
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

	// Image usage.
	usage := C.VkFlags(C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT)
	s.usg = driver.URenderTarget
	if !s.pf.IsInternal() {
		var fprop C.VkFormatProperties
		C.vkGetPhysicalDeviceFormatProperties(s.d.pdev, convPixelFmt(s.pf), &fprop)
		feat := fprop.optimalTilingFeatures
		// TODO: Consider exposing format features so
		// we can be more flexible here.
		spld := C.VkFlags(C.VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_LINEAR_BIT)
		stor := C.VkFlags(C.VK_FORMAT_FEATURE_STORAGE_IMAGE_BIT)
		if s.pf.IsNonfloatColor() {
			spld = C.VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT
			if s.pf.Size()&3 == 0 {
				stor = C.VK_FORMAT_FEATURE_STORAGE_IMAGE_ATOMIC_BIT
			}
		}
		if feat&spld != 0 && capab.supportedUsageFlags&C.VK_IMAGE_USAGE_SAMPLED_BIT != 0 {
			usage |= C.VK_IMAGE_USAGE_SAMPLED_BIT
			s.usg |= driver.UShaderSample
		}
		if feat&stor != 0 && capab.supportedUsageFlags&C.VK_IMAGE_USAGE_STORAGE_BIT != 0 {
			usage |= C.VK_IMAGE_USAGE_STORAGE_BIT
			s.usg |= driver.UShaderRead | driver.UShaderWrite
		}
		if capab.supportedUsageFlags&C.VK_IMAGE_USAGE_TRANSFER_SRC_BIT != 0 {
			usage |= C.VK_IMAGE_USAGE_TRANSFER_SRC_BIT
			s.usg |= driver.UCopySrc
		}
		if capab.supportedUsageFlags&C.VK_IMAGE_USAGE_TRANSFER_DST_BIT != 0 {
			usage |= C.VK_IMAGE_USAGE_TRANSFER_DST_BIT
			s.usg |= driver.UCopyDst
		}
	}

	// Present mode.
	var nmode C.uint32_t
	res = C.vkGetPhysicalDeviceSurfacePresentModesKHR(s.d.pdev, s.sf, &nmode, nil)
	if err := checkResult(res); err != nil {
		return err
	}
	modes := make([]C.VkPresentModeKHR, nmode)
	res = C.vkGetPhysicalDeviceSurfacePresentModesKHR(s.d.pdev, s.sf, &nmode, unsafe.SliceData(modes))
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
	defer C.vkDestroySwapchainKHR(s.d.dev, s.sc, nil)
	info := C.VkSwapchainCreateInfoKHR{
		sType:            C.VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
		surface:          s.sf,
		minImageCount:    nimg,
		imageFormat:      fmts[ifmt].format,
		imageColorSpace:  fmts[ifmt].colorSpace,
		imageExtent:      extent,
		imageArrayLayers: 1,
		imageUsage:       usage,
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
// It sets the views field of s.
// If len(s.views) is not zero, it calls Destroy on each view.
func (s *swapchain) newViews() error {
	for i := range s.views {
		defer s.views[i].(*imageView).i.Destroy() // Unnecessary currently.
		s.views[i].Destroy()
	}
	var nimg C.uint32_t
	res := C.vkGetSwapchainImagesKHR(s.d.dev, s.sc, &nimg, nil)
	if err := checkResult(res); err != nil {
		return err
	}
	imgs := make([]C.VkImage, nimg)
	res = C.vkGetSwapchainImagesKHR(s.d.dev, s.sc, &nimg, unsafe.SliceData(imgs))
	if err := checkResult(res); err != nil {
		return err
	}
	img := image{
		s:   s,
		fmt: convPixelFmt(s.pf),
		// BUG: Need to check the internal format's numeric type.
		nonfp: !s.pf.IsInternal() && s.pf.IsNonfloatColor(),
		subres: C.VkImageSubresourceRange{
			aspectMask: C.VK_IMAGE_ASPECT_COLOR_BIT,
			levelCount: 1,
			layerCount: 1,
		},
	}
	if len(s.views) != int(nimg) {
		s.views = make([]driver.ImageView, nimg)
	}
	for i := range imgs {
		img := img
		img.img = imgs[i]
		// Notice that view keeps a reference to img.
		view, err := img.NewView(driver.IView2D, 0, 1, 0, 1)
		if err != nil {
			for ; i > 0; i-- {
				defer s.views[i-1].(*imageView).i.Destroy()
				s.views[i-1].Destroy()
			}
			s.views = nil
			return err
		}
		s.views[i] = view
	}
	return nil
}

func (s *swapchain) createSem() (sem C.VkSemaphore, err error) {
	info := C.VkSemaphoreCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO,
	}
	res := C.vkCreateSemaphore(s.d.dev, &info, nil, &sem)
	err = checkResult(res)
	return
}

func (s *swapchain) createQueSync() (qs queueSync, err error) {
	if qs.rendWait, err = s.createSem(); err != nil {
		return
	}
	if qs.presWait, err = s.createSem(); err != nil {
		goto fail
	}
	if qs.presRel, err = s.d.newCmdBuffer(s.qfam); err != nil {
		goto fail
	}
	if qs.presAcq, err = s.d.newCmdBuffer(s.qfam); err != nil {
		goto fail
	}
	return
fail:
	s.destroyQueSync(&qs)
	return
}

func (s *swapchain) destroyQueSync(qs *queueSync) {
	C.vkDestroySemaphore(s.d.dev, qs.rendWait, nil)
	C.vkDestroySemaphore(s.d.dev, qs.presWait, nil)
	if qs.presRel != nil {
		qs.presRel.Destroy()
	}
	if qs.presAcq != nil {
		qs.presAcq.Destroy()
	}
	*qs = queueSync{}
}

// syncSetup creates the synchronization data required for
// presentation of s.
// It sets the nextSem, presSem, queSync, viewSync, syncUsed,
// pendOp, presInfo and badSem fields of s.
// The caller must ensure that no semaphores are in use before
// calling this method.
func (s *swapchain) syncSetup() error {
	n := len(s.views)
	// There must be no acquisitions when this method
	// is called so we do not need to worry about
	// clearing viewSync/pendOp/syncUsed.
	if len(s.viewSync) != n {
		s.viewSync = make([]int, n)
	}
	if len(s.pendOp) != n {
		s.pendOp = make([]bool, n)
	}
	n = 1 + n - s.minImg
	if len(s.syncUsed) != n {
		s.syncUsed = make([]bool, n)
	}

	// presInfo never changes.
	if s.presInfo == nil {
		s.presInfo = (*C.VkPresentInfoKHR)(C.malloc(C.sizeof_VkPresentInfoKHR))
		*s.presInfo = C.VkPresentInfoKHR{
			sType:              C.VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
			waitSemaphoreCount: 1,
			pWaitSemaphores:    (*C.VkSemaphore)(C.malloc(C.sizeof_VkSemaphore)),
			swapchainCount:     1,
			pSwapchains:        (*C.VkSwapchainKHR)(C.malloc(C.sizeof_VkSwapchainKHR)),
			pImageIndices:      (*C.uint32_t)(C.malloc(C.sizeof_uint32_t)),
		}
	}

	if s.qfam == s.d.qfam {
		// Single queue. The rendering command buffer
		// waits a nextSem and signals a presSem.
		// queSync is not needed.
		for i := range s.queSync {
			s.destroyQueSync(&s.queSync[i])
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
			for ; i < n; i++ {
				var qs queueSync
				var err error
				if qs, err = s.createQueSync(); err != nil {
					return err
				}
				s.queSync = append(s.queSync, qs)
			}
		case i > n:
			for ; i > n; i-- {
				s.destroyQueSync(&s.queSync[i-1])
			}
			s.queSync = s.queSync[:n]
		}
	}

	// nextSem elements never become invalid.
	i := len(s.nextSem)
	switch {
	case i < n:
		for ; i < n; i++ {
			sem, err := s.createSem()
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

	// presSem elements may become invalid if Present
	// fails to enqueue the operation.
	if s.badSem {
		for _, x := range s.presSem {
			C.vkDestroySemaphore(s.d.dev, x, nil)
		}
		s.presSem = s.presSem[:0]
		s.badSem = false
	}
	n = len(s.views)
	i = len(s.presSem)
	switch {
	case i < n:
		for ; i < n; i++ {
			sem, err := s.createSem()
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

// moreSync appends a new element to s.nextSem, s.queSync
// (if needed) and s.syncUsed.
func (s *swapchain) moreSync() error {
	sem, err := s.createSem()
	if err != nil {
		return err
	}
	if s.qfam != s.d.qfam {
		qs, err := s.createQueSync()
		if err != nil {
			C.vkDestroySemaphore(s.d.dev, sem, nil)
			return err
		}
		s.queSync = append(s.queSync, qs)
	}
	s.nextSem = append(s.nextSem, sem)
	s.syncUsed = append(s.syncUsed, false)
	return nil
}

// Next returns the index of the next writable image view.
func (s *swapchain) Next() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.broken {
		return -1, driver.ErrSwapchain
	}
	if s.curImg > len(s.views)-s.minImg {
		return -1, driver.ErrNoBackbuffer
	}
	var sync int
	for ; sync < len(s.syncUsed); sync++ {
		if !s.syncUsed[sync] {
			break
		}
	}
	if sync == len(s.syncUsed) {
		if err := s.moreSync(); err != nil {
			return -1, err
		}
	}
	var idx C.uint32_t
	var null C.VkFence
	res := C.vkAcquireNextImageKHR(s.d.dev, s.sc, C.UINT64_MAX, s.nextSem[sync], null, &idx)
	switch res {
	case C.VK_SUCCESS, C.VK_SUBOPTIMAL_KHR:
		s.curImg++
		s.viewSync[idx] = sync
		s.syncUsed[sync] = true
		s.broken = res == C.VK_SUBOPTIMAL_KHR
		return int(idx), nil
	case C.VK_ERROR_OUT_OF_DATE_KHR:
		s.broken = true
		return -1, driver.ErrSwapchain
	default:
		if err := checkResult(res); err != nil {
			return -1, err
		}
		// Should never happen.
		println(res)
		panic("unexpected result in Swapchain.Next")
	}
}

func (s *swapchain) getNextSem(index int) C.VkSemaphore {
	s.mu.Lock()
	defer s.mu.Unlock()
	sync := s.viewSync[index]
	if !s.syncUsed[sync] {
		panic("invalid call to swapchain.getNextSem")
	}
	return s.nextSem[sync]
}

func (s *swapchain) getPresSem(index int) C.VkSemaphore {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.syncUsed[s.viewSync[index]] {
		panic("invalid call to swapchain.getPresSem")
	}
	return s.presSem[index]
}

func (s *swapchain) getQueSync(index int) queueSync {
	s.mu.Lock()
	defer s.mu.Unlock()
	sync := s.viewSync[index]
	if !s.syncUsed[sync] {
		panic("invalid call to swapchain.getQueSync")
	}
	return s.queSync[sync]
}

func (s *swapchain) getViewSync(index int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	sync := s.viewSync[index]
	if !s.syncUsed[sync] {
		panic("invalid call to swapchain.getViewSync")
	}
	return sync
}

// Present presents the image view identified by index.
//
// NOTE: Next may return this index as soon as Present
// completes, which would mean that s.viewSync[index]
// was overwritten. getViewSync must be called before
// this method to ensure that the correct data is made
// available by yieldSync.
func (s *swapchain) Present(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.presInfo.pWaitSemaphores = s.presSem[index]
	*s.presInfo.pSwapchains = s.sc
	*s.presInfo.pImageIndices = C.uint32_t(index)
	s.d.qmus[s.qfam].Lock()
	res := C.vkQueuePresentKHR(s.d.ques[s.qfam], s.presInfo)
	s.d.qmus[s.qfam].Unlock()
	s.curImg--
	switch res {
	case C.VK_SUCCESS:
		return nil
	case C.VK_SUBOPTIMAL_KHR, C.VK_ERROR_OUT_OF_DATE_KHR:
		s.broken = true
		return driver.ErrSwapchain
	case C.VK_ERROR_SURFACE_LOST_KHR, C.VK_ERROR_FULL_SCREEN_EXCLUSIVE_MODE_LOST_EXT:
		s.broken = true
		return driver.ErrWindow
	default:
		if err := checkResult(res); err != nil {
			s.broken = true
			// Unlike the cases above, it cannot be assumed
			// that the wait operation will happen, so the
			// semaphores in presSem must be destroyed.
			s.badSem = true
			return err
		}
		// Should never happen.
		println(res)
		panic("unexpected result in Swapchain.Present")
	}
}

// yieldSync yields synchronization data retained by Next.
// It must be called after Present.
func (s *swapchain) yieldSync(sync int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.syncUsed[sync] {
		panic("invalid call to swapchain.yieldSync")
	}
	s.syncUsed[sync] = false
}

func (s *swapchain) casPendOp(index int, old bool, new bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendOp[index] == old {
		s.pendOp[index] = new
		return true
	}
	return false
}

func (s *swapchain) setPendOp(index int, new bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendOp[index] = new
	return
}

// Recreate recreates the swapchain.
func (s *swapchain) Recreate() error {
	s.d.qmus[s.qfam].Lock()
	C.vkQueueWaitIdle(s.d.ques[s.qfam])
	s.d.qmus[s.qfam].Unlock()
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

// Usage returns the image views' driver.Usage.
func (s *swapchain) Usage() driver.Usage { return s.usg }

// Destroy destroys the swapchain.
func (s *swapchain) Destroy() {
	if s == nil {
		return
	}
	if s.d != nil {
		s.d.qmus[s.d.qfam].Lock()
		C.vkQueueWaitIdle(s.d.ques[s.d.qfam])
		s.d.qmus[s.d.qfam].Unlock()
		if s.qfam != s.d.qfam {
			s.d.qmus[s.qfam].Lock()
			C.vkQueueWaitIdle(s.d.ques[s.qfam])
			s.d.qmus[s.qfam].Unlock()
		}
		if s.presInfo != nil {
			C.free(unsafe.Pointer(s.presInfo.pWaitSemaphores))
			C.free(unsafe.Pointer(s.presInfo.pSwapchains))
			C.free(unsafe.Pointer(s.presInfo.pImageIndices))
			C.free(unsafe.Pointer(s.presInfo))
		}
		for _, x := range s.queSync {
			s.destroyQueSync(&x)
		}
		for _, x := range s.presSem {
			C.vkDestroySemaphore(s.d.dev, x, nil)
		}
		for _, x := range s.nextSem {
			C.vkDestroySemaphore(s.d.dev, x, nil)
		}
		for _, v := range s.views {
			i := v.(*imageView).i
			v.Destroy()
			i.Destroy()
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
