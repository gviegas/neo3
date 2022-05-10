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

// Record records commands in the command buffer.
func (cb *cmdBuffer) Record(cmd []driver.Command, cache *driver.CmdCache) error {
	if err := cb.begin(); err != nil {
		return err
	}
	for i := range cmd {
		switch cmd[i].Type {
		case driver.CPass:
			cb.passCmd(cmd[i].Index, cache)
		case driver.CWork:
			cb.workCmd(cmd[i].Index, cache)
		case driver.CBlit:
			cb.blitCmd(cmd[i].Index, cache)
		}
	}
	return nil
}

// begin puts the command buffer in the recording state.
func (cb *cmdBuffer) begin() error {
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

// end puts the command buffer in the executable state.
func (cb *cmdBuffer) end() error {
	if cb.begun {
		cb.begun = false
		return checkResult(C.vkEndCommandBuffer(cb.cb))
	}
	return nil
}

// reset puts the command buffer in the initial state.
func (cb *cmdBuffer) reset() error {
	err := checkResult(C.vkResetCommandBuffer(cb.cb, 0))
	if err != nil {
		return err
	}
	cb.begun = false
	return nil
}

// barrier records a memory barrier in the command buffer.
func (cb *cmdBuffer) barrier(stg1, stg2 C.VkPipelineStageFlags, acc1, acc2 C.VkAccessFlags) {
	mb := C.VkMemoryBarrier{
		sType:         C.VK_STRUCTURE_TYPE_MEMORY_BARRIER,
		srcAccessMask: acc1,
		dstAccessMask: acc2,
	}
	C.vkCmdPipelineBarrier(cb.cb, stg1, stg2, 0, 1, &mb, 0, nil, 0, nil)
}

// transition records an image layout transitiion in the command buffer.
func (cb *cmdBuffer) transition(img *image, to C.VkImageLayout, stg1, stg2 C.VkPipelineStageFlags, acc1, acc2 C.VkAccessFlags) {
	if img.layout == to {
		return
	}
	imb := C.VkImageMemoryBarrier{
		sType:            C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		srcAccessMask:    acc1,
		dstAccessMask:    acc2,
		oldLayout:        img.layout,
		newLayout:        to,
		image:            img.img,
		subresourceRange: img.subres,
	}
	img.layout = to
	C.vkCmdPipelineBarrier(cb.cb, stg1, stg2, 0, 0, nil, 0, nil, 1, &imb)
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

// passCmd records a driver.PassCmd in the command buffer.
func (cb *cmdBuffer) passCmd(index int, cache *driver.CmdCache) {
	cmd := &cache.Pass[index]
	pass := cmd.Pass.(*renderPass)
	fb := cmd.FB.(*framebuf)
	var pclr *C.VkClearValue
	nclr := len(cmd.Clear)
	if nclr > 0 {
		pclr = (*C.VkClearValue)(C.malloc(C.size_t(nclr) * C.sizeof_VkClearValue))
		defer C.free(unsafe.Pointer(pclr))
		sclr := unsafe.Slice(pclr, nclr)
		for i := range sclr {
			if pass.aspect[i] == C.VK_IMAGE_ASPECT_COLOR_BIT {
				color := [4]C.float{
					C.float(cmd.Clear[i].Color[0]),
					C.float(cmd.Clear[i].Color[1]),
					C.float(cmd.Clear[i].Color[2]),
					C.float(cmd.Clear[i].Color[3]),
				}
				raw := (*byte)(unsafe.Pointer(&color[0]))
				copy(sclr[i][:], unsafe.Slice(raw, unsafe.Sizeof(color)))
			} else {
				ds := C.VkClearDepthStencilValue{
					depth:   C.float(cmd.Clear[i].Depth),
					stencil: C.uint32_t(cmd.Clear[i].Stencil),
				}
				raw := (*byte)(unsafe.Pointer(&ds))
				copy(sclr[i][:], unsafe.Slice(raw, unsafe.Sizeof(ds)))
			}
		}
	}
	info := C.VkRenderPassBeginInfo{
		sType:       C.VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
		renderPass:  pass.pass,
		framebuffer: fb.fb,
		renderArea: C.VkRect2D{
			extent: C.VkExtent2D{
				width:  C.uint32_t(fb.width),
				height: C.uint32_t(fb.height),
			},
		},
		clearValueCount: C.uint32_t(nclr),
		pClearValues:    pclr,
	}
	C.vkCmdBeginRenderPass(cb.cb, &info, C.VK_SUBPASS_CONTENTS_INLINE)
	cb.subpassCmd(cmd.SubCmd[0], cache)
	for i := 1; i < len(cmd.SubCmd); i++ {
		C.vkCmdNextSubpass(cb.cb, C.VK_SUBPASS_CONTENTS_INLINE)
		cb.subpassCmd(cmd.SubCmd[i], cache)
	}
	C.vkCmdEndRenderPass(cb.cb)
}

// workCmd records a driver.WorkCmd in the command buffer.
func (cb *cmdBuffer) workCmd(index int, cache *driver.CmdCache) {
	if cache.Work[index].Wait {
		// TODO: Improve this.
		stg1 := C.VkPipelineStageFlags(C.VK_PIPELINE_STAGE_ALL_COMMANDS_BIT)
		stg2 := C.VkPipelineStageFlags(C.VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT)
		acc1 := C.VkAccessFlags(C.VK_ACCESS_MEMORY_WRITE_BIT)
		acc2 := C.VkAccessFlags(C.VK_ACCESS_SHADER_WRITE_BIT | C.VK_ACCESS_SHADER_READ_BIT)
		cb.barrier(stg1, stg2, acc1, acc2)
	}
	for _, c := range cache.Work[index].Cmd {
		switch c.Type {
		case driver.CPipeline:
			cb.pipelineCmd(cache.Pipeline[c.Index], C.VK_PIPELINE_BIND_POINT_COMPUTE)
		case driver.CDescTable:
			cb.descTableCmd(cache.DescTable[c.Index], C.VK_PIPELINE_BIND_POINT_COMPUTE)
		case driver.CDispatch:
			cb.dispatchCmd(cache.Dispatch[c.Index])
		}
	}
}

// blitCmd records a driver.BlitCmd in the command buffer.
func (cb *cmdBuffer) blitCmd(index int, cache *driver.CmdCache) {
	if cache.Blit[index].Wait {
		// TODO: Improve this.
		stg1 := C.VkPipelineStageFlags(C.VK_PIPELINE_STAGE_ALL_COMMANDS_BIT)
		stg2 := C.VkPipelineStageFlags(C.VK_PIPELINE_STAGE_TRANSFER_BIT)
		acc1 := C.VkAccessFlags(C.VK_ACCESS_MEMORY_WRITE_BIT)
		acc2 := C.VkAccessFlags(C.VK_ACCESS_TRANSFER_WRITE_BIT | C.VK_ACCESS_TRANSFER_READ_BIT)
		cb.barrier(stg1, stg2, acc1, acc2)
	}
	for _, c := range cache.Blit[index].Cmd {
		switch c.Type {
		case driver.CBufferCopy:
			cb.bufferCopyCmd(cache.BufferCopy[c.Index])
		case driver.CImageCopy:
			cb.imageCopyCmd(&cache.ImageCopy[c.Index])
		case driver.CBufImgCopy:
			cb.bufImgCopyCmd(&cache.BufImgCopy[c.Index])
		case driver.CImgBufCopy:
			cb.imgBufCopyCmd(&cache.ImgBufCopy[c.Index])
		case driver.CFill:
			cb.fillCmd(cache.Fill[c.Index])
		}
	}
}

// subpassCmd records a driver.SubpassCmd in the command buffer.
func (cb *cmdBuffer) subpassCmd(index int, cache *driver.CmdCache) {
	for _, c := range cache.Subpass[index].Cmd {
		switch c.Type {
		case driver.CPipeline:
			cb.pipelineCmd(cache.Pipeline[c.Index], C.VK_PIPELINE_BIND_POINT_GRAPHICS)
		case driver.CViewport:
			cb.viewportCmd(cache.Viewport[c.Index])
		case driver.CScissor:
			cb.scissorCmd(cache.Scissor[c.Index])
		case driver.CBlendColor:
			cb.blendColorCmd(cache.BlendColor[c.Index])
		case driver.CStencilRef:
			cb.stencilRefCmd(cache.StencilRef[c.Index])
		case driver.CVertexBuf:
			cb.vertexBufCmd(cache.VertexBuf[c.Index])
		case driver.CIndexBuf:
			cb.indexBufCmd(cache.IndexBuf[c.Index])
		case driver.CDescTable:
			cb.descTableCmd(cache.DescTable[c.Index], C.VK_PIPELINE_BIND_POINT_GRAPHICS)
		case driver.CDraw:
			cb.drawCmd(cache.Draw[c.Index])
		case driver.CIndexDraw:
			cb.indexDrawCmd(cache.IndexDraw[c.Index])
		}
	}
}

// pipelineCmd records a driver.PipelineCmd in the command buffer.
func (cb *cmdBuffer) pipelineCmd(cmd driver.PipelineCmd, bp C.VkPipelineBindPoint) {
	pl := cmd.PL.(*pipeline)
	C.vkCmdBindPipeline(cb.cb, bp, pl.pl)
}

// viewportCmd records a driver.ViewportCmd in the command buffer.
func (cb *cmdBuffer) viewportCmd(cmd driver.ViewportCmd) {
	nvp := len(cmd.VP)
	switch {
	case nvp == 1:
		vp := C.VkViewport{
			x:        C.float(cmd.VP[0].X),
			y:        C.float(cmd.VP[0].Y),
			width:    C.float(cmd.VP[0].Width),
			height:   C.float(cmd.VP[0].Height),
			minDepth: C.float(cmd.VP[0].Znear),
			maxDepth: C.float(cmd.VP[0].Zfar),
		}
		C.vkCmdSetViewport(cb.cb, 0, 1, &vp)
	case nvp > 1:
		vp := make([]C.VkViewport, nvp)
		for i := range vp {
			vp[i] = C.VkViewport{
				x:        C.float(cmd.VP[i].X),
				y:        C.float(cmd.VP[i].Y),
				width:    C.float(cmd.VP[i].Width),
				height:   C.float(cmd.VP[i].Height),
				minDepth: C.float(cmd.VP[i].Znear),
				maxDepth: C.float(cmd.VP[i].Zfar),
			}
		}
		C.vkCmdSetViewport(cb.cb, 0, C.uint32_t(nvp), &vp[0])
	}
}

// scissorCmd records a driver.ScissorCmd in the command buffer.
func (cb *cmdBuffer) scissorCmd(cmd driver.ScissorCmd) {
	ns := len(cmd.S)
	switch {
	case ns == 1:
		s := C.VkRect2D{
			offset: C.VkOffset2D{
				x: C.int32_t(cmd.S[0].X),
				y: C.int32_t(cmd.S[0].Y),
			},
			extent: C.VkExtent2D{
				width:  C.uint32_t(cmd.S[0].Width),
				height: C.uint32_t(cmd.S[0].Height),
			},
		}
		C.vkCmdSetScissor(cb.cb, 0, 1, &s)
	case ns > 1:
		s := make([]C.VkRect2D, ns)
		for i := range s {
			s[i] = C.VkRect2D{
				offset: C.VkOffset2D{
					x: C.int32_t(cmd.S[i].X),
					y: C.int32_t(cmd.S[i].Y),
				},
				extent: C.VkExtent2D{
					width:  C.uint32_t(cmd.S[i].Width),
					height: C.uint32_t(cmd.S[i].Height),
				},
			}
		}
		C.vkCmdSetScissor(cb.cb, 0, C.uint32_t(ns), &s[0])
	}
}

// blendColorCmd records a driver.BlendColorCmd in the command buffer.
func (cb *cmdBuffer) blendColorCmd(cmd driver.BlendColorCmd) {
	color := [4]C.float{
		C.float(cmd.R),
		C.float(cmd.G),
		C.float(cmd.B),
		C.float(cmd.A),
	}
	C.vkCmdSetBlendConstants(cb.cb, &color[0])
}

// stencilRefCmd records a driver.StencilRefCmd in the command buffer.
func (cb *cmdBuffer) stencilRefCmd(cmd driver.StencilRefCmd) {
	C.vkCmdSetStencilReference(cb.cb, C.VK_STENCIL_FACE_FRONT_AND_BACK, C.uint32_t(cmd.Value))
}

// vertexBufCmd records a driver.VertexBufCmd in the command buffer.
func (cb *cmdBuffer) vertexBufCmd(cmd driver.VertexBufCmd) {
	nbuf := len(cmd.Buf)
	switch {
	case nbuf == 1:
		buf := cmd.Buf[0].(*buffer).buf
		off := C.VkDeviceSize(cmd.Off[0])
		C.vkCmdBindVertexBuffers(cb.cb, C.uint32_t(cmd.Start), C.uint32_t(nbuf), &buf, &off)
	case nbuf > 1:
		buf := make([]C.VkBuffer, nbuf)
		off := make([]C.VkDeviceSize, nbuf)
		for i := range buf {
			buf[i] = cmd.Buf[i].(*buffer).buf
			off[i] = C.VkDeviceSize(cmd.Off[i])
		}
		C.vkCmdBindVertexBuffers(cb.cb, C.uint32_t(cmd.Start), C.uint32_t(nbuf), &buf[0], &off[0])
	}
}

// indexBufCmd records a driver.IndexBufCmd in the command buffer.
func (cb *cmdBuffer) indexBufCmd(cmd driver.IndexBufCmd) {
	var typ C.VkIndexType
	switch cmd.Format {
	case driver.Index16:
		typ = C.VK_INDEX_TYPE_UINT16
	case driver.Index32:
		typ = C.VK_INDEX_TYPE_UINT32
	}
	C.vkCmdBindIndexBuffer(cb.cb, cmd.Buf.(*buffer).buf, C.VkDeviceSize(cmd.Off), typ)
}

// descTableCmd records a driver.DescTableCmd in the command buffer.
func (cb *cmdBuffer) descTableCmd(cmd driver.DescTableCmd, bp C.VkPipelineBindPoint) {
	desc := cmd.Desc.(*descTable)
	ncpy := len(cmd.Copy)
	switch {
	case ncpy == 1:
		set := desc.h[cmd.Start].sets[cmd.Copy[0]]
		C.vkCmdBindDescriptorSets(cb.cb, bp, desc.layout, C.uint32_t(cmd.Start), C.uint32_t(ncpy), &set, 0, nil)
	case ncpy > 1:
		set := make([]C.VkDescriptorSet, ncpy)
		for i := range set {
			set[i] = desc.h[cmd.Start+i].sets[cmd.Copy[i]]
		}
		C.vkCmdBindDescriptorSets(cb.cb, bp, desc.layout, C.uint32_t(cmd.Start), C.uint32_t(ncpy), &set[0], 0, nil)
	}
}

// drawCmd records a driver.DrawCmd in the command buffer.
func (cb *cmdBuffer) drawCmd(cmd driver.DrawCmd) {
	nvert := C.uint32_t(cmd.VertCount)
	ninst := C.uint32_t(cmd.InstCount)
	bvert := C.uint32_t(cmd.BaseVert)
	binst := C.uint32_t(cmd.BaseInst)
	C.vkCmdDraw(cb.cb, nvert, ninst, bvert, binst)
}

// indexDrawCmd records a driver.IndexDrawCmd in the command buffer.
func (cb *cmdBuffer) indexDrawCmd(cmd driver.IndexDrawCmd) {
	nidx := C.uint32_t(cmd.IdxCount)
	ninst := C.uint32_t(cmd.InstCount)
	bidx := C.uint32_t(cmd.BaseIdx)
	voff := C.int32_t(cmd.VertOff)
	binst := C.uint32_t(cmd.BaseInst)
	C.vkCmdDrawIndexed(cb.cb, nidx, ninst, bidx, voff, binst)
}

// dispatchCmd records a driver.DispatchCmd in the command buffer.
func (cb *cmdBuffer) dispatchCmd(cmd driver.DispatchCmd) {
	nx := C.uint32_t(cmd.Count[0])
	ny := C.uint32_t(cmd.Count[1])
	nz := C.uint32_t(cmd.Count[2])
	C.vkCmdDispatch(cb.cb, nx, ny, nz)
}

// bufferCopyCmd records a driver.BufferCopyCmd in the command buffer.
func (cb *cmdBuffer) bufferCopyCmd(cmd driver.BufferCopyCmd) {
	cpy := C.VkBufferCopy{
		srcOffset: C.VkDeviceSize(cmd.FromOff),
		dstOffset: C.VkDeviceSize(cmd.ToOff),
		size:      C.VkDeviceSize(cmd.Size),
	}
	C.vkCmdCopyBuffer(cb.cb, cmd.From.(*buffer).buf, cmd.To.(*buffer).buf, 1, &cpy)
}

// imageCopyCmd records a driver.ImageCopyCmd in the command buffer.
func (cb *cmdBuffer) imageCopyCmd(cmd *driver.ImageCopyCmd) {
	from := cmd.From.(*image)
	to := cmd.To.(*image)
	cpy := C.VkImageCopy{
		srcSubresource: C.VkImageSubresourceLayers{
			aspectMask:     from.subres.aspectMask,
			mipLevel:       C.uint32_t(cmd.FromLevel),
			baseArrayLayer: C.uint32_t(cmd.FromLayer),
			layerCount:     C.uint32_t(cmd.Layers),
		},
		srcOffset: C.VkOffset3D{
			x: C.int32_t(cmd.FromOff.X),
			y: C.int32_t(cmd.FromOff.Y),
			z: C.int32_t(cmd.FromOff.Z),
		},
		dstSubresource: C.VkImageSubresourceLayers{
			aspectMask:     to.subres.aspectMask,
			mipLevel:       C.uint32_t(cmd.ToLevel),
			baseArrayLayer: C.uint32_t(cmd.ToLayer),
			layerCount:     C.uint32_t(cmd.Layers),
		},
		dstOffset: C.VkOffset3D{
			x: C.int32_t(cmd.ToOff.X),
			y: C.int32_t(cmd.ToOff.Y),
			z: C.int32_t(cmd.ToOff.Z),
		},
		extent: C.VkExtent3D{
			width:  C.uint32_t(cmd.Size.Width),
			height: C.uint32_t(cmd.Size.Height),
			depth:  C.uint32_t(cmd.Size.Depth),
		},
	}
	// TODO: Ensure images have transitioned to the correct layout.
	layout := C.VkImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
	C.vkCmdCopyImage(cb.cb, from.img, layout, to.img, layout, 1, &cpy)
}

// bufImgCopyCmd records a driver.BufImgCopyCmd in the command buffer.
func (cb *cmdBuffer) bufImgCopyCmd(cmd *driver.BufImgCopyCmd) {
	buf := cmd.Buf.(*buffer)
	img := cmd.Img.(*image)
	var aspect C.VkImageAspectFlags
	if img.subres.aspectMask == C.VK_IMAGE_ASPECT_DEPTH_BIT|C.VK_IMAGE_ASPECT_STENCIL_BIT {
		if cmd.DepthCopy {
			aspect = C.VK_IMAGE_ASPECT_DEPTH_BIT
		} else {
			aspect = C.VK_IMAGE_ASPECT_STENCIL_BIT
		}
	} else {
		aspect = img.subres.aspectMask
	}
	cpy := C.VkBufferImageCopy{
		bufferOffset:      C.VkDeviceSize(cmd.BufOff),
		bufferRowLength:   C.uint32_t(cmd.Stride[0]),
		bufferImageHeight: C.uint32_t(cmd.Stride[1]),
		imageSubresource: C.VkImageSubresourceLayers{
			aspectMask:     aspect,
			mipLevel:       C.uint32_t(cmd.Level),
			baseArrayLayer: C.uint32_t(cmd.Layer),
			layerCount:     1,
		},
		imageOffset: C.VkOffset3D{
			x: C.int32_t(cmd.ImgOff.X),
			y: C.int32_t(cmd.ImgOff.Y),
			z: C.int32_t(cmd.ImgOff.Z),
		},
		imageExtent: C.VkExtent3D{
			width:  C.uint32_t(cmd.Size.Width),
			height: C.uint32_t(cmd.Size.Height),
			depth:  C.uint32_t(cmd.Size.Depth),
		},
	}
	// TODO: Ensure image has transitioned to the correct layout.
	layout := C.VkImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
	C.vkCmdCopyBufferToImage(cb.cb, buf.buf, img.img, layout, 1, &cpy)
}

// imgBufCopyCmd records a driver.ImgBufCopyCmd in the command buffer.
func (cb *cmdBuffer) imgBufCopyCmd(cmd *driver.ImgBufCopyCmd) {
	img := cmd.Img.(*image)
	buf := cmd.Buf.(*buffer)
	var aspect C.VkImageAspectFlags
	if img.subres.aspectMask == C.VK_IMAGE_ASPECT_DEPTH_BIT|C.VK_IMAGE_ASPECT_STENCIL_BIT {
		if cmd.DepthCopy {
			aspect = C.VK_IMAGE_ASPECT_DEPTH_BIT
		} else {
			aspect = C.VK_IMAGE_ASPECT_STENCIL_BIT
		}
	} else {
		aspect = img.subres.aspectMask
	}
	cpy := C.VkBufferImageCopy{
		bufferOffset:      C.VkDeviceSize(cmd.BufOff),
		bufferRowLength:   C.uint32_t(cmd.Stride[0]),
		bufferImageHeight: C.uint32_t(cmd.Stride[1]),
		imageSubresource: C.VkImageSubresourceLayers{
			aspectMask:     aspect,
			mipLevel:       C.uint32_t(cmd.Level),
			baseArrayLayer: C.uint32_t(cmd.Layer),
			layerCount:     1,
		},
		imageOffset: C.VkOffset3D{
			x: C.int32_t(cmd.ImgOff.X),
			y: C.int32_t(cmd.ImgOff.Y),
			z: C.int32_t(cmd.ImgOff.Z),
		},
		imageExtent: C.VkExtent3D{
			width:  C.uint32_t(cmd.Size.Width),
			height: C.uint32_t(cmd.Size.Height),
			depth:  C.uint32_t(cmd.Size.Depth),
		},
	}
	// TODO: Ensure image has transitioned to the correct layout.
	layout := C.VkImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
	C.vkCmdCopyImageToBuffer(cb.cb, img.img, layout, buf.buf, 1, &cpy)
}

// fillCmd records a driver.FillCmd in the command buffer.
func (cb *cmdBuffer) fillCmd(cmd driver.FillCmd) {
	val := C.uint32_t(cmd.Byte)
	val |= val<<24 | val<<16 | val<<8
	C.vkCmdFillBuffer(cb.cb, cmd.Buf.(*buffer).buf, C.VkDeviceSize(cmd.Off), C.VkDeviceSize(cmd.Size), val)
}

// End ends command recording and prepares the command buffer for execution.
func (cb *cmdBuffer) End() error {
	return cb.end()
}

// Reset discards all recorded commands from the command buffer.
func (cb *cmdBuffer) Reset() error {
	return cb.reset()
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
		defer func() {
			sc.mu.Lock()
			sc.curImg--
			sc.syncUsed[sync] = false
			sc.mu.Unlock()
		}()
	default:
		panic("corrupted command buffer presentation")
	}

	// Wait until queue submission has completed execution.
	// Note that queue presentation is not waited for, and as such
	// may not have completed when we send to ch.
	for {
		res := C.vkWaitForFences(d.dev, 1, &cd.fence, C.VK_TRUE, C.UINT64_MAX)
		switch res {
		case C.VK_SUCCESS:
			ch <- nil
			return
		case C.VK_TIMEOUT:
		default:
			ch <- checkResult(res)
			return
		}
	}
}
