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
	d     *Driver
	qfam  C.uint32_t
	pool  C.VkCommandPool
	cb    C.VkCommandBuffer
	begun bool

	// When set, sc indicates that there is
	// an ongoing presentation operation
	// that will target scView.
	// scNext and scPres are used to track
	// which methods of sc were called.
	sc     *swapchain
	scView int
	scNext bool
	scPres bool
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
	if !cb.begun {
		info := C.VkCommandBufferBeginInfo{
			sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
			flags: C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
		}
		err := checkResult(C.vkBeginCommandBuffer(cb.cb, &info))
		if err != nil {
			return err
		}
		cb.begun = true
	}
	return nil
}

// End ends command recording and prepares the command buffer for execution.
func (cb *cmdBuffer) End() error {
	if cb.begun {
		cb.begun = false
		return checkResult(C.vkEndCommandBuffer(cb.cb))
	}
	return nil
}

// Reset discards all recorded commands from the command buffer.
func (cb *cmdBuffer) Reset() error {
	err := checkResult(C.vkResetCommandBuffer(cb.cb, 0))
	if err != nil {
		return err
	}
	cb.begun = false
	return nil
}

// Barrier inserts a number of global barriers in the command buffer.
func (cb *cmdBuffer) Barrier(b []driver.Barrier) {
	// TODO: Use new synchronization barrier (1.3/synchronization2).
	mb := C.VkMemoryBarrier{
		sType: C.VK_STRUCTURE_TYPE_MEMORY_BARRIER,
	}
	for i := range b {
		sstg := convSync(b[i].SyncBefore)
		dstg := convSync(b[i].SyncAfter)
		mb.srcAccessMask = convAccess(b[i].AccessBefore)
		mb.dstAccessMask = convAccess(b[i].AccessAfter)
		C.vkCmdPipelineBarrier(cb.cb, sstg, dstg, 0, 1, &mb, 0, nil, 0, nil)
	}
}

// Transition inserts a number of image layout transitions in the
// command buffer.
func (cb *cmdBuffer) Transition(t []driver.Transition) {
	// TODO: Use new synchronization barrier (1.3/synchronization2).
	imb := C.VkImageMemoryBarrier{
		sType: C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
	}
	for i := range t {
		sstg := convSync(t[i].SyncBefore)
		dstg := convSync(t[i].SyncAfter)
		imb.srcAccessMask = convAccess(t[i].AccessBefore)
		imb.dstAccessMask = convAccess(t[i].AccessAfter)
		imb.oldLayout = convLayout(t[i].LayoutBefore)
		imb.newLayout = convLayout(t[i].LayoutAfter)
		view := t[i].IView.(*imageView)
		if view.i != nil {
			imb.image = view.i.img
		} else {
			for i := range view.s.views {
				if view.s.views[i] == view {
					imb.image = view.s.imgs[i]
					break
				}
			}
		}
		imb.subresourceRange = view.subres
		C.vkCmdPipelineBarrier(cb.cb, sstg, dstg, 0, 0, nil, 0, nil, 1, &imb)
	}
}

// scBarrier records an image memory barrier in the command buffer.
// The image is taken from cb.sc.imgs[cb.scView].
// It assumes that all images in the swapchain have a single layer.
func (cb *cmdBuffer) scBarrier(lay1, lay2 C.VkImageLayout, que1, que2 C.uint32_t, stg1, stg2 C.VkPipelineStageFlags, acc1, acc2 C.VkAccessFlags) {
	imb := C.VkImageMemoryBarrier{
		sType:               C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		srcAccessMask:       acc1,
		dstAccessMask:       acc2,
		oldLayout:           lay1,
		newLayout:           lay2,
		srcQueueFamilyIndex: que1,
		dstQueueFamilyIndex: que2,
		image:               cb.sc.imgs[cb.scView],
		subresourceRange: C.VkImageSubresourceRange{
			aspectMask: C.VK_IMAGE_ASPECT_COLOR_BIT,
			levelCount: 1,
			layerCount: 1,
		},
	}
	C.vkCmdPipelineBarrier(cb.cb, stg1, stg2, 0, 0, nil, 0, nil, 1, &imb)
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
	// TODO: Ensure images have transitioned to the correct layout.
	layout := C.VkImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
	C.vkCmdCopyImage(cb.cb, from.img, layout, to.img, layout, 1, &cpy)
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
	// TODO: Ensure image has transitioned to the correct layout.
	layout := C.VkImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
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
	// TODO: Ensure image has transitioned to the correct layout.
	layout := C.VkImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
	C.vkCmdCopyImageToBuffer(cb.cb, img.img, layout, buf.buf, 1, &cpy)
}

// Fill fills a buffer range with copies of a byte value.
func (cb *cmdBuffer) Fill(buf driver.Buffer, off int64, value byte, size int64) {
	val := C.uint32_t(value)
	val |= val<<24 | val<<16 | val<<8
	C.vkCmdFillBuffer(cb.cb, buf.(*buffer).buf, C.VkDeviceSize(off), C.VkDeviceSize(size), val)
}

// Destroy destroys the command buffer.
func (cb *cmdBuffer) Destroy() {
	if cb == nil {
		return
	}
	if cb.d != nil {
		// TODO: Skip wait if not in pending state.
		C.vkQueueWaitIdle(cb.d.ques[cb.qfam])
		C.vkDestroyCommandPool(cb.d.dev, cb.pool, nil)
	}
	*cb = cmdBuffer{}
}

// commitData contains common data used during a call to the
// Driver.Commit method.
// It is only safe to reuse the data after the Commit call
// writes to the provided channel.
type commitData struct {
	fence C.VkFence
	cb    []C.VkCommandBuffer      // C memory.
	sem   []C.VkSemaphore          // C memory.
	stg   []C.VkPipelineStageFlags // C memory.
}

// newCommitData creates new commit data.
func (d *Driver) newCommitData() (*commitData, error) {
	info := C.VkFenceCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
	}
	var fence C.VkFence
	err := checkResult(C.vkCreateFence(d.dev, &info, nil, &fence))
	if err != nil {
		return nil, err
	}
	const (
		ncb  = 4
		nsem = 3
		nstg = 1
	)
	var p unsafe.Pointer
	p = C.malloc(C.sizeof_VkCommandBuffer * ncb)
	cb := unsafe.Slice((*C.VkCommandBuffer)(p), ncb)
	p = C.malloc(C.sizeof_VkSemaphore * nsem)
	sem := unsafe.Slice((*C.VkSemaphore)(p), nsem)
	p = C.malloc(C.sizeof_VkPipelineStageFlags * nstg)
	stg := unsafe.Slice((*C.VkPipelineStageFlags)(p), nstg)
	return &commitData{
		fence: fence,
		cb:    cb,
		sem:   sem,
		stg:   stg,
	}, nil
}

// destroyCommitData destroys commit data.
func (d *Driver) destroyCommitData(cd *commitData) {
	if cd == nil {
		return
	}
	C.vkDestroyFence(d.dev, cd.fence, nil)
	C.free(unsafe.Pointer(&cd.cb[0]))
	C.free(unsafe.Pointer(&cd.sem[0]))
	C.free(unsafe.Pointer(&cd.stg[0]))
	*cd = commitData{}
}

// resizeCB resizes cd.cb.
func (cd *commitData) resizeCB(min int) {
	n := len(cd.cb)
	switch {
	case n < min:
		for n < min {
			n *= 2
		}
	case n >= 2*min:
		n = min
	default:
		return
	}
	p := C.realloc(unsafe.Pointer(&cd.cb[0]), C.sizeof_VkCommandBuffer*C.size_t(n))
	cd.cb = unsafe.Slice((*C.VkCommandBuffer)(p), n)
}

// Commit commits a batch of command buffers to the GPU for execution.
//
// TODO: Allow multiple presentation requests per commit and split
// into multiple submit infos to avoid stalling on semaphores.
func (d *Driver) Commit(cb []driver.CmdBuffer, ch chan<- error) {
	if len(cb) == 0 {
		ch <- nil
		return
	}
	// Take commit data from the driver an return it when
	// this call completes.
	// If too many calls to Commit were issued, we will
	// block here waiting for data to become available.
	cd := <-d.cdata
	defer func() { d.cdata <- cd }()
	err := checkResult(C.vkResetFences(d.dev, 1, &cd.fence))
	if err != nil {
		ch <- err
		return
	}
	cd.resizeCB(len(cb))
	for i := range cb {
		cd.cb[i] = cb[i].(*cmdBuffer).cb
	}
	info := C.VkSubmitInfo{
		sType:              C.VK_STRUCTURE_TYPE_SUBMIT_INFO,
		commandBufferCount: C.uint32_t(len(cb)),
		pCommandBuffers:    &cd.cb[0],
	}

	// Command buffers that have a pending present request
	// will contain a non-nil swapchain.
	// Note that the swapchain's Next and Present methods
	// need not be called on the same command buffer.
	var pres [2]*cmdBuffer
	for i := range cb {
		if c := cb[i].(*cmdBuffer); c.sc != nil {
			if c.scNext {
				pres[0] = c
			}
			if c.scPres {
				pres[1] = c
				break
			}
		}
	}
	release := func() {}
	switch {
	case pres[0] == nil && pres[1] == nil:
		// There is nothing to present.
		d.qchan <- 1
		res := C.vkQueueSubmit(d.ques[d.qfam], 1, &info, cd.fence)
		<-d.qchan
		if err = checkResult(res); err != nil {
			ch <- err
			return
		}
	case pres[0] != nil && pres[1] != nil:
		// There is a pending present request.
		// We assume that pres[0].sc and pres[1].sc refer
		// to the same swapchain.
		sc := pres[0].sc
		iv := pres[0].scView
		sync := sc.viewSync[iv]
		strd := 1 + len(sc.views) - sc.minImg
		cd.sem[0] = sc.sems[sync]
		cd.stg[0] = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT
		info.waitSemaphoreCount = 1
		info.pWaitSemaphores = &cd.sem[0]
		info.pWaitDstStageMask = &cd.stg[0]
		info.signalSemaphoreCount = 1
		info.pSignalSemaphores = &cd.sem[1]
		var presSem *C.VkSemaphore
		d.qchan <- 1
		if sc.qfam == d.qfam {
			cd.sem[1] = sc.sems[strd+iv]
			res := C.vkQueueSubmit(d.ques[d.qfam], 1, &info, cd.fence)
			if err = checkResult(res); err != nil {
				// We need to release the swapchain's image.
				// Presenting it is trick because its state in unknown,
				// so we take the easy way out and mark the swapchain
				// as broken instead. Further uses of the swapchain
				// will fail and the caller will have to either
				// recreate or destroy it.
				sc.broken = true
				<-d.qchan
				ch <- err
				return
			}
			presSem = &cd.sem[1]
		} else {
			var null C.VkFence
			cd.sem[1] = sc.sems[sync+strd]
			res := C.vkQueueSubmit(d.ques[d.qfam], 1, &info, null)
			if err = checkResult(res); err != nil {
				sc.broken = true
				<-d.qchan
				ch <- err
				return
			}
			cd.cb[0] = sc.pcbs[sync].(*cmdBuffer).cb
			cd.sem[2] = sc.sems[strd*2+iv]
			cd.stg[0] = C.VK_PIPELINE_STAGE_ALL_COMMANDS_BIT
			info.commandBufferCount = 1
			info.pWaitSemaphores = &cd.sem[1]
			info.pSignalSemaphores = &cd.sem[2]
			res = C.vkQueueSubmit(d.ques[sc.qfam], 1, &info, cd.fence)
			if err = checkResult(res); err != nil {
				sc.broken = true
				C.vkQueueWaitIdle(d.ques[d.qfam])
				<-d.qchan
				ch <- err
				return
			}
			presSem = &cd.sem[2]
		}
		// Ignore presentation error.
		sc.present(iv, presSem)
		<-d.qchan
		pres[0].sc = nil
		pres[1].sc = nil
		release = func() {
			sc.mu.Lock()
			sc.curImg--
			sc.syncUsed[sync] = false
			sc.mu.Unlock()
		}
	default:
		panic("corrupted command buffer presentation")
	}

	// Wait until queue submission has completed execution.
	// Note that queue presentation is not waited for, and as such
	// may not have completed when we send to ch.
	res := C.vkWaitForFences(d.dev, 1, &cd.fence, C.VK_TRUE, C.UINT64_MAX)
	release()
	switch res {
	case C.VK_SUCCESS:
		ch <- nil
	default:
		switch err := checkResult(res); err {
		case nil:
			// Should never happen.
			panic("unexpected result from fence wait")
		default:
			ch <- err
		}
	}
}

// convSync converts a driver.Sync to a VkPipelineStageFlags.
func convSync(sync driver.Sync) C.VkPipelineStageFlags {
	if sync == driver.SNone {
		return C.VK_PIPELINE_STAGE_NONE // 0
	}
	if sync&driver.SAll != 0 {
		return C.VK_PIPELINE_STAGE_ALL_COMMANDS_BIT
	}

	var flags C.VkPipelineStageFlags
	if sync&driver.SDraw != 0 {
		flags |= C.VK_PIPELINE_STAGE_ALL_GRAPHICS_BIT
	} else {
		if sync&driver.SVertexInput != 0 {
			flags |= C.VK_PIPELINE_STAGE_VERTEX_INPUT_BIT
		}
		if sync&driver.SVertexShading != 0 {
			// NOTE: This is the latest currently supported.
			flags |= C.VK_PIPELINE_STAGE_VERTEX_SHADER_BIT
		}
		if sync&driver.SFragmentShading != 0 {
			flags |= C.VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT
		}
		if sync&driver.SColorOutput != 0 {
			flags |= C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT
		}
		if sync&driver.SDSOutput != 0 {
			flags |= C.VK_PIPELINE_STAGE_LATE_FRAGMENT_TESTS_BIT
		}
	}
	if sync&driver.SComputeShading != 0 {
		flags |= C.VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT
	}
	if sync&(driver.SResolve|driver.SCopy) != 0 {
		flags |= C.VK_PIPELINE_STAGE_TRANSFER_BIT
	}
	return flags
}

// convAccess converts a driver.Access to a VkAccessFlags.
func convAccess(acc driver.Access) C.VkAccessFlags {
	if acc == driver.ANone {
		return C.VK_ACCESS_NONE // 0
	}

	var flags C.VkAccessFlags
	if acc&driver.AAnyRead != 0 {
		flags |= C.VK_ACCESS_MEMORY_READ_BIT
	} else {
		if acc&driver.AVertexBufRead != 0 {
			flags |= C.VK_ACCESS_VERTEX_ATTRIBUTE_READ_BIT
		}
		if acc&driver.AIndexBufRead != 0 {
			flags |= C.VK_ACCESS_INDEX_READ_BIT
		}
		if acc&driver.AColorRead != 0 {
			flags |= C.VK_ACCESS_COLOR_ATTACHMENT_READ_BIT
		}
		if acc&driver.ADSRead != 0 {
			flags |= C.VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_READ_BIT
		}
		if acc&(driver.AResolveRead|driver.ACopyRead) != 0 {
			flags |= C.VK_ACCESS_TRANSFER_READ_BIT
		}
		if acc&driver.AShaderRead != 0 {
			flags |= C.VK_ACCESS_SHADER_READ_BIT
		}
	}

	if acc&driver.AAnyWrite != 0 {
		flags |= C.VK_ACCESS_MEMORY_WRITE_BIT
	} else {
		if acc&driver.AColorWrite != 0 {
			flags |= C.VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT
		}
		if acc&driver.ADSWrite != 0 {
			flags |= C.VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT
		}
		if acc&(driver.AResolveWrite|driver.ACopyWrite) != 0 {
			flags |= C.VK_ACCESS_TRANSFER_WRITE_BIT
		}
		if acc&driver.AShaderWrite != 0 {
			flags |= C.VK_ACCESS_SHADER_WRITE_BIT
		}
	}
	return flags
}

// convLayout converts a driver.Layout to a VkImageLayout.
func convLayout(lay driver.Layout) C.VkImageLayout {
	switch lay {
	case driver.LUndefined:
		return C.VK_IMAGE_LAYOUT_UNDEFINED
	case driver.LCommon:
		return C.VK_IMAGE_LAYOUT_GENERAL
	case driver.LColorTarget:
		return C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	case driver.LDSTarget:
		return C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
	case driver.LDSRead:
		return C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_READ_ONLY_OPTIMAL
	case driver.LResolveSrc, driver.LCopySrc:
		return C.VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
	case driver.LResolveDst, driver.LCopyDst:
		return C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	case driver.LShaderRead:
		return C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL
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
