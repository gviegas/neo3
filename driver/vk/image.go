// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <proc.h>
import "C"

import (
	"github.com/gviegas/scene/driver"
)

// image implements driver.Image.
type image struct {
	m      *memory
	img    C.VkImage
	fmt    C.VkFormat
	subres C.VkImageSubresourceRange
	layout C.VkImageLayout
}

// NewImage creates a new image.
func (d *Driver) NewImage(pf driver.PixelFmt, size driver.Dim3D, layers, levels, samples int, usg driver.Usage) (driver.Image, error) {
	format := convPixelFmt(pf)
	scount := convSamples(samples)
	aspect := aspectOf(pf)

	var typ C.VkImageType
	var flags C.VkImageCreateFlags
	switch {
	case size.Depth > 1:
		if d.dvers >= C.VK_API_VERSION_1_1 {
			flags |= C.VK_IMAGE_CREATE_2D_ARRAY_COMPATIBLE_BIT
		}
		typ = C.VK_IMAGE_TYPE_3D
	case size.Height > 1:
		if samples == 1 {
			if size.Width == size.Height && layers >= 6 {
				flags |= C.VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT
			}
		}
		typ = C.VK_IMAGE_TYPE_2D
	default:
		typ = C.VK_IMAGE_TYPE_1D
	}

	var usage C.VkImageUsageFlags
	if usg&(driver.UShaderRead|driver.UShaderWrite) != 0 {
		usage |= C.VK_IMAGE_USAGE_STORAGE_BIT
	}
	if usg&driver.UShaderSample != 0 {
		usage |= C.VK_IMAGE_USAGE_SAMPLED_BIT
	}
	if usg&driver.URenderTarget != 0 {
		if aspect == C.VK_IMAGE_ASPECT_COLOR_BIT {
			usage |= C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT
		} else {
			usage |= C.VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT
		}
	}
	// At least one valid usage must have been set.
	if usage == 0 {
		// We panic here because this is certainly a
		// client error (i.e., this image is useless)
		// and also because the spec forbids creating
		// a view in this case.
		panic("cannot create image without a valid usage")
	}
	usage |= C.VK_IMAGE_USAGE_TRANSFER_SRC_BIT
	usage |= C.VK_IMAGE_USAGE_TRANSFER_DST_BIT

	var prop C.VkImageFormatProperties
	res := C.vkGetPhysicalDeviceImageFormatProperties(d.pdev, format, typ, C.VK_IMAGE_TILING_OPTIMAL, usage, flags, &prop)
	if err := checkResult(res); err != nil {
		return nil, err
	} else {
		w := int(prop.maxExtent.width)
		h := int(prop.maxExtent.height)
		d := int(prop.maxExtent.depth)
		ly := int(prop.maxArrayLayers)
		lv := int(prop.maxMipLevels)
		sc := C.VkSampleCountFlagBits(prop.sampleCounts)
		if size.Width > w || size.Height > h || size.Depth > d || layers > ly || levels > lv || scount&sc == 0 {
			// TODO: This error is a bit misleading.
			return nil, errUnsupportedFormat
		}
	}

	info := C.VkImageCreateInfo{
		sType:     C.VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		flags:     flags,
		imageType: typ,
		format:    format,
		extent: C.VkExtent3D{
			width:  C.uint32_t(size.Width),
			height: C.uint32_t(size.Height),
			depth:  C.uint32_t(size.Depth),
		},
		mipLevels:     C.uint32_t(levels),
		arrayLayers:   C.uint32_t(layers),
		samples:       scount,
		tiling:        C.VK_IMAGE_TILING_OPTIMAL,
		usage:         usage,
		sharingMode:   C.VK_SHARING_MODE_EXCLUSIVE,
		initialLayout: C.VK_IMAGE_LAYOUT_UNDEFINED,
	}
	var img C.VkImage
	err := checkResult(C.vkCreateImage(d.dev, &info, nil, &img))
	if err != nil {
		return nil, err
	}

	var req C.VkMemoryRequirements
	C.vkGetImageMemoryRequirements(d.dev, img, &req)
	m, err := d.newMemory(req, false)
	if err != nil {
		C.vkDestroyImage(d.dev, img, nil)
		return nil, err
	}
	err = checkResult(C.vkBindImageMemory(d.dev, img, m.mem, 0))
	if err != nil {
		m.free()
		C.vkDestroyImage(d.dev, img, nil)
		return nil, err
	}
	m.bound = true

	im := &image{
		m:   m,
		img: img,
		fmt: format,
		subres: C.VkImageSubresourceRange{
			aspectMask: aspect,
			levelCount: C.uint32_t(levels),
			layerCount: C.uint32_t(layers),
		},
		layout: info.initialLayout,
	}
	if err = im.transition(); err != nil {
		im.Destroy()
		return nil, err
	}
	return im, nil
}

// transition transitions the image to the general layout.
// TODO: Improve this.
func (im *image) transition() error {
	// TODO: Should put a lock here if this is ever going
	// to be used outside of NewImage.
	if im.layout == C.VK_IMAGE_LAYOUT_GENERAL {
		return nil
	}
	ic, err := im.m.d.NewCmdBuffer()
	if err != nil {
		return err
	}
	cb := ic.(*cmdBuffer)
	defer cb.Destroy()
	if err := cb.Begin(); err != nil {
		return err
	}
	cb.transition(im, C.VK_IMAGE_LAYOUT_GENERAL, 0, 0, 0, 0)
	if err := cb.End(); err != nil {
		return err
	}
	ch := make(chan error)
	go im.m.d.Commit([]driver.CmdBuffer{cb}, ch)
	err = <-ch
	if err == nil {
		im.layout = C.VK_IMAGE_LAYOUT_GENERAL
	}
	return err
}

// Destroy destroys the image.
func (im *image) Destroy() {
	if im == nil {
		return
	}
	if im.m != nil {
		C.vkDestroyImage(im.m.d.dev, im.img, nil)
		im.m.free()
	}
	*im = image{}
}

// imageView implements driver.ImageView.
type imageView struct {
	i      *image     // Created from an image (s field is nil).
	s      *swapchain // Created from a swapchain (i field is nil).
	view   C.VkImageView
	subres C.VkImageSubresourceRange
}

// NewView creates a new image view.
func (im *image) NewView(typ driver.ViewType, layer, layers, level, levels int) (driver.ImageView, error) {
	var viewType C.VkImageViewType
	switch typ {
	case driver.IView1D:
		viewType = C.VK_IMAGE_VIEW_TYPE_1D
	case driver.IView2D, driver.IView2DMS:
		viewType = C.VK_IMAGE_VIEW_TYPE_2D
	case driver.IView3D:
		viewType = C.VK_IMAGE_VIEW_TYPE_3D
	case driver.IViewCube:
		viewType = C.VK_IMAGE_VIEW_TYPE_CUBE
	case driver.IView1DArray:
		viewType = C.VK_IMAGE_VIEW_TYPE_1D_ARRAY
	case driver.IView2DArray, driver.IView2DMSArray:
		viewType = C.VK_IMAGE_VIEW_TYPE_2D_ARRAY
	case driver.IViewCubeArray:
		viewType = C.VK_IMAGE_VIEW_TYPE_CUBE_ARRAY
	}
	info := C.VkImageViewCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		image:    im.img,
		viewType: viewType,
		format:   im.fmt,
		components: C.VkComponentMapping{
			r: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			g: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			b: C.VK_COMPONENT_SWIZZLE_IDENTITY,
			a: C.VK_COMPONENT_SWIZZLE_IDENTITY,
		},
		subresourceRange: C.VkImageSubresourceRange{
			aspectMask:     im.subres.aspectMask,
			baseMipLevel:   C.uint32_t(level),
			levelCount:     C.uint32_t(levels),
			baseArrayLayer: C.uint32_t(layer),
			layerCount:     C.uint32_t(layers),
		},
	}
	var view C.VkImageView
	err := checkResult(C.vkCreateImageView(im.m.d.dev, &info, nil, &view))
	if err != nil {
		return nil, err
	}
	return &imageView{
		i:      im,
		view:   view,
		subres: info.subresourceRange,
	}, nil
}

// Destroy destroys the image view.
func (v *imageView) Destroy() {
	if v == nil {
		return
	}
	if v.i != nil {
		C.vkDestroyImageView(v.i.m.d.dev, v.view, nil)
	} else if v.s != nil {
		C.vkDestroyImageView(v.s.d.dev, v.view, nil)
	}
	*v = imageView{}
}

// convPixelFmt converts a driver.PixelFmt to a VkFormat.
func convPixelFmt(pf driver.PixelFmt) C.VkFormat {
	if pf.IsInternal() {
		// The driver.FInternal bit is not set in any
		// of the Vulkan formats, so this hack works.
		return C.VkFormat(^driver.FInternal & pf)
	}

	switch pf {
	case driver.RGBA8un:
		return C.VK_FORMAT_R8G8B8A8_UNORM
	case driver.RGBA8n:
		return C.VK_FORMAT_R8G8B8A8_SNORM
	case driver.RGBA8sRGB:
		return C.VK_FORMAT_R8G8B8A8_SRGB
	case driver.BGRA8un:
		return C.VK_FORMAT_B8G8R8A8_UNORM
	case driver.BGRA8sRGB:
		return C.VK_FORMAT_B8G8R8A8_SRGB
	case driver.RG8un:
		return C.VK_FORMAT_R8G8_UNORM
	case driver.RG8n:
		return C.VK_FORMAT_R8G8_SNORM
	case driver.R8un:
		return C.VK_FORMAT_R8_UNORM
	case driver.R8n:
		return C.VK_FORMAT_R8_SNORM

	case driver.RGBA16f:
		return C.VK_FORMAT_R16G16B16A16_SFLOAT
	case driver.RG16f:
		return C.VK_FORMAT_R16G16_SFLOAT
	case driver.R16f:
		return C.VK_FORMAT_R16_SFLOAT

	case driver.RGBA32f:
		return C.VK_FORMAT_R32G32B32A32_SFLOAT
	case driver.RG32f:
		return C.VK_FORMAT_R32G32_SFLOAT
	case driver.R32f:
		return C.VK_FORMAT_R32_SFLOAT

	case driver.D16un:
		return C.VK_FORMAT_D16_UNORM
	case driver.D32f:
		return C.VK_FORMAT_D32_SFLOAT
	case driver.S8ui:
		return C.VK_FORMAT_S8_UINT
	case driver.D24unS8ui:
		return C.VK_FORMAT_D24_UNORM_S8_UINT
	case driver.D32fS8ui:
		return C.VK_FORMAT_D32_SFLOAT_S8_UINT
	}

	// Expected to be unreachable.
	return C.VK_FORMAT_UNDEFINED
}

// internalFmt returns vf as an internal driver.PixelFmt.
func internalFmt(vf C.VkFormat) driver.PixelFmt { return driver.PixelFmt(vf) | driver.FInternal }

// convSamples converts a samples value to a VkSampleCountFlagBits.
func convSamples(ns int) C.VkSampleCountFlagBits {
	switch ns {
	case 1:
		return C.VK_SAMPLE_COUNT_1_BIT
	case 2:
		return C.VK_SAMPLE_COUNT_2_BIT
	case 4:
		return C.VK_SAMPLE_COUNT_4_BIT
	case 8:
		return C.VK_SAMPLE_COUNT_8_BIT
	case 16:
		return C.VK_SAMPLE_COUNT_16_BIT
	case 32:
		return C.VK_SAMPLE_COUNT_32_BIT
	case 64:
		return C.VK_SAMPLE_COUNT_64_BIT
	}

	// Expected to be unreachable.
	return ^C.VkSampleCountFlagBits(0)
}

// aspectOf returns a VkImageAspectFlags identifying the aspects of
// a given driver.PixelFmt.
func aspectOf(pf driver.PixelFmt) C.VkImageAspectFlags {
	switch pf {
	case driver.D24unS8ui, driver.D32fS8ui:
		return C.VK_IMAGE_ASPECT_DEPTH_BIT | C.VK_IMAGE_ASPECT_STENCIL_BIT
	case driver.D16un, driver.D32f:
		return C.VK_IMAGE_ASPECT_DEPTH_BIT
	case driver.S8ui:
		return C.VK_IMAGE_ASPECT_STENCIL_BIT
	}
	return C.VK_IMAGE_ASPECT_COLOR_BIT
}
