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

// ShaderCode implements driver.ShaderCode.
type shaderCode struct {
	d   *Driver
	mod C.VkShaderModule
}

// NewShaderCode creates a new shader code.
func (d *Driver) NewShaderCode(data []byte) (driver.ShaderCode, error) {
	n := len(data)
	// The spec mandates that the code size be a multiple of four.
	if n == 0 || n&3 != 0 {
		return nil, errors.New("vk: invalid shader code size")
	}
	if uintptr(unsafe.Pointer(unsafe.SliceData(data)))&3 != 0 {
		return nil, errors.New("vk: misaligned shader code data")
	}
	p := C.malloc(C.size_t(n))
	defer C.free(p)
	copy(unsafe.Slice((*byte)(p), n), data)
	info := C.VkShaderModuleCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		codeSize: C.size_t(n),
		pCode:    (*C.uint32_t)(p),
	}
	var mod C.VkShaderModule
	err := checkResult(C.vkCreateShaderModule(d.dev, &info, nil, &mod))
	if err != nil {
		return nil, err
	}
	return &shaderCode{
		d:   d,
		mod: mod,
	}, nil
}

// Destroy destroys the shader code.
func (c *shaderCode) Destroy() {
	if c == nil {
		return
	}
	if c.d != nil {
		C.vkDestroyShaderModule(c.d.dev, c.mod, nil)
	}
	*c = shaderCode{}
}
