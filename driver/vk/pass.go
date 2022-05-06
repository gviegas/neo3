// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"errors"
	"unsafe"

	"github.com/gviegas/scene/driver"
)

// renderPass implements driver.RenderPass.
type renderPass struct {
	d    *Driver
	pass C.VkRenderPass
	// Aspect of each attachment.
	aspect []C.VkImageAspectFlags
	// Number of color attachments used by
	// each subpass.
	ncolor []int
}

// NewRenderPass creates a new render pass.
func (d *Driver) NewRenderPass(att []driver.Attachment, sub []driver.Subpass) (driver.RenderPass, error) {
	// Render passes need not have any attachments.
	var patt *C.VkAttachmentDescription

	// However, they must have at least one subpass.
	psub := (*C.VkSubpassDescription)(C.malloc(C.size_t(len(sub)) * C.sizeof_VkSubpassDescription))
	defer C.free(unsafe.Pointer(psub))
	ssub := unsafe.Slice(psub, len(sub))

	if len(att) > 0 {
		patt = (*C.VkAttachmentDescription)(C.malloc(C.size_t(len(att)) * C.sizeof_VkAttachmentDescription))
		defer C.free(unsafe.Pointer(patt))
		satt := unsafe.Slice(patt, len(att))

		for i := range satt {
			satt[i] = C.VkAttachmentDescription{
				//flags:          C.VK_ATTACHMENT_DESCRIPTION_MAY_ALIAS_BIT,
				format:         convPixelFmt(att[i].Format),
				samples:        convSamples(att[i].Samples),
				loadOp:         convLoadOp(att[i].Load[0]),
				storeOp:        convStoreOp(att[i].Store[0]),
				stencilLoadOp:  convLoadOp(att[i].Load[1]),
				stencilStoreOp: convStoreOp(att[i].Store[1]),
				initialLayout:  C.VK_IMAGE_LAYOUT_GENERAL,
				finalLayout:    C.VK_IMAGE_LAYOUT_GENERAL,
			}
		}

		nref := len(sub) * len(att)
		pref := (*C.VkAttachmentReference)(C.malloc(C.size_t(nref) * C.sizeof_VkAttachmentReference))
		defer C.free(unsafe.Pointer(pref))
		sref := unsafe.Slice(pref, nref)
		ppre := (*C.uint32_t)(C.malloc(C.size_t(nref) * C.sizeof_uint32_t))
		defer C.free(unsafe.Pointer(ppre))
		spre := unsafe.Slice(ppre, nref)

		// We will preserve anything that is not used.
		noPre := make([]bool, len(att))

		for i := range ssub {
			var pclr, pds, pres *C.VkAttachmentReference
			var ppre *C.uint32_t
			npre := 0

			if len(sub[i].Color) > 0 {
				pclr = &sref[0]
				for j, k := range sub[i].Color {
					sref[j].attachment = C.uint32_t(k)
					sref[j].layout = C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
					noPre[k] = true
				}
			}
			if sub[i].DS >= 0 && sub[i].DS < len(att) {
				pds = &sref[len(sub[i].Color)]
				pds.attachment = C.uint32_t(sub[i].DS)
				pds.layout = C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
				noPre[sub[i].DS] = true
			}
			if len(sub[i].MSR) > 0 {
				ires := len(sub[i].Color)
				if pds != nil {
					ires++
				}
				pres = &sref[ires]
				// TODO: Depth/stencil resolve.
				for j, k := range sub[i].MSR {
					if sub[i].MSR[j] >= 0 && sub[i].MSR[j] < len(att) {
						sref[ires+j].attachment = C.uint32_t(k)
						sref[ires+j].layout = C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
						noPre[k] = true
					} else {
						sref[ires+j].attachment = C.VK_ATTACHMENT_UNUSED
						sref[ires+j].layout = C.VK_IMAGE_LAYOUT_UNDEFINED
					}
				}
			}
			for j := range noPre {
				if !noPre[j] {
					spre[npre] = C.uint32_t(j)
					npre++
				} else {
					noPre[j] = false
				}
			}
			if npre != 0 {
				ppre = &spre[0]
			}
			sref = sref[len(att):]
			spre = spre[len(att):]

			ssub[i] = C.VkSubpassDescription{
				pipelineBindPoint:       C.VK_PIPELINE_BIND_POINT_GRAPHICS,
				colorAttachmentCount:    C.uint32_t(len(sub[i].Color)),
				pColorAttachments:       pclr,
				pResolveAttachments:     pres,
				pDepthStencilAttachment: pds,
				preserveAttachmentCount: C.uint32_t(npre),
				pPreserveAttachments:    ppre,
			}
		}
	} else {
		// This is a render pass with no render targets.
		for i := range ssub {
			ssub[i] = C.VkSubpassDescription{
				pipelineBindPoint: C.VK_PIPELINE_BIND_POINT_GRAPHICS,
			}
		}
	}

	// In the worst case, we will have half the subpasses running in
	// parallel with external dependencies while the other half, also
	// running in parallel, waits for the first half to complete.
	// This translates to a lot of dependencies.
	maxDep := (len(sub) + len(sub)&1) / 2
	maxDep = maxDep + maxDep*maxDep
	pdep := (*C.VkSubpassDependency)(C.malloc(C.size_t(maxDep) * C.sizeof_VkSubpassDependency))
	defer C.free(unsafe.Pointer(pdep))
	sdep := unsafe.Slice(pdep, maxDep)

	// TODO: Improve this.
	//const srcStg = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT
	const srcStg = C.VK_PIPELINE_STAGE_ALL_COMMANDS_BIT
	const dstStg = C.VK_PIPELINE_STAGE_DRAW_INDIRECT_BIT
	const srcAcc = C.VK_ACCESS_MEMORY_WRITE_BIT
	const dstAcc = C.VK_ACCESS_MEMORY_WRITE_BIT | C.VK_ACCESS_MEMORY_READ_BIT

	var iwait, idep, ndep int
	if sub[0].Wait {
		// Wait in the first subpass is treated as external dependency.
		sdep[0] = C.VkSubpassDependency{
			srcSubpass:    C.VK_SUBPASS_EXTERNAL,
			dstSubpass:    0,
			srcStageMask:  srcStg,
			dstStageMask:  dstStg,
			srcAccessMask: srcAcc,
			dstAccessMask: dstAcc,
		}
		ndep++
		idep++
	}
	for i := 1; i < len(sub); i++ {
		switch {
		case sub[i].Wait:
			// This subpass can only starts executing when all the previous
			// ones have finished.
			for j := iwait; j < i; j++ {
				sdep[ndep] = C.VkSubpassDependency{
					srcSubpass:    C.uint32_t(j),
					dstSubpass:    C.uint32_t(i),
					srcStageMask:  srcStg,
					dstStageMask:  dstStg,
					srcAccessMask: srcAcc,
					dstAccessMask: dstAcc,
				}
				ndep++
			}
			// Wait is now relative to this subpass.
			iwait = i
			idep = ndep
		case ndep > 0:
			// This subpass can execute in parallel with the previous ones,
			// but must wait along with them.
			for j := idep - 1; j >= 0 && sdep[j].dstSubpass == C.uint32_t(iwait); j-- {
				sdep[ndep] = C.VkSubpassDependency{
					srcSubpass:    sdep[j].srcSubpass,
					dstSubpass:    C.uint32_t(i),
					srcStageMask:  srcStg,
					dstStageMask:  dstStg,
					srcAccessMask: srcAcc,
					dstAccessMask: dstAcc,
				}
				ndep++
			}
		default:
			continue
		}
	}

	info := C.VkRenderPassCreateInfo{
		sType:           C.VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
		attachmentCount: C.uint32_t(len(att)),
		pAttachments:    patt,
		subpassCount:    C.uint32_t(len(sub)),
		pSubpasses:      psub,
		dependencyCount: C.uint32_t(ndep),
		pDependencies:   pdep,
	}
	var pass C.VkRenderPass
	err := checkResult(C.vkCreateRenderPass(d.dev, &info, nil, &pass))
	if err != nil {
		return nil, err
	}
	// Image aspect is needed when clearing attachments in a render pass.
	aspect := make([]C.VkImageAspectFlags, len(att))
	for i := range aspect {
		aspect[i] = aspectOf(att[i].Format)
	}
	// Color count is needed when defining the color blend state.
	ncolor := make([]int, len(sub))
	for i := range ncolor {
		ncolor[i] = len(sub[i].Color)
	}
	return &renderPass{
		d:      d,
		pass:   pass,
		aspect: aspect,
		ncolor: ncolor,
	}, nil
}

// Destroy destroy the render pass.
func (p *renderPass) Destroy() {
	if p == nil {
		return
	}
	if p.d != nil {
		C.vkDestroyRenderPass(p.d.dev, p.pass, nil)
	}
	*p = renderPass{}
}

// framebuf implements driver.Framebuf.
type framebuf struct {
	p      *renderPass
	fb     C.VkFramebuffer
	width  int
	height int
}

// NewFB creates a new framebuffer.
func (p *renderPass) NewFB(iv []driver.ImageView, width, height, layers int) (driver.Framebuf, error) {
	var pview *C.VkImageView
	if len(iv) > 0 {
		pview = (*C.VkImageView)(C.malloc(C.size_t(len(iv)) * C.sizeof_VkImageView))
		defer C.free(unsafe.Pointer(pview))
		sview := unsafe.Slice(pview, len(iv))
		for i := range iv {
			iv := iv[i].(*imageView)
			if iv == nil {
				return nil, errors.New("nil image view")
			}
			sview[i] = iv.view
		}
	}
	info := C.VkFramebufferCreateInfo{
		sType:           C.VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
		renderPass:      p.pass,
		attachmentCount: C.uint32_t(len(iv)),
		pAttachments:    pview,
		width:           C.uint32_t(width),
		height:          C.uint32_t(height),
		layers:          C.uint32_t(layers),
	}
	var fb C.VkFramebuffer
	err := checkResult(C.vkCreateFramebuffer(p.d.dev, &info, nil, &fb))
	if err != nil {
		return nil, err
	}
	return &framebuf{
		p:      p,
		fb:     fb,
		width:  width,
		height: height,
	}, nil
}

// Destroy destroys the framebuffer.
func (f *framebuf) Destroy() {
	if f == nil {
		return
	}
	if f.p != nil {
		C.vkDestroyFramebuffer(f.p.d.dev, f.fb, nil)
	}
	*f = framebuf{}
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
