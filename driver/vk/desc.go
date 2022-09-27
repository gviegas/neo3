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

// descHeap implements driver.DescHeap.
type descHeap struct {
	d      *Driver
	layout C.VkDescriptorSetLayout
	pool   C.VkDescriptorPool
	sets   []C.VkDescriptorSet
	ds     []driver.Descriptor

	// Number of descriptors of each type in ds.
	// These values are needed every time that new sets
	// are allocated, so we compute them once.
	nbuf   int
	nimg   int
	nconst int
	ntex   int
	nsplr  int
}

// NewDescHeap creates a new descriptor heap.
func (d *Driver) NewDescHeap(ds []driver.Descriptor) (driver.DescHeap, error) {
	var nbuf, nimg, nconst, ntex, nsplr int
	p := (*C.VkDescriptorSetLayoutBinding)(C.malloc(C.size_t(len(ds)) * C.sizeof_VkDescriptorSetLayoutBinding))
	defer C.free(unsafe.Pointer(p))
	binds := unsafe.Slice(p, len(ds))

	for i := range ds {
		switch ds[i].Type {
		case driver.DBuffer:
			nbuf += ds[i].Len
			binds[i].descriptorType = C.VK_DESCRIPTOR_TYPE_STORAGE_BUFFER
		case driver.DImage:
			nimg += ds[i].Len
			binds[i].descriptorType = C.VK_DESCRIPTOR_TYPE_STORAGE_IMAGE
		case driver.DConstant:
			nconst += ds[i].Len
			binds[i].descriptorType = C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
		case driver.DTexture:
			ntex += ds[i].Len
			binds[i].descriptorType = C.VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE
		case driver.DSampler:
			nsplr += ds[i].Len
			binds[i].descriptorType = C.VK_DESCRIPTOR_TYPE_SAMPLER
		}
		// Descriptor.Nr is the binding number in Vulkan, which must be
		// unique within a descriptor set.
		for j := i + 1; j < len(ds); j++ {
			if ds[i].Nr == ds[j].Nr {
				return nil, errors.New("descriptor number is not unique")
			}
		}
		binds[i].binding = C.uint32_t(ds[i].Nr)
		binds[i].descriptorCount = C.uint32_t(ds[i].Len)
		binds[i].stageFlags = convStage(ds[i].Stages)
		binds[i].pImmutableSamplers = nil
	}

	info := C.VkDescriptorSetLayoutCreateInfo{
		sType:        C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
		bindingCount: C.uint32_t(len(binds)),
		pBindings:    p,
	}
	var layout C.VkDescriptorSetLayout
	err := checkResult(C.vkCreateDescriptorSetLayout(d.dev, &info, nil, &layout))
	if err != nil {
		return nil, err
	}
	// To avoid consuming memory needlessly, neither descHeap.pool
	// nor descHeap.sets are initialized here. Pool creation and
	// descriptor set allocation is left to New.
	return &descHeap{
		d:      d,
		layout: layout,
		ds:     ds,
		nbuf:   nbuf,
		nimg:   nimg,
		nconst: nconst,
		ntex:   ntex,
		nsplr:  nsplr,
	}, nil
}

// New creates enough storage for n copies of each descriptor.
// TODO: Check if using a shared pool improves performance.
func (h *descHeap) New(n int) error {
	switch {
	case n == len(h.sets):
		return nil
	case len(h.sets) == 0:
		// Nothing to destroy/free.
	default:
		C.vkDestroyDescriptorPool(h.d.dev, h.pool, nil)
		C.free(unsafe.Pointer(&h.sets[0]))
		h.sets = nil
		if n == 0 {
			return nil
		}
	}

	// TODO: Consider storing some of this data in descHeap.
	const ntype = 5
	p := (*C.VkDescriptorPoolSize)(C.malloc(ntype * C.sizeof_VkDescriptorPoolSize))
	defer C.free(unsafe.Pointer(p))
	sizes := unsafe.Slice(p, ntype)
	dc := [ntype]struct {
		typ C.VkDescriptorType
		cnt C.uint32_t
	}{
		{C.VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, C.uint32_t(h.nbuf * n)},
		{C.VK_DESCRIPTOR_TYPE_STORAGE_IMAGE, C.uint32_t(h.nimg * n)},
		{C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, C.uint32_t(h.nconst * n)},
		{C.VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE, C.uint32_t(h.ntex * n)},
		{C.VK_DESCRIPTOR_TYPE_SAMPLER, C.uint32_t(h.nsplr * n)},
	}
	nsize := 0
	for i := range dc {
		if dc[i].cnt == 0 {
			continue
		}
		sizes[nsize]._type = dc[i].typ
		sizes[nsize].descriptorCount = dc[i].cnt
		nsize++
	}

	info := C.VkDescriptorPoolCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
		maxSets:       C.uint32_t(n),
		poolSizeCount: C.uint32_t(nsize),
		pPoolSizes:    p,
	}
	var pool C.VkDescriptorPool
	err := checkResult(C.vkCreateDescriptorPool(h.d.dev, &info, nil, &pool))
	if err != nil {
		return err
	}

	// We need two arrays with the same length, one to receive the
	// descriptor set handles and another to indicate which layout
	// to use for each set (they will use the same layout here).
	// Only the pointer to descriptor sets is kept.
	sp := (*C.VkDescriptorSet)(C.malloc(C.size_t(n) * C.sizeof_VkDescriptorSet))
	lp := (*C.VkDescriptorSetLayout)(C.malloc(C.size_t(n) * C.sizeof_VkDescriptorSetLayout))
	defer C.free(unsafe.Pointer(lp))
	layouts := unsafe.Slice(lp, n)
	for i := range layouts {
		layouts[i] = h.layout
	}

	sinfo := C.VkDescriptorSetAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
		descriptorPool:     pool,
		descriptorSetCount: C.uint32_t(n),
		pSetLayouts:        lp,
	}
	err = checkResult(C.vkAllocateDescriptorSets(h.d.dev, &sinfo, sp))
	if err != nil {
		C.vkDestroyDescriptorPool(h.d.dev, pool, nil)
		C.free(unsafe.Pointer(sp))
		return err
	}
	h.pool = pool
	h.sets = unsafe.Slice(sp, n)
	return nil
}

// SetBuffer updates the buffer ranges referred by the given descriptor of
// the given heap copy.
func (h *descHeap) SetBuffer(cpy, nr, start int, buf []driver.Buffer, off, size []int64) {
	p := (*C.VkDescriptorBufferInfo)(C.malloc(C.size_t(len(buf)) * C.sizeof_VkDescriptorBufferInfo))
	defer C.free(unsafe.Pointer(p))
	s := unsafe.Slice(p, len(buf))
	for i := range s {
		s[i] = C.VkDescriptorBufferInfo{
			buffer: buf[i].(*buffer).buf,
			offset: C.VkDeviceSize(off[i]),
			_range: C.VkDeviceSize(size[i]),
		}
	}
	write := C.VkWriteDescriptorSet{
		sType:           C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		dstSet:          h.sets[cpy],
		dstBinding:      C.uint32_t(nr),
		dstArrayElement: C.uint32_t(start),
		descriptorCount: C.uint32_t(len(buf)),
		descriptorType:  h.typeOf(nr),
		pBufferInfo:     p,
	}
	C.vkUpdateDescriptorSets(h.d.dev, 1, &write, 0, nil)
}

// SetImage updates the image views referred by the given descriptor of
// the given heap copy.
func (h *descHeap) SetImage(cpy, nr, start int, iv []driver.ImageView) {
	p := (*C.VkDescriptorImageInfo)(C.malloc(C.size_t(len(iv)) * C.sizeof_VkDescriptorImageInfo))
	defer C.free(unsafe.Pointer(p))
	s := unsafe.Slice(p, len(iv))
	typ := h.typeOf(nr)
	var lay C.VkImageLayout
	if typ == C.VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE {
		lay = C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL
	} else {
		lay = C.VK_IMAGE_LAYOUT_GENERAL
	}
	for i := range s {
		s[i] = C.VkDescriptorImageInfo{
			imageView:   iv[i].(*imageView).view,
			imageLayout: lay,
		}
	}
	write := C.VkWriteDescriptorSet{
		sType:           C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		dstSet:          h.sets[cpy],
		dstBinding:      C.uint32_t(nr),
		dstArrayElement: C.uint32_t(start),
		descriptorCount: C.uint32_t(len(iv)),
		descriptorType:  typ,
		pImageInfo:      p,
	}
	C.vkUpdateDescriptorSets(h.d.dev, 1, &write, 0, nil)
}

// SetSampler updates the samplers referred by the given descriptor of
// the given heap copy.
func (h *descHeap) SetSampler(cpy, nr, start int, splr []driver.Sampler) {
	p := (*C.VkDescriptorImageInfo)(C.malloc(C.size_t(len(splr)) * C.sizeof_VkDescriptorImageInfo))
	defer C.free(unsafe.Pointer(p))
	s := unsafe.Slice(p, len(splr))
	for i := range s {
		s[i] = C.VkDescriptorImageInfo{
			sampler: splr[i].(*sampler).splr,
		}
	}
	write := C.VkWriteDescriptorSet{
		sType:           C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		dstSet:          h.sets[cpy],
		dstBinding:      C.uint32_t(nr),
		dstArrayElement: C.uint32_t(start),
		descriptorCount: C.uint32_t(len(splr)),
		descriptorType:  h.typeOf(nr),
		pImageInfo:      p,
	}
	C.vkUpdateDescriptorSets(h.d.dev, 1, &write, 0, nil)
}

// Count returns the number of heap copies created by New.
func (h *descHeap) Count() int { return len(h.sets) }

// Destroy destroys the descriptor heap.
func (h *descHeap) Destroy() {
	if h == nil {
		return
	}
	if h.d != nil {
		C.vkDestroyDescriptorSetLayout(h.d.dev, h.layout, nil)
		// Note that h.pool is never cleared by New, just replaced.
		if len(h.sets) != 0 {
			C.vkDestroyDescriptorPool(h.d.dev, h.pool, nil)
			C.free(unsafe.Pointer(&h.sets[0]))
		}
	}
	*h = descHeap{}
}

// typeOf returns the VkDescriptorType of the descriptor in h
// identified by the binding descNr.
func (h *descHeap) typeOf(descNr int) C.VkDescriptorType {
	var typ C.VkDescriptorType
	for i := range h.ds {
		if h.ds[i].Nr != descNr {
			continue
		}
		switch h.ds[i].Type {
		case driver.DBuffer:
			typ = C.VK_DESCRIPTOR_TYPE_STORAGE_BUFFER
		case driver.DImage:
			typ = C.VK_DESCRIPTOR_TYPE_STORAGE_IMAGE
		case driver.DConstant:
			typ = C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
		case driver.DTexture:
			typ = C.VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE
		case driver.DSampler:
			typ = C.VK_DESCRIPTOR_TYPE_SAMPLER
		}
		break
	}
	return typ
}

// descTable implements driver.DescTable.
type descTable struct {
	d      *Driver
	h      []*descHeap
	layout C.VkPipelineLayout
}

// NewDescTable creates a new descriptor table.
func (d *Driver) NewDescTable(dh []driver.DescHeap) (driver.DescTable, error) {
	h := make([]*descHeap, len(dh))
	for i := range h {
		h[i] = dh[i].(*descHeap)
	}
	var p *C.VkDescriptorSetLayout
	if len(h) > 0 {
		p = (*C.VkDescriptorSetLayout)(C.malloc(C.size_t(len(h)) * C.sizeof_VkDescriptorSetLayout))
		defer C.free(unsafe.Pointer(p))
		sl := unsafe.Slice(p, len(h))
		for i := range h {
			sl[i] = h[i].layout
		}
	}
	info := C.VkPipelineLayoutCreateInfo{
		sType:          C.VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
		setLayoutCount: C.uint32_t(len(h)),
		pSetLayouts:    p,
	}
	var layout C.VkPipelineLayout
	err := checkResult(C.vkCreatePipelineLayout(d.dev, &info, nil, &layout))
	if err != nil {
		return nil, err
	}
	return &descTable{
		d:      d,
		h:      h,
		layout: layout,
	}, nil
}

// Destroy destroys the descriptor table.
func (t *descTable) Destroy() {
	if t == nil {
		return
	}
	if t.d != nil {
		C.vkDestroyPipelineLayout(t.d.dev, t.layout, nil)
	}
	*t = descTable{}
}

// convStage converts a driver.Stage to a VkShaderStageFlags.
func convStage(stg driver.Stage) (flags C.VkShaderStageFlags) {
	if stg&driver.SVertex != 0 {
		flags |= C.VK_SHADER_STAGE_VERTEX_BIT
	}
	if stg&driver.SFragment != 0 {
		flags |= C.VK_SHADER_STAGE_FRAGMENT_BIT
	}
	if stg&driver.SCompute != 0 {
		flags |= C.VK_SHADER_STAGE_COMPUTE_BIT
	}
	return
}
