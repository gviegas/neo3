// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <proc.h>
import "C"

import (
	"github.com/gviegas/scene/driver"
)

// buffer implements driver.Buffer.
type buffer struct {
	m   *memory
	buf C.VkBuffer
}

// NewBuffer creates a new buffer.
func (d *Driver) NewBuffer(size int64, visible bool, usg driver.Usage) (driver.Buffer, error) {
	// TODO: Some of these usages may not be required.
	var u C.VkBufferUsageFlags
	u |= C.VK_BUFFER_USAGE_TRANSFER_SRC_BIT
	u |= C.VK_BUFFER_USAGE_TRANSFER_DST_BIT
	u |= C.VK_BUFFER_USAGE_INDIRECT_BUFFER_BIT
	if usg&(driver.UShaderRead|driver.UShaderWrite) != 0 {
		u |= C.VK_BUFFER_USAGE_STORAGE_TEXEL_BUFFER_BIT
		u |= C.VK_BUFFER_USAGE_STORAGE_BUFFER_BIT
	}
	if usg&driver.UShaderConst != 0 {
		u |= C.VK_BUFFER_USAGE_UNIFORM_TEXEL_BUFFER_BIT
		u |= C.VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT
	}
	if usg&driver.UVertexData != 0 {
		u |= C.VK_BUFFER_USAGE_VERTEX_BUFFER_BIT
	}
	if usg&driver.UIndexData != 0 {
		u |= C.VK_BUFFER_USAGE_INDEX_BUFFER_BIT
	}

	info := C.VkBufferCreateInfo{
		sType:       C.VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		size:        C.VkDeviceSize(size),
		usage:       u,
		sharingMode: C.VK_SHARING_MODE_EXCLUSIVE,
	}
	var buf C.VkBuffer
	err := checkResult(C.vkCreateBuffer(d.dev, &info, nil, &buf))
	if err != nil {
		return nil, err
	}

	var req C.VkMemoryRequirements
	C.vkGetBufferMemoryRequirements(d.dev, buf, &req)
	m, err := d.newMemory(req, visible)
	if err != nil {
		C.vkDestroyBuffer(d.dev, buf, nil)
		return nil, err
	}
	err = checkResult(C.vkBindBufferMemory(d.dev, buf, m.mem, 0))
	if err != nil {
		m.free()
		C.vkDestroyBuffer(d.dev, buf, nil)
		return nil, err
	}
	m.bound = true
	if visible {
		// Keep the memory mapped for the lifetime of the buffer.
		if err = m.mmap(); err != nil {
			m.free()
			C.vkDestroyBuffer(d.dev, buf, nil)
			return nil, err
		}
	}

	return &buffer{
		m:   m,
		buf: buf,
	}, nil
}

// Visible returns whether the buffer is host visible.
func (b *buffer) Visible() bool { return b.m.vis }

// Bytes returns a slice of length b.Cap() referring to the underlying data.
func (b *buffer) Bytes() []byte { return b.m.p }

// Cap returns the capacity of the buffer in bytes.
func (b *buffer) Cap() int64 { return b.m.size }

// Destroy destroys the buffer.
func (b *buffer) Destroy() {
	if b == nil {
		return
	}
	if b.m != nil {
		C.vkDestroyBuffer(b.m.d.dev, b.buf, nil)
		b.m.free()
	}
	*b = buffer{}
}
