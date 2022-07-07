// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"unsafe"

	"github.com/gviegas/scene/driver"
)

// cmdBuffer implements driver.CmdBuffer.
type cmdBuffer struct {
	d      *Driver
	qfam   C.uint32_t
	pool   C.VkCommandPool
	cb     C.VkCommandBuffer
	status cbStatus
	err    error // Why cbFailed.
	pres   []presentOp
}

// cbStatus represents the status of the
// command buffer at a given time.
type cbStatus int

// cbStatus constants.
const (
	// Yet to begun.
	// Set after creation, committing and
	// resetting.
	cbIdle cbStatus = iota
	// Ready to record commands.
	// Set after a successful call to Begin.
	cbBegun
	// Ready to be committed.
	// Set after a successful call to End.
	cbEnded
	// Ongoing commit.
	// Set during a call to Commit.
	cbCommitted
	// Command recording failed.
	// Set when a command cannot be recorded.
	cbFailed
)

// presentOp defines the association between an ongoing
// present operation and a rendering command buffer.
// During a call to Transition in a rendering command buffer,
// swapchain views are identified as such, queue transfers
// are performed as needed, and a new presentOp is added to
// the command buffer representing this dependency.
// At Commit time, the presentOp are used to correctly order
// the queue submissions.
type presentOp struct {
	sc     *swapchain
	view   int
	wait   bool // Rendering must wait on semaphore.
	signal bool // Rendering must signal semaphore.
	qrel   bool // Queue released by sc.qfam.
	qacq   bool // Queue acquired by sc.qfam.
}

// NewCmdBuffer creates a new command buffer.
// Its pool is created using d.qfam.
func (d *Driver) NewCmdBuffer() (driver.CmdBuffer, error) {
	return d.newCmdBuffer(d.qfam)
}

// newCmdBuffer creates a new command buffer.
// The command buffer handle is allocated from an exclusive command pool.
// It must only be submitted to d.ques[qfam].
func (d *Driver) newCmdBuffer(qfam C.uint32_t) (driver.CmdBuffer, error) {
	var pool C.VkCommandPool
	poolInfo := C.VkCommandPoolCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		flags:            C.VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
		queueFamilyIndex: qfam,
	}
	err := checkResult(C.vkCreateCommandPool(d.dev, &poolInfo, nil, &pool))
	if err != nil {
		return nil, err
	}
	var cb C.VkCommandBuffer
	cbInfo := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		commandPool:        pool,
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandBufferCount: 1,
	}
	err = checkResult(C.vkAllocateCommandBuffers(d.dev, &cbInfo, &cb))
	if err != nil {
		C.vkDestroyCommandPool(d.dev, pool, nil)
		return nil, err
	}
	return &cmdBuffer{
		d:    d,
		qfam: qfam,
		pool: pool,
		cb:   cb,
	}, nil
}

// Begin prepares the command buffer for recording.
func (cb *cmdBuffer) Begin() error {
	switch cb.status {
	case cbIdle:
		info := C.VkCommandBufferBeginInfo{
			sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
			flags: C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
		}
		err := checkResult(C.vkBeginCommandBuffer(cb.cb, &info))
		if err != nil {
			return err
		}
		cb.status = cbBegun
		return nil
	case cbBegun, cbFailed:
		// Note that cbFailed is handled on End.
		return nil
	}
	// Client error.
	panic("invalid call to CmdBuffer.Begin")
}

// End ends command recording and prepares the command buffer for execution.
func (cb *cmdBuffer) End() error {
	switch cb.status {
	case cbBegun:
		if err := checkResult(C.vkEndCommandBuffer(cb.cb)); err != nil {
			// Calling Begin implicitly resets cb.cb.
			cb.status = cbIdle
			cb.detachSC()
			return err
		}
		cb.status = cbEnded
		return nil
	case cbEnded:
		return nil
	case cbFailed:
		C.vkEndCommandBuffer(cb.cb)
		C.vkResetCommandBuffer(cb.cb, 0)
		cb.status = cbIdle
		cb.detachSC()
		if cb.err == nil {
			panic("unexpected nil error in failed command recording")
		}
		return cb.err
	}
	// Client error.
	panic("invalid call to CmdBuffer.End")
}

// Reset discards all recorded commands from the command buffer.
func (cb *cmdBuffer) Reset() error {
	switch cb.status {
	case cbCommitted:
		// Client error.
		panic("invalid call to CmdBuffer.Reset")
	case cbBegun, cbFailed:
		// Need to end recording before resetting.
		C.vkEndCommandBuffer(cb.cb)
		fallthrough
	default:
		// In case of failure here, we can rely on the implicit
		// reset done during Begin.
		cb.status = cbIdle
		cb.detachSC()
		err := checkResult(C.vkResetCommandBuffer(cb.cb, 0))
		if err != nil {
			return err
		}
		return nil
	}
}

// Barrier inserts a number of global barriers in the command buffer.
func (cb *cmdBuffer) Barrier(b []driver.Barrier) {
	nb := len(b)
	pb := (*C.VkMemoryBarrier2)(C.malloc(C.sizeof_VkMemoryBarrier2 * C.size_t(nb)))
	sb := unsafe.Slice(pb, nb)
	for i := range sb {
		sb[i] = C.VkMemoryBarrier2{
			sType:         C.VK_STRUCTURE_TYPE_MEMORY_BARRIER_2,
			srcStageMask:  convSync(b[i].SyncBefore),
			srcAccessMask: convAccess(b[i].AccessBefore),
			dstStageMask:  convSync(b[i].SyncAfter),
			dstAccessMask: convAccess(b[i].AccessAfter),
		}
	}
	dep := C.VkDependencyInfo{
		sType:              C.VK_STRUCTURE_TYPE_DEPENDENCY_INFO,
		memoryBarrierCount: C.uint32_t(nb),
		pMemoryBarriers:    pb,
	}
	C.vkCmdPipelineBarrier2(cb.cb, &dep)
	C.free(unsafe.Pointer(pb))
}

// Transition inserts a number of image layout transitions in the
// command buffer.
func (cb *cmdBuffer) Transition(t []driver.Transition) {
	nib := len(t)
	pib := (*C.VkImageMemoryBarrier2)(C.malloc(C.sizeof_VkImageMemoryBarrier2 * C.size_t(nib)))
	sib := unsafe.Slice(pib, nib)
	for i := range sib {
		view := t[i].View.(*imageView)
		sib[i] = C.VkImageMemoryBarrier2{
			sType:            C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER_2,
			srcStageMask:     convSync(t[i].SyncBefore),
			srcAccessMask:    convAccess(t[i].AccessBefore),
			dstStageMask:     convSync(t[i].SyncAfter),
			dstAccessMask:    convAccess(t[i].AccessAfter),
			oldLayout:        convLayout(t[i].LayoutBefore),
			newLayout:        convLayout(t[i].LayoutAfter),
			subresourceRange: view.subres,
		}
		if view.i != nil {
			sib[i].image = view.i.img
			continue
		}
		// For swapchain views, we need to identify
		// dependencies to wait on/signal and possibly
		// perform queue transfers if rendering and
		// presentation queues differ.
		viewIdx := -1
		for i := range view.s.views {
			if view.s.views[i] == view {
				viewIdx = i
				break
			}
		}
		sib[i].image = view.s.imgs[viewIdx]
		presIdx := 0
		for ; presIdx < len(cb.pres); presIdx++ {
			if cb.pres[presIdx].sc == view.s && cb.pres[presIdx].view == viewIdx {
				break
			}
		}
		if presIdx == len(cb.pres) {
			cb.pres = append(cb.pres, presentOp{sc: view.s, view: viewIdx})
		}
		if !view.s.pendOp[viewIdx] {
			view.s.pendOp[viewIdx] = true
			cb.pres[presIdx].wait = true
		}
		if cb.qfam == view.s.qfam {
			if t[i].LayoutAfter == driver.LPresent {
				cb.pres[presIdx].signal = true
			}
			// Just the layout transitions from/to
			// driver.LPresent, which the client is
			// required to perform, will suffice.
			continue
		}
		if t[i].LayoutAfter == driver.LPresent {
			cb.pres[presIdx].signal = true
			// Queue transfer from rendering to presentation.
			// This transfer must always be performed when
			// using different queues.
			sib[i].srcQueueFamilyIndex = cb.qfam
			sib[i].dstQueueFamilyIndex = view.s.qfam
			dep := C.VkDependencyInfo{
				sType:                   C.VK_STRUCTURE_TYPE_DEPENDENCY_INFO,
				imageMemoryBarrierCount: 1,
				pImageMemoryBarriers:    &sib[i],
			}
			syncIdx := view.s.viewSync[viewIdx]
			presAcq := view.s.queSync[syncIdx].presAcq.(*cmdBuffer)
			if err := presAcq.Begin(); err != nil {
				cb.status = cbFailed
				continue
			}
			C.vkCmdPipelineBarrier2(presAcq.cb, &dep)
			if err := presAcq.End(); err != nil {
				cb.status = cbFailed
				continue
			}
			cb.pres[presIdx].qacq = true
			continue
		}
		if t[i].LayoutBefore == driver.LPresent {
			// Queue transfer from presentation to rendering.
			// This transfer can be skipped by transitioning
			// from driver.LUndefined instead.
			sib[i].srcQueueFamilyIndex = view.s.qfam
			sib[i].dstQueueFamilyIndex = cb.qfam
			dep := C.VkDependencyInfo{
				sType:                   C.VK_STRUCTURE_TYPE_DEPENDENCY_INFO,
				imageMemoryBarrierCount: 1,
				pImageMemoryBarriers:    &sib[i],
			}
			syncIdx := view.s.viewSync[viewIdx]
			presRel := view.s.queSync[syncIdx].presRel.(*cmdBuffer)
			if err := presRel.Begin(); err != nil {
				cb.status = cbFailed
				continue
			}
			C.vkCmdPipelineBarrier2(presRel.cb, &dep)
			if err := presRel.End(); err != nil {
				cb.status = cbFailed
				continue
			}
			cb.pres[presIdx].qrel = true
			continue
		}
	}
	dep := C.VkDependencyInfo{
		sType:                   C.VK_STRUCTURE_TYPE_DEPENDENCY_INFO,
		imageMemoryBarrierCount: C.uint32_t(nib),
		pImageMemoryBarriers:    pib,
	}
	C.vkCmdPipelineBarrier2(cb.cb, &dep)
	C.free(unsafe.Pointer(pib))
}

// BeginPass begins a render pass.
func (cb *cmdBuffer) BeginPass(width, height, layers int, color []driver.ColorTarget, ds *driver.DSTarget) {
	natt := len(color) + 2
	patt := (*C.VkRenderingAttachmentInfo)(C.malloc(C.sizeof_VkRenderingAttachmentInfo * C.size_t(natt)))
	satt := unsafe.Slice(patt, natt)
	var (
		pcolor   *C.VkRenderingAttachmentInfo
		pdepth   *C.VkRenderingAttachmentInfo
		pstencil *C.VkRenderingAttachmentInfo
	)
	if natt-2 > 0 {
		// Has color attachment(s).
		pcolor = patt
		for i := range color {
			var cview C.VkImageView
			if color[i].Color == nil {
				// Implementations must ignore attachments whose imageView
				// field is VK_NULL_HANDLE.
				satt[i] = C.VkRenderingAttachmentInfo{
					sType:     C.VK_STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
					imageView: cview,
				}
				continue
			}
			cview = color[i].Color.(*imageView).view
			var rview C.VkImageView
			rmode := C.VkResolveModeFlagBits(C.VK_RESOLVE_MODE_NONE)
			if color[i].Resolve != nil {
				rview = color[i].Resolve.(*imageView).view
				// NOTE: Color formats are all fp currently.
				rmode = C.VK_RESOLVE_MODE_AVERAGE_BIT
			}
			var clear C.VkClearValue
			fvalue := [4]C.float{
				C.float(color[i].Clear[0]),
				C.float(color[i].Clear[1]),
				C.float(color[i].Clear[2]),
				C.float(color[i].Clear[3]),
			}
			bclear := (*byte)(unsafe.Pointer(&fvalue[0]))
			copy(clear[:], unsafe.Slice(bclear, unsafe.Sizeof(color[i].Clear)))
			satt[i] = C.VkRenderingAttachmentInfo{
				sType:              C.VK_STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
				imageView:          cview,
				imageLayout:        C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
				resolveMode:        rmode,
				resolveImageView:   rview,
				resolveImageLayout: C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
				loadOp:             convLoadOp(color[i].Load),
				storeOp:            convStoreOp(color[i].Store),
				clearValue:         clear,
			}
		}
	}
	if ds != nil {
		// Has depth/stencil attachment.
		pdepth = &satt[natt-2]
		pstencil = &satt[natt-1]
		var dsview C.VkImageView
		*pdepth = C.VkRenderingAttachmentInfo{
			sType:              C.VK_STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
			imageView:          dsview, // VK_NULL_HANDLE
			imageLayout:        C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
			resolveImageLayout: C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
		}
		*pstencil = *pdepth
		if ds.DS != nil {
			dsview = ds.DS.(*imageView).view
			var rview C.VkImageView
			rmode := C.VkResolveModeFlagBits(C.VK_RESOLVE_MODE_NONE)
			if ds.Resolve != nil {
				rview = ds.Resolve.(*imageView).view
				// Implementations must support this mode
				// (assuming the format itself supports MS).
				rmode = C.VK_RESOLVE_MODE_SAMPLE_ZERO_BIT
			}
			var clear C.VkClearDepthStencilValue
			sclear := unsafe.Slice((*byte)(unsafe.Pointer(&clear)), unsafe.Sizeof(clear))
			aspect := ds.DS.(*imageView).subres.aspectMask
			if aspect&C.VK_IMAGE_ASPECT_DEPTH_BIT != 0 {
				pdepth.imageView = dsview
				pdepth.resolveMode = rmode
				pdepth.resolveImageView = rview
				pdepth.loadOp = convLoadOp(ds.LoadD)
				pdepth.storeOp = convStoreOp(ds.StoreD)
				clear.depth = C.float(ds.ClearD)
				copy(pdepth.clearValue[:], sclear)
			}
			if aspect&C.VK_IMAGE_ASPECT_STENCIL_BIT != 0 {
				pstencil.imageView = dsview
				pstencil.resolveMode = rmode
				pstencil.resolveImageView = rview
				pstencil.loadOp = convLoadOp(ds.LoadS)
				pstencil.storeOp = convStoreOp(ds.StoreS)
				clear.stencil = C.uint32_t(ds.ClearS)
				copy(pstencil.clearValue[:], sclear)
			}
		}
	}
	info := C.VkRenderingInfo{
		sType: C.VK_STRUCTURE_TYPE_RENDERING_INFO,
		renderArea: C.VkRect2D{
			extent: C.VkExtent2D{
				width:  C.uint32_t(width),
				height: C.uint32_t(height),
			},
		},
		layerCount:           C.uint32_t(layers),
		viewMask:             0,
		colorAttachmentCount: C.uint32_t(natt - 2),
		pColorAttachments:    pcolor,
		pDepthAttachment:     pdepth,
		pStencilAttachment:   pstencil,
	}
	C.vkCmdBeginRendering(cb.cb, &info)
	C.free(unsafe.Pointer(patt))
}

// EndPass ends the current render pass.
func (cb *cmdBuffer) EndPass() {
	C.vkCmdEndRendering(cb.cb)
}

// SetPipeline sets the pipeline.
func (cb *cmdBuffer) SetPipeline(pl driver.Pipeline) {
	pipeln := pl.(*pipeline)
	C.vkCmdBindPipeline(cb.cb, pipeln.bindp, pipeln.pl)
}

// SetViewport sets the bounds of one or more viewports.
func (cb *cmdBuffer) SetViewport(vp []driver.Viewport) {
	nvp := len(vp)
	switch {
	case nvp == 1:
		vport := C.VkViewport{
			x:        C.float(vp[0].X),
			y:        C.float(vp[0].Y),
			width:    C.float(vp[0].Width),
			height:   C.float(vp[0].Height),
			minDepth: C.float(vp[0].Znear),
			maxDepth: C.float(vp[0].Zfar),
		}
		C.vkCmdSetViewport(cb.cb, 0, 1, &vport)
	case nvp > 1:
		vport := make([]C.VkViewport, nvp)
		for i := range vport {
			vport[i] = C.VkViewport{
				x:        C.float(vp[i].X),
				y:        C.float(vp[i].Y),
				width:    C.float(vp[i].Width),
				height:   C.float(vp[i].Height),
				minDepth: C.float(vp[i].Znear),
				maxDepth: C.float(vp[i].Zfar),
			}
		}
		C.vkCmdSetViewport(cb.cb, 0, C.uint32_t(nvp), &vport[0])
	}
}

// SetScissor sets the rectangles of one or more viewport scissors.
func (cb *cmdBuffer) SetScissor(sciss []driver.Scissor) {
	nsciss := len(sciss)
	switch {
	case nsciss == 1:
		rect := C.VkRect2D{
			offset: C.VkOffset2D{
				x: C.int32_t(sciss[0].X),
				y: C.int32_t(sciss[0].Y),
			},
			extent: C.VkExtent2D{
				width:  C.uint32_t(sciss[0].Width),
				height: C.uint32_t(sciss[0].Height),
			},
		}
		C.vkCmdSetScissor(cb.cb, 0, 1, &rect)
	case nsciss > 1:
		rect := make([]C.VkRect2D, nsciss)
		for i := range rect {
			rect[i] = C.VkRect2D{
				offset: C.VkOffset2D{
					x: C.int32_t(sciss[i].X),
					y: C.int32_t(sciss[i].Y),
				},
				extent: C.VkExtent2D{
					width:  C.uint32_t(sciss[i].Width),
					height: C.uint32_t(sciss[i].Height),
				},
			}
		}
		C.vkCmdSetScissor(cb.cb, 0, C.uint32_t(nsciss), &rect[0])
	}
}

// SetBlendColor sets the constant blend color.
func (cb *cmdBuffer) SetBlendColor(r, g, b, a float32) {
	color := [4]C.float{
		C.float(r),
		C.float(g),
		C.float(b),
		C.float(a),
	}
	C.vkCmdSetBlendConstants(cb.cb, &color[0])
}

// SetStencilRef sets the stencil reference value.
func (cb *cmdBuffer) SetStencilRef(value uint32) {
	C.vkCmdSetStencilReference(cb.cb, C.VK_STENCIL_FACE_FRONT_AND_BACK, C.uint32_t(value))
}

// SetVertexBuf sets one or more vertex buffers.
func (cb *cmdBuffer) SetVertexBuf(start int, buf []driver.Buffer, off []int64) {
	nbuf := len(buf)
	switch {
	case nbuf == 1:
		buf := buf[0].(*buffer).buf
		off := C.VkDeviceSize(off[0])
		C.vkCmdBindVertexBuffers(cb.cb, C.uint32_t(start), 1, &buf, &off)
	case nbuf > 1:
		sbuf := make([]C.VkBuffer, nbuf)
		soff := make([]C.VkDeviceSize, nbuf)
		for i := range sbuf {
			sbuf[i] = buf[i].(*buffer).buf
			soff[i] = C.VkDeviceSize(off[i])
		}
		C.vkCmdBindVertexBuffers(cb.cb, C.uint32_t(start), C.uint32_t(nbuf), &sbuf[0], &soff[0])
	}
}

// SetIndexBuf sets the index buffer.
func (cb *cmdBuffer) SetIndexBuf(format driver.IndexFmt, buf driver.Buffer, off int64) {
	var typ C.VkIndexType
	switch format {
	case driver.Index16:
		typ = C.VK_INDEX_TYPE_UINT16
	case driver.Index32:
		typ = C.VK_INDEX_TYPE_UINT32
	}
	C.vkCmdBindIndexBuffer(cb.cb, buf.(*buffer).buf, C.VkDeviceSize(off), typ)
}

// SetDescTableGraph sets a descriptor table range for graphics pipelines.
func (cb *cmdBuffer) SetDescTableGraph(table driver.DescTable, start int, heapCopy []int) {
	cb.setDescTable(table, start, heapCopy, C.VK_PIPELINE_BIND_POINT_GRAPHICS)
}

// SetDescTableComp sets a descriptor table range for compute pipelines.
func (cb *cmdBuffer) SetDescTableComp(table driver.DescTable, start int, heapCopy []int) {
	cb.setDescTable(table, start, heapCopy, C.VK_PIPELINE_BIND_POINT_COMPUTE)
}

// setDescTable sets a descriptor table range for a given bind point.
func (cb *cmdBuffer) setDescTable(table driver.DescTable, start int, heapCopy []int, bindPoint C.VkPipelineBindPoint) {
	desc := table.(*descTable)
	ncpy := len(heapCopy)
	switch {
	case ncpy == 1:
		set := desc.h[start].sets[heapCopy[0]]
		C.vkCmdBindDescriptorSets(cb.cb, bindPoint, desc.layout, C.uint32_t(start), 1, &set, 0, nil)
	case ncpy > 1:
		set := make([]C.VkDescriptorSet, ncpy)
		for i := range set {
			set[i] = desc.h[start+i].sets[heapCopy[i]]
		}
		C.vkCmdBindDescriptorSets(cb.cb, bindPoint, desc.layout, C.uint32_t(start), C.uint32_t(ncpy), &set[0], 0, nil)
	}
}

// Draw draws primitives.
func (cb *cmdBuffer) Draw(vertCount, instCount, baseVert, baseInst int) {
	nvert := C.uint32_t(vertCount)
	ninst := C.uint32_t(instCount)
	bvert := C.uint32_t(baseVert)
	binst := C.uint32_t(baseInst)
	C.vkCmdDraw(cb.cb, nvert, ninst, bvert, binst)
}

// DrawIndexed draws indexed primitives.
func (cb *cmdBuffer) DrawIndexed(idxCount, instCount, baseIdx, vertOff, baseInst int) {
	nidx := C.uint32_t(idxCount)
	ninst := C.uint32_t(instCount)
	bidx := C.uint32_t(baseIdx)
	voff := C.int32_t(vertOff)
	binst := C.uint32_t(baseInst)
	C.vkCmdDrawIndexed(cb.cb, nidx, ninst, bidx, voff, binst)
}

// Dispatch dispatches compute thread groups.
func (cb *cmdBuffer) Dispatch(grpCountX, grpCountY, grpCountZ int) {
	nx := C.uint32_t(grpCountX)
	ny := C.uint32_t(grpCountY)
	nz := C.uint32_t(grpCountZ)
	C.vkCmdDispatch(cb.cb, nx, ny, nz)
}

// CopyBuffer copies data between buffers.
func (cb *cmdBuffer) CopyBuffer(param *driver.BufferCopy) {
	cpy := C.VkBufferCopy{
		srcOffset: C.VkDeviceSize(param.FromOff),
		dstOffset: C.VkDeviceSize(param.ToOff),
		size:      C.VkDeviceSize(param.Size),
	}
	C.vkCmdCopyBuffer(cb.cb, param.From.(*buffer).buf, param.To.(*buffer).buf, 1, &cpy)
}

// CopyImage copies data between images.
func (cb *cmdBuffer) CopyImage(param *driver.ImageCopy) {
	from := param.From.(*image)
	to := param.To.(*image)
	cpy := C.VkImageCopy{
		srcSubresource: C.VkImageSubresourceLayers{
			aspectMask:     from.subres.aspectMask,
			mipLevel:       C.uint32_t(param.FromLevel),
			baseArrayLayer: C.uint32_t(param.FromLayer),
			layerCount:     C.uint32_t(param.Layers),
		},
		srcOffset: C.VkOffset3D{
			x: C.int32_t(param.FromOff.X),
			y: C.int32_t(param.FromOff.Y),
			z: C.int32_t(param.FromOff.Z),
		},
		dstSubresource: C.VkImageSubresourceLayers{
			aspectMask:     to.subres.aspectMask,
			mipLevel:       C.uint32_t(param.ToLevel),
			baseArrayLayer: C.uint32_t(param.ToLayer),
			layerCount:     C.uint32_t(param.Layers),
		},
		dstOffset: C.VkOffset3D{
			x: C.int32_t(param.ToOff.X),
			y: C.int32_t(param.ToOff.Y),
			z: C.int32_t(param.ToOff.Z),
		},
		extent: C.VkExtent3D{
			width:  C.uint32_t(param.Size.Width),
			height: C.uint32_t(param.Size.Height),
			depth:  C.uint32_t(param.Size.Depth),
		},
	}
	const (
		slayout = C.VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
		dlayout = C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	)
	C.vkCmdCopyImage(cb.cb, from.img, slayout, to.img, dlayout, 1, &cpy)
}

// CopyBufToImg copies data from a buffer to an image.
func (cb *cmdBuffer) CopyBufToImg(param *driver.BufImgCopy) {
	buf := param.Buf.(*buffer)
	img := param.Img.(*image)
	var aspect C.VkImageAspectFlags
	if img.subres.aspectMask == C.VK_IMAGE_ASPECT_DEPTH_BIT|C.VK_IMAGE_ASPECT_STENCIL_BIT {
		if param.DepthCopy {
			aspect = C.VK_IMAGE_ASPECT_DEPTH_BIT
		} else {
			aspect = C.VK_IMAGE_ASPECT_STENCIL_BIT
		}
	} else {
		aspect = img.subres.aspectMask
	}
	cpy := C.VkBufferImageCopy{
		bufferOffset:      C.VkDeviceSize(param.BufOff),
		bufferRowLength:   C.uint32_t(param.Stride[0]),
		bufferImageHeight: C.uint32_t(param.Stride[1]),
		imageSubresource: C.VkImageSubresourceLayers{
			aspectMask:     aspect,
			mipLevel:       C.uint32_t(param.Level),
			baseArrayLayer: C.uint32_t(param.Layer),
			layerCount:     1,
		},
		imageOffset: C.VkOffset3D{
			x: C.int32_t(param.ImgOff.X),
			y: C.int32_t(param.ImgOff.Y),
			z: C.int32_t(param.ImgOff.Z),
		},
		imageExtent: C.VkExtent3D{
			width:  C.uint32_t(param.Size.Width),
			height: C.uint32_t(param.Size.Height),
			depth:  C.uint32_t(param.Size.Depth),
		},
	}
	const layout = C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	C.vkCmdCopyBufferToImage(cb.cb, buf.buf, img.img, layout, 1, &cpy)
}

// CopyImgToBuf copies data from an image to a buffer.
func (cb *cmdBuffer) CopyImgToBuf(param *driver.BufImgCopy) {
	img := param.Img.(*image)
	buf := param.Buf.(*buffer)
	var aspect C.VkImageAspectFlags
	if img.subres.aspectMask == C.VK_IMAGE_ASPECT_DEPTH_BIT|C.VK_IMAGE_ASPECT_STENCIL_BIT {
		if param.DepthCopy {
			aspect = C.VK_IMAGE_ASPECT_DEPTH_BIT
		} else {
			aspect = C.VK_IMAGE_ASPECT_STENCIL_BIT
		}
	} else {
		aspect = img.subres.aspectMask
	}
	cpy := C.VkBufferImageCopy{
		bufferOffset:      C.VkDeviceSize(param.BufOff),
		bufferRowLength:   C.uint32_t(param.Stride[0]),
		bufferImageHeight: C.uint32_t(param.Stride[1]),
		imageSubresource: C.VkImageSubresourceLayers{
			aspectMask:     aspect,
			mipLevel:       C.uint32_t(param.Level),
			baseArrayLayer: C.uint32_t(param.Layer),
			layerCount:     1,
		},
		imageOffset: C.VkOffset3D{
			x: C.int32_t(param.ImgOff.X),
			y: C.int32_t(param.ImgOff.Y),
			z: C.int32_t(param.ImgOff.Z),
		},
		imageExtent: C.VkExtent3D{
			width:  C.uint32_t(param.Size.Width),
			height: C.uint32_t(param.Size.Height),
			depth:  C.uint32_t(param.Size.Depth),
		},
	}
	const layout = C.VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
	C.vkCmdCopyImageToBuffer(cb.cb, img.img, layout, buf.buf, 1, &cpy)
}

// Fill fills a buffer range with copies of a byte value.
func (cb *cmdBuffer) Fill(buf driver.Buffer, off int64, value byte, size int64) {
	val := C.uint32_t(value)
	val |= val<<24 | val<<16 | val<<8
	C.vkCmdFillBuffer(cb.cb, buf.(*buffer).buf, C.VkDeviceSize(off), C.VkDeviceSize(size), val)
}

// detachSC clears any existing dependencies between the
// command buffer and swapchains.
// cb.pres is set to contain no elements.
func (cb *cmdBuffer) detachSC() {
	for i := range cb.pres {
		if cb.pres[i].wait {
			cb.pres[i].sc.pendOp[cb.pres[i].view] = false
		}
	}
	cb.pres = cb.pres[:0]
}

// Destroy destroys the command buffer.
func (cb *cmdBuffer) Destroy() {
	if cb == nil {
		return
	}
	cb.detachSC()
	if cb.d != nil {
		// TODO: Skip wait if not in pending state.
		C.vkQueueWaitIdle(cb.d.ques[cb.qfam])
		C.vkDestroyCommandPool(cb.d.dev, cb.pool, nil)
	}
	*cb = cmdBuffer{}
}

// commitInfo contains common data structures used during
// a call to the Driver.Commit method.
// It is only safe to reuse these data after the Commit
// call returns.
type commitInfo struct {
	subInfo []C.VkSubmitInfo2             // Go memory.
	cbInfo  []C.VkCommandBufferSubmitInfo // C memory.
	semInfo []C.VkSemaphoreSubmitInfo     // C memory.
}

// newCommitInfo creates new commitInfo data.
func (d *Driver) newCommitInfo() (*commitInfo, error) {
	const (
		nsub = 4
		ncb  = 4
		nsem = ncb * 2
	)
	var p unsafe.Pointer
	p = C.malloc(C.sizeof_VkCommandBufferSubmitInfo * ncb)
	cbInfo := unsafe.Slice((*C.VkCommandBufferSubmitInfo)(p), ncb)
	p = C.malloc(C.sizeof_VkSemaphoreSubmitInfo * nsem)
	semInfo := unsafe.Slice((*C.VkSemaphoreSubmitInfo)(p), nsem)
	return &commitInfo{
		subInfo: make([]C.VkSubmitInfo2, nsub),
		cbInfo:  cbInfo,
		semInfo: semInfo,
	}, nil
}

// destroyCommitInfo destroys ci.
func (d *Driver) destroyCommitInfo(ci *commitInfo) {
	if ci == nil {
		return
	}
	C.free(unsafe.Pointer(&ci.cbInfo[0]))
	C.free(unsafe.Pointer(&ci.semInfo[0]))
	*ci = commitInfo{}
}

// resizeCB resizes ci.cbInfo.
func (ci *commitInfo) resizeCB(cbInfoN int) {
	const min = 1
	if cbInfoN < min {
		cbInfoN = min
	}
	n := cap(ci.cbInfo)
	switch {
	case n < cbInfoN:
		for n < cbInfoN {
			n *= 2
		}
	case n >= 2*cbInfoN:
		n = cbInfoN
	default:
		return
	}
	p := C.realloc(unsafe.Pointer(&ci.cbInfo[0]), C.sizeof_VkCommandBufferSubmitInfo*C.size_t(n))
	ci.cbInfo = unsafe.Slice((*C.VkCommandBufferSubmitInfo)(p), n)
}

// resizeSem resizes ci.semInfo.
func (ci *commitInfo) resizeSem(semInfoN int) {
	const min = 1
	if semInfoN < min {
		semInfoN = min
	}
	n := cap(ci.semInfo)
	switch {
	case n < semInfoN:
		for n < semInfoN {
			n *= 2
		}
	case n >= 2*semInfoN:
		n = semInfoN
	default:
		return
	}
	p := C.realloc(unsafe.Pointer(&ci.semInfo[0]), C.sizeof_VkSemaphoreSubmitInfo*C.size_t(n))
	ci.semInfo = unsafe.Slice((*C.VkSemaphoreSubmitInfo)(p), n)
}

// commitSync contains common synchronization data used
// during a call to the Driver.Commit method.
// It is only safe to reuse these data after the Commit
// call writes to the provided channel.
type commitSync struct {
	fence []C.VkFence
}

// newCommitSync creates new commitSync data.
// It initializes commitSync.fence with a single fence.
func (d *Driver) newCommitSync() (*commitSync, error) {
	cs := new(commitSync)
	if err := d.resizeCommitFence(cs, 1); err != nil {
		return nil, err
	}
	return cs, nil
}

// resizeCommitFence resizes cs.fence.
// NOTE: It only increases the size currently.
func (d *Driver) resizeCommitFence(cs *commitSync, fenceN int) error {
	n := len(cs.fence)
	if n >= fenceN || fenceN < 1 {
		return nil
	}
	info := C.VkFenceCreateInfo{sType: C.VK_STRUCTURE_TYPE_FENCE_CREATE_INFO}
	var fence C.VkFence
	for i := n; i < fenceN; i++ {
		err := checkResult(C.vkCreateFence(d.dev, &info, nil, &fence))
		if err != nil {
			return err
		}
		cs.fence = append(cs.fence, fence)
	}
	return nil
}

// waitCommitFence waits for a number of cs.fence.
// fenceN must be at least 1 and no greater than len(cs.fence).
func (d *Driver) waitCommitFence(cs *commitSync, fenceN int) error {
	res := C.vkWaitForFences(d.dev, C.uint32_t(fenceN), &cs.fence[0], C.VK_TRUE, C.UINT64_MAX)
	switch res {
	case C.VK_SUCCESS:
		return nil
	default:
		switch err := checkResult(res); err {
		case nil:
			// Should never happen.
			panic("unexpected result from fence waiting")
		default:
			return err
		}
	}
}

// resetCommitFence resets a number of cs.fence.
// fenceN must be at least 1 and no greater than len(cs.fence).
func (d *Driver) resetCommitFence(cs *commitSync, fenceN int) error {
	return checkResult(C.vkResetFences(d.dev, C.uint32_t(fenceN), &cs.fence[0]))
}

// destroyCommitSync destroys cs.
func (d *Driver) destroyCommitSync(cs *commitSync) {
	if cs != nil {
		for _, fence := range cs.fence {
			C.vkDestroyFence(d.dev, fence, nil)
		}
	}
}

// Commit commits a work item to the GPU for execution.
func (d *Driver) Commit(wk *driver.WorkItem, ch chan<- *driver.WorkItem) error {
	if wk == nil || len(wk.Work) == 0 || ch == nil {
		// Client error.
		panic("invalid call to GPU.Commit")
	}
	// Take commit data from the driver an return it when
	// this call completes.
	// If too many calls to Commit were issued, we will
	// block here waiting that another call completes.
	ci := <-d.cinfo
	defer func() { d.cinfo <- ci }()
	cs := <-d.csync
	if err := d.resetCommitFence(cs, len(cs.fence)); err != nil {
		d.csync <- cs
		return err
	}
	fenceN := 1

	// Start by identifying what we will need to submit.
	type submit struct {
		cb     *cmdBuffer
		wait   []C.VkSemaphore
		signal []C.VkSemaphore
	}
	var (
		// Rendering command buffers.
		rend = make([]submit, len(wk.Work))
		// Presentation command buffers that
		// release queue ownership.
		presRel []submit
		// Presentation command buffers that
		// acquire queue ownership.
		presAcq []submit
	)
	for i := range wk.Work {
		cb := wk.Work[i].(*cmdBuffer)
		rend[i].cb = cb
		for i := range cb.pres {
			var (
				sc   = cb.pres[i].sc
				view = cb.pres[i].view
				sync = sc.viewSync[view]
			)
			if cb.pres[i].wait {
				sem := sc.nextSem[sync : sync+1]
				if cb.pres[i].qrel {
					presRel = append(presRel, submit{
						cb:     sc.queSync[sync].presRel.(*cmdBuffer),
						wait:   sem,
						signal: []C.VkSemaphore{sc.queSync[sync].rendWait},
					})
					sem = presRel[len(presRel)-1].signal
				}
				rend[i].wait = append(rend[i].wait, sem...)
			}
			if cb.pres[i].signal {
				sem := sc.presSem[view : view+1]
				if cb.pres[i].qacq {
					presAcq = append(presAcq, submit{
						cb:     sc.queSync[sync].presAcq.(*cmdBuffer),
						wait:   []C.VkSemaphore{sc.queSync[sync].presWait},
						signal: sem,
					})
					sem = presAcq[len(presAcq)-1].wait
				}
				rend[i].signal = append(rend[i].signal, sem...)
			}
		}
	}

	// TODO: Consider calculating these values in the
	// previous loop instead.
	var (
		cbInfoN  = len(rend)
		semInfoN int
	)
	for i := range rend {
		semInfoN += len(rend[i].wait) + len(rend[i].signal)
	}
	if n := len(presRel); n > 0 {
		if n > cbInfoN {
			cbInfoN = n
		}
		if 2*n > semInfoN {
			semInfoN = 2 * n
		}
	}
	if n := len(presAcq); n > 0 {
		if n > cbInfoN {
			cbInfoN = n
		}
		if 2*n > semInfoN {
			semInfoN = 2 * n
		}
	}
	ci.resizeCB(cbInfoN)
	ci.resizeSem(semInfoN)

	// Presentation queue's command buffers that release
	// ownership must be submitted first.
	if n := len(presRel); n > 0 {
		ci.subInfo = ci.subInfo[:0]
		var (
			subInfo int
			presQF  = presRel[0].cb.qfam
		)
		for i := 0; i < n; i++ {
			ci.subInfo = append(ci.subInfo, C.VkSubmitInfo2{
				sType:                    C.VK_STRUCTURE_TYPE_SUBMIT_INFO_2,
				waitSemaphoreInfoCount:   1,
				pWaitSemaphoreInfos:      &ci.semInfo[2*i],
				commandBufferInfoCount:   1,
				pCommandBufferInfos:      &ci.cbInfo[i],
				signalSemaphoreInfoCount: 1,
				pSignalSemaphoreInfos:    &ci.semInfo[2*i+1],
			})
			ci.cbInfo[i] = C.VkCommandBufferSubmitInfo{
				sType:         C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_SUBMIT_INFO,
				commandBuffer: presRel[i].cb.cb,
			}
			ci.semInfo[2*i] = C.VkSemaphoreSubmitInfo{
				sType:     C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
				semaphore: presRel[i].wait[0],
				stageMask: C.VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT,
			}
			ci.semInfo[2*i+1] = C.VkSemaphoreSubmitInfo{
				sType:     C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
				semaphore: presRel[i].signal[0],
				stageMask: C.VK_PIPELINE_STAGE_2_ALL_COMMANDS_BIT,
			}
			if i == n-1 || presQF != presRel[i+1].cb.qfam {
				var null C.VkFence
				subN := C.uint32_t(1 + i - subInfo)
				d.qmus[presQF].Lock()
				res := C.vkQueueSubmit2(d.ques[presQF], subN, &ci.subInfo[subInfo], null)
				d.qmus[presQF].Unlock()
				if err := checkResult(res); err != nil {
					d.csync <- cs
					return err
				}
				if i < n-1 {
					subInfo = i + 1
					presQF = presRel[i+1].cb.qfam
				} else {
					break
				}
			}
		}
	}
	// Rendering command buffers must be submitted after
	// all queue release and before all queue acquisition
	// operations that happen in the presentation queue.
	ci.subInfo = ci.subInfo[:0]
	var (
		cbInfo  int
		semInfo int
	)
	for i := range rend {
		var (
			waitInfoN = len(rend[i].wait)
			sigInfoN  = len(rend[i].signal)
			waitInfo  = semInfo
			sigInfo   = waitInfo + waitInfoN
		)
		ci.subInfo = append(ci.subInfo, C.VkSubmitInfo2{
			sType:                    C.VK_STRUCTURE_TYPE_SUBMIT_INFO_2,
			waitSemaphoreInfoCount:   C.uint32_t(waitInfoN),
			commandBufferInfoCount:   1,
			pCommandBufferInfos:      &ci.cbInfo[cbInfo],
			signalSemaphoreInfoCount: C.uint32_t(sigInfoN),
		})
		ci.cbInfo[cbInfo] = C.VkCommandBufferSubmitInfo{
			sType:         C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_SUBMIT_INFO,
			commandBuffer: rend[i].cb.cb,
		}
		if waitInfoN > 0 {
			ci.subInfo[len(ci.subInfo)-1].pWaitSemaphoreInfos = &ci.semInfo[waitInfo]
			for j := range rend[i].wait {
				ci.semInfo[waitInfo] = C.VkSemaphoreSubmitInfo{
					sType:     C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
					semaphore: rend[i].wait[j],
					stageMask: C.VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT,
				}
				waitInfo++
			}
		}
		if sigInfoN > 0 {
			ci.subInfo[len(ci.subInfo)-1].pSignalSemaphoreInfos = &ci.semInfo[sigInfo]
			for j := range rend[i].signal {
				ci.semInfo[sigInfo] = C.VkSemaphoreSubmitInfo{
					sType:     C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
					semaphore: rend[i].signal[j],
					stageMask: C.VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT,
				}
				sigInfo++
			}
		}
		semInfo = sigInfo
	}
	if n := len(presAcq); n == 0 {
		d.qmus[d.qfam].Lock()
		res := C.vkQueueSubmit2(d.ques[d.qfam], C.uint32_t(len(rend)), &ci.subInfo[0], cs.fence[0])
		d.qmus[d.qfam].Unlock()
		if err := checkResult(res); err != nil {
			d.csync <- cs
			return err
		}
	} else {
		d.qmus[d.qfam].Lock()
		res := C.vkQueueSubmit2(d.ques[d.qfam], C.uint32_t(len(rend)), &ci.subInfo[0], cs.fence[0])
		d.qmus[d.qfam].Unlock()
		if err := checkResult(res); err != nil {
			d.csync <- cs
			return err
		}
		// Presentation queue's command buffers that acquire
		// ownership must be submitted last.
		ci.subInfo = ci.subInfo[:0]
		var (
			subInfo int
			presQF  = presAcq[0].cb.qfam
		)
		for i := 0; i < n; i++ {
			ci.subInfo = append(ci.subInfo, C.VkSubmitInfo2{
				sType:                    C.VK_STRUCTURE_TYPE_SUBMIT_INFO_2,
				waitSemaphoreInfoCount:   1,
				pWaitSemaphoreInfos:      &ci.semInfo[2*i],
				commandBufferInfoCount:   1,
				pCommandBufferInfos:      &ci.cbInfo[i],
				signalSemaphoreInfoCount: 1,
				pSignalSemaphoreInfos:    &ci.semInfo[2*i+1],
			})
			ci.cbInfo[i] = C.VkCommandBufferSubmitInfo{
				sType:         C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_SUBMIT_INFO,
				commandBuffer: presAcq[i].cb.cb,
			}
			ci.semInfo[2*i] = C.VkSemaphoreSubmitInfo{
				sType:     C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
				semaphore: presAcq[i].wait[0],
				stageMask: C.VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT,
			}
			ci.semInfo[2*i+1] = C.VkSemaphoreSubmitInfo{
				sType:     C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
				semaphore: presAcq[i].signal[0],
				stageMask: C.VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT,
			}
			if i == n-1 || presQF != presAcq[i+1].cb.qfam {
				if err := d.resizeCommitFence(cs, fenceN+1); err != nil {
					d.waitCommitFence(cs, fenceN)
					d.csync <- cs
					return err
				}
				subN := C.uint32_t(1 + i - subInfo)
				d.qmus[presQF].Lock()
				res = C.vkQueueSubmit2(d.ques[presQF], subN, &ci.subInfo[subInfo], cs.fence[fenceN])
				d.qmus[presQF].Unlock()
				if err := checkResult(res); err != nil {
					d.waitCommitFence(cs, fenceN)
					d.csync <- cs
					return err
				}
				fenceN++
				if i < n-1 {
					subInfo = i + 1
					presQF = presAcq[i+1].cb.qfam
				} else {
					break
				}
			}
		}
	}

	// Wait in the background for queue submissions to
	// complete execution.
	for i := range rend {
		rend[i].cb.status = cbCommitted
		rend[i].cb.detachSC()
	}
	go func() {
		wk.Err = d.waitCommitFence(cs, fenceN)
		for i := range rend {
			rend[i].cb.status = cbIdle
		}
		ch <- wk
		d.csync <- cs
	}()
	return nil
}

// convSync converts a driver.Sync to a VkPipelineStageFlags2.
func convSync(sync driver.Sync) C.VkPipelineStageFlags2 {
	if sync == driver.SNone {
		return C.VK_PIPELINE_STAGE_2_NONE // 0
	}
	if sync&driver.SAll != 0 {
		return C.VK_PIPELINE_STAGE_2_ALL_COMMANDS_BIT
	}

	var flags C.VkPipelineStageFlags2
	if sync&driver.SGraphics != 0 {
		flags |= C.VK_PIPELINE_STAGE_2_ALL_GRAPHICS_BIT
	} else {
		if sync&driver.SVertexInput != 0 {
			flags |= C.VK_PIPELINE_STAGE_2_VERTEX_INPUT_BIT
		}
		if sync&driver.SVertexShading != 0 {
			flags |= C.VK_PIPELINE_STAGE_2_PRE_RASTERIZATION_SHADERS_BIT
		}
		if sync&driver.SFragmentShading != 0 {
			flags |= C.VK_PIPELINE_STAGE_2_FRAGMENT_SHADER_BIT
		}
		if sync&driver.SDSOutput != 0 {
			flags |= C.VK_PIPELINE_STAGE_2_EARLY_FRAGMENT_TESTS_BIT
			flags |= C.VK_PIPELINE_STAGE_2_LATE_FRAGMENT_TESTS_BIT
		}
		if sync&driver.SColorOutput != 0 {
			flags |= C.VK_PIPELINE_STAGE_2_COLOR_ATTACHMENT_OUTPUT_BIT
		}
	}
	if sync&driver.SComputeShading != 0 {
		flags |= C.VK_PIPELINE_STAGE_2_COMPUTE_SHADER_BIT
	}
	if sync&driver.SCopy != 0 {
		flags |= C.VK_PIPELINE_STAGE_2_TRANSFER_BIT
	}
	return flags
}

// convAccess converts a driver.Access to a VkAccessFlags2.
func convAccess(acc driver.Access) C.VkAccessFlags2 {
	if acc == driver.ANone {
		return C.VK_ACCESS_2_NONE // 0
	}

	var flags C.VkAccessFlags2
	if acc&driver.AAnyRead != 0 {
		flags |= C.VK_ACCESS_2_MEMORY_READ_BIT
	} else {
		if acc&driver.AVertexBufRead != 0 {
			flags |= C.VK_ACCESS_2_VERTEX_ATTRIBUTE_READ_BIT
		}
		if acc&driver.AIndexBufRead != 0 {
			flags |= C.VK_ACCESS_2_INDEX_READ_BIT
		}
		if acc&driver.AShaderRead != 0 {
			flags |= C.VK_ACCESS_2_SHADER_READ_BIT
		}
		if acc&driver.AColorRead != 0 {
			flags |= C.VK_ACCESS_2_COLOR_ATTACHMENT_READ_BIT
		}
		if acc&driver.ADSRead != 0 {
			flags |= C.VK_ACCESS_2_DEPTH_STENCIL_ATTACHMENT_READ_BIT
		}
		if acc&driver.ACopyRead != 0 {
			flags |= C.VK_ACCESS_2_TRANSFER_READ_BIT
		}
	}

	if acc&driver.AAnyWrite != 0 {
		flags |= C.VK_ACCESS_2_MEMORY_WRITE_BIT
	} else {
		if acc&driver.AShaderWrite != 0 {
			flags |= C.VK_ACCESS_2_SHADER_WRITE_BIT
		}
		if acc&driver.AColorWrite != 0 {
			flags |= C.VK_ACCESS_2_COLOR_ATTACHMENT_WRITE_BIT
		}
		if acc&driver.ADSWrite != 0 {
			flags |= C.VK_ACCESS_2_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT
		}
		if acc&driver.ACopyWrite != 0 {
			flags |= C.VK_ACCESS_2_TRANSFER_WRITE_BIT
		}
	}
	return flags
}

// convLayout converts a driver.Layout to a VkImageLayout.
func convLayout(lay driver.Layout) C.VkImageLayout {
	switch lay {
	case driver.LUndefined:
		return C.VK_IMAGE_LAYOUT_UNDEFINED
	case driver.LShaderRead:
		return C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL
	case driver.LColorTarget:
		return C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	case driver.LDSTarget:
		return C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
	case driver.LDSRead:
		return C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_READ_ONLY_OPTIMAL
	case driver.LCopySrc:
		return C.VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
	case driver.LCopyDst:
		return C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	case driver.LPresent:
		return C.VK_IMAGE_LAYOUT_PRESENT_SRC_KHR
	}

	// Expected to be unreachable.
	return ^C.VkImageLayout(0)
}

// convLoadOp converts a driver.LoadOp to a VkAttachmentLoadOp.
func convLoadOp(op driver.LoadOp) C.VkAttachmentLoadOp {
	switch op {
	case driver.LDontCare:
		return C.VK_ATTACHMENT_LOAD_OP_DONT_CARE
	case driver.LClear:
		return C.VK_ATTACHMENT_LOAD_OP_CLEAR
	case driver.LLoad:
		return C.VK_ATTACHMENT_LOAD_OP_LOAD
	}

	// Expected to be unreachable.
	return ^C.VkAttachmentLoadOp(0)
}

// convStoreOp converts a driver.StoreOp to a VkAttachmentStoreOp.
func convStoreOp(op driver.StoreOp) C.VkAttachmentStoreOp {
	switch op {
	case driver.SDontCare:
		return C.VK_ATTACHMENT_STORE_OP_DONT_CARE
	case driver.SStore:
		return C.VK_ATTACHMENT_STORE_OP_STORE
	}

	// Expected to be unreachable.
	return ^C.VkAttachmentStoreOp(0)
}
