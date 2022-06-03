// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <proc.h>
import "C"

import (
	"github.com/gviegas/scene/driver"
)

// sampler implements driver.Sampler.
type sampler struct {
	d    *Driver
	splr C.VkSampler
}

// NewSampler creates a new sampler.
func (d *Driver) NewSampler(spln *driver.Sampling) (driver.Sampler, error) {
	info := C.VkSamplerCreateInfo{
		sType:        C.VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
		magFilter:    convFilter(spln.Mag),
		minFilter:    convFilter(spln.Min),
		mipmapMode:   convMipFilter(spln.Mipmap),
		addressModeU: convAddrMode(spln.AddrU),
		addressModeV: convAddrMode(spln.AddrV),
		addressModeW: convAddrMode(spln.AddrW),
		// TODO: Anisotropy is a feature - disable it for now.
		//maxAnisotropy: C.float(maxAniso),
		compareEnable: C.VK_TRUE,
		compareOp:     convCmpFunc(spln.Cmp),
		minLod:        C.float(spln.MinLOD),
		maxLod:        C.float(spln.MaxLOD),
		borderColor:   C.VK_BORDER_COLOR_FLOAT_OPAQUE_BLACK,
	}
	var splr C.VkSampler
	err := checkResult(C.vkCreateSampler(d.dev, &info, nil, &splr))
	if err != nil {
		return nil, err
	}
	return &sampler{
		d:    d,
		splr: splr,
	}, nil
}

// Destroy destroys the sampler.
func (s *sampler) Destroy() {
	if s == nil {
		return
	}
	if s.d != nil {
		C.vkDestroySampler(s.d.dev, s.splr, nil)
	}
	*s = sampler{}
}

// convFilter converts a driver.Filter to a VkFilter.
func convFilter(f driver.Filter) C.VkFilter {
	switch f {
	case driver.FNearest:
		return C.VK_FILTER_NEAREST
	case driver.FLinear:
		return C.VK_FILTER_LINEAR
	}

	// Expected to be unreachable.
	return ^C.VkFilter(0)
}

// convMipFilter converts a driver.Filter to a VkSamplerMipmapMode.
func convMipFilter(f driver.Filter) C.VkSamplerMipmapMode {
	switch f {
	case driver.FNoMipmap, driver.FNearest:
		return C.VK_SAMPLER_MIPMAP_MODE_NEAREST
	case driver.FLinear:
		return C.VK_SAMPLER_MIPMAP_MODE_LINEAR
	}

	// Expected to be unreachable.
	return ^C.VkSamplerMipmapMode(0)
}

// convAddrMode converts a driver.AddrMode to a VkSamplerAdressMode.
func convAddrMode(am driver.AddrMode) C.VkSamplerAddressMode {
	switch am {
	case driver.AWrap:
		return C.VK_SAMPLER_ADDRESS_MODE_REPEAT
	case driver.AMirror:
		return C.VK_SAMPLER_ADDRESS_MODE_MIRRORED_REPEAT
	case driver.AClamp:
		return C.VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE
	}

	// Expected to be unreachable.
	return ^C.VkSamplerAddressMode(0)
}

// convCmpFunc converts a driver.CmpFunc to a VkCompareOp.
func convCmpFunc(cf driver.CmpFunc) C.VkCompareOp {
	switch cf {
	case driver.CNever:
		return C.VK_COMPARE_OP_NEVER
	case driver.CLess:
		return C.VK_COMPARE_OP_LESS
	case driver.CEqual:
		return C.VK_COMPARE_OP_EQUAL
	case driver.CLessEqual:
		return C.VK_COMPARE_OP_LESS_OR_EQUAL
	case driver.CGreater:
		return C.VK_COMPARE_OP_GREATER
	case driver.CNotEqual:
		return C.VK_COMPARE_OP_NOT_EQUAL
	case driver.CGreaterEqual:
		return C.VK_COMPARE_OP_GREATER_OR_EQUAL
	case driver.CAlways:
		return C.VK_COMPARE_OP_ALWAYS
	}

	// Expected to be unreachable.
	return ^C.VkCompareOp(0)
}
