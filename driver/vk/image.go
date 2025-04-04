// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <proc.h>
import "C"

import (
	"gviegas/neo3/driver"
)

// image implements driver.Image.
type image struct {
	m      *memory    // Created by Driver.NewImage (s field is nil).
	s      *swapchain // Created by Driver.NewSwapchain (m field is nil).
	img    C.VkImage
	fmt    C.VkFormat
	nonfp  bool // Need to be aware of ui/i color formats in some cases.
	subres C.VkImageSubresourceRange
	usg    C.VkImageUsageFlags
}

// NewImage creates a new image.
func (d *Driver) NewImage(pf driver.PixelFmt, size driver.Dim3D, layers, levels, samples int, usg driver.Usage) (driver.Image, error) {
	format := convPixelFmt(pf)
	scount := convSamples(samples)
	aspect := aspectOf(pf)

	var typ C.VkImageType
	var extent C.VkExtent3D
	var flags C.VkImageCreateFlags
	switch {
	case size.Depth >= 1:
		// NOTE: This is disallowed currently.
		//if d.dvers >= C.VK_API_VERSION_1_1 {
		//	flags |= C.VK_IMAGE_CREATE_2D_ARRAY_COMPATIBLE_BIT
		//}
		typ = C.VK_IMAGE_TYPE_3D
		extent = C.VkExtent3D{
			width:  C.uint32_t(size.Width),
			height: C.uint32_t(size.Height),
			depth:  C.uint32_t(size.Depth),
		}
	case size.Height >= 1:
		if samples == 1 {
			if size.Width == size.Height && layers >= 6 {
				flags |= C.VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT
			}
		}
		typ = C.VK_IMAGE_TYPE_2D
		extent = C.VkExtent3D{
			width:  C.uint32_t(size.Width),
			height: C.uint32_t(size.Height),
			depth:  1,
		}
	default:
		typ = C.VK_IMAGE_TYPE_1D
		extent = C.VkExtent3D{
			width:  C.uint32_t(size.Width),
			height: 1,
			depth:  1,
		}
	}

	var usage C.VkImageUsageFlags
	if usg&driver.UCopySrc != 0 {
		usage |= C.VK_IMAGE_USAGE_TRANSFER_SRC_BIT
	}
	if usg&driver.UCopyDst != 0 {
		usage |= C.VK_IMAGE_USAGE_TRANSFER_DST_BIT
	}
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
	// TODO: This check should be driver-independent.
	if usage == 0 {
		panic("cannot create image without a valid usage")
	}

	var prop C.VkImageFormatProperties
	res := C.vkGetPhysicalDeviceImageFormatProperties(d.pdev, format, typ, C.VK_IMAGE_TILING_OPTIMAL, usage, flags, &prop)
	if err := checkResult(res); err != nil {
		return nil, err
	}
	if extent.width > prop.maxExtent.width || extent.height > prop.maxExtent.height || extent.depth > prop.maxExtent.depth ||
		C.uint32_t(layers) > prop.maxArrayLayers || C.uint32_t(levels) > prop.maxMipLevels ||
		C.VkSampleCountFlags(scount)&prop.sampleCounts == 0 {
		// TODO: This error is a bit misleading.
		return nil, errUnsupportedFormat
	}

	info := C.VkImageCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		flags:         flags,
		imageType:     typ,
		format:        format,
		extent:        extent,
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
		m:     m,
		img:   img,
		fmt:   format,
		nonfp: pf.IsNonfloatColor(),
		subres: C.VkImageSubresourceRange{
			aspectMask: aspect,
			levelCount: C.uint32_t(levels),
			layerCount: C.uint32_t(layers),
		},
		usg: usage,
	}
	return im, nil
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

// Two planes for depth/stencil formats.
// When used as render target, the aspect flags of the subresource
// don't matter. However, for descriptor binding, only one bit
// can be set. We handle this case by creating the first view as
// depth-only and the second view as stencil-only.
const maxPlane = 2

// imageView implements driver.ImageView.
type imageView struct {
	i      *image
	view   [maxPlane]C.VkImageView
	subres [maxPlane]C.VkImageSubresourceRange
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
	}

	// NOTE: Currently this assumes that only depth/stencil
	// may need another view.
	var view [maxPlane]C.VkImageView
	subres := [maxPlane]C.VkImageSubresourceRange{
		{
			aspectMask:     im.subres.aspectMask,
			baseMipLevel:   C.uint32_t(level),
			levelCount:     C.uint32_t(levels),
			baseArrayLayer: C.uint32_t(layer),
			layerCount:     C.uint32_t(layers),
		},
	}
	n := 1
	const (
		comb = C.VK_IMAGE_ASPECT_DEPTH_BIT | C.VK_IMAGE_ASPECT_STENCIL_BIT
		bind = C.VK_IMAGE_USAGE_SAMPLED_BIT | C.VK_IMAGE_USAGE_STORAGE_BIT
	)
	if im.subres.aspectMask == comb && im.usg&bind != 0 {
		subres[1] = subres[0]
		subres[1].aspectMask = C.VK_IMAGE_ASPECT_STENCIL_BIT
		subres[0].aspectMask = C.VK_IMAGE_ASPECT_DEPTH_BIT
		n = 2
	}

	var dev C.VkDevice
	if im.m != nil {
		dev = im.m.d.dev
	} else {
		dev = im.s.d.dev
	}
	for i := 0; i < n; i++ {
		info.subresourceRange = subres[i]
		err := checkResult(C.vkCreateImageView(dev, &info, nil, &view[i]))
		if err != nil {
			for j := 0; j < i; j++ {
				C.vkDestroyImageView(dev, view[j], nil)
			}
			return nil, err
		}
	}
	return &imageView{
		i:      im,
		view:   view,
		subres: subres,
	}, nil
}

// Image returns the image from which the view was created.
func (v *imageView) Image() driver.Image { return v.i }

// Destroy destroys the image view.
func (v *imageView) Destroy() {
	if v == nil {
		return
	}
	if v.i != nil {
		if v.i.m != nil {
			for i := range v.view {
				C.vkDestroyImageView(v.i.m.d.dev, v.view[i], nil)
			}
		} else if v.i.s != nil {
			for i := range v.view {
				C.vkDestroyImageView(v.i.s.d.dev, v.view[i], nil)
			}
		}
	}
	*v = imageView{}
}

// convPixelFmt converts a driver.PixelFmt to a VkFormat.
func convPixelFmt(pf driver.PixelFmt) C.VkFormat {
	if pf.IsInternal() {
		return C.VkFormat(^pf + 1)
	}

	switch pf {
	case driver.FInvalid:
		return C.VK_FORMAT_UNDEFINED

	case driver.RGBA8Unorm:
		return C.VK_FORMAT_R8G8B8A8_UNORM
	case driver.RGBA8Norm:
		return C.VK_FORMAT_R8G8B8A8_SNORM
	case driver.RGBA8Uint:
		return C.VK_FORMAT_R8G8B8A8_UINT
	case driver.RGBA8Int:
		return C.VK_FORMAT_R8G8B8A8_SINT
	case driver.RGBA8SRGB:
		return C.VK_FORMAT_R8G8B8A8_SRGB
	case driver.BGRA8Unorm:
		return C.VK_FORMAT_B8G8R8A8_UNORM
	case driver.BGRA8SRGB:
		return C.VK_FORMAT_B8G8R8A8_SRGB
	case driver.RG8Unorm:
		return C.VK_FORMAT_R8G8_UNORM
	case driver.RG8Norm:
		return C.VK_FORMAT_R8G8_SNORM
	case driver.RG8Uint:
		return C.VK_FORMAT_R8G8_UINT
	case driver.RG8Int:
		return C.VK_FORMAT_R8G8_SINT
	case driver.R8Unorm:
		return C.VK_FORMAT_R8_UNORM
	case driver.R8Norm:
		return C.VK_FORMAT_R8_SNORM
	case driver.R8Uint:
		return C.VK_FORMAT_R8_UINT
	case driver.R8Int:
		return C.VK_FORMAT_R8_SINT

	case driver.RGBA16Float:
		return C.VK_FORMAT_R16G16B16A16_SFLOAT
	case driver.RGBA16Uint:
		return C.VK_FORMAT_R16G16B16A16_UINT
	case driver.RGBA16Int:
		return C.VK_FORMAT_R16G16B16A16_SINT
	case driver.RG16Float:
		return C.VK_FORMAT_R16G16_SFLOAT
	case driver.RG16Uint:
		return C.VK_FORMAT_R16G16_UINT
	case driver.RG16Int:
		return C.VK_FORMAT_R16G16_SINT
	case driver.R16Float:
		return C.VK_FORMAT_R16_SFLOAT
	case driver.R16Uint:
		return C.VK_FORMAT_R16_UINT
	case driver.R16Int:
		return C.VK_FORMAT_R16_SINT

	case driver.RGBA32Float:
		return C.VK_FORMAT_R32G32B32A32_SFLOAT
	case driver.RGBA32Uint:
		return C.VK_FORMAT_R32G32B32A32_UINT
	case driver.RGBA32Int:
		return C.VK_FORMAT_R32G32B32A32_SINT
	case driver.RG32Float:
		return C.VK_FORMAT_R32G32_SFLOAT
	case driver.RG32Uint:
		return C.VK_FORMAT_R32G32_UINT
	case driver.RG32Int:
		return C.VK_FORMAT_R32G32_SINT
	case driver.R32Float:
		return C.VK_FORMAT_R32_SFLOAT
	case driver.R32Uint:
		return C.VK_FORMAT_R32_UINT
	case driver.R32Int:
		return C.VK_FORMAT_R32_SINT

	case driver.D16Unorm:
		return C.VK_FORMAT_D16_UNORM
	case driver.D32Float:
		return C.VK_FORMAT_D32_SFLOAT
	case driver.S8Uint:
		return C.VK_FORMAT_S8_UINT
	case driver.D24UnormS8Uint:
		return C.VK_FORMAT_D24_UNORM_S8_UINT
	case driver.D32FloatS8Uint:
		return C.VK_FORMAT_D32_SFLOAT_S8_UINT
	}

	// Expected to be unreachable.
	return ^C.VkFormat(0)
}

// internalFmt returns vf as an internal driver.PixelFmt.
func internalFmt(vf C.VkFormat) driver.PixelFmt { return driver.PixelFmt(int32(^vf) + 1) }

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
	case driver.FInvalid:
		return 0
	case driver.D24UnormS8Uint, driver.D32FloatS8Uint:
		return C.VK_IMAGE_ASPECT_DEPTH_BIT | C.VK_IMAGE_ASPECT_STENCIL_BIT
	case driver.D16Unorm, driver.D32Float:
		return C.VK_IMAGE_ASPECT_DEPTH_BIT
	case driver.S8Uint:
		return C.VK_IMAGE_ASPECT_STENCIL_BIT
	}
	return C.VK_IMAGE_ASPECT_COLOR_BIT
}
