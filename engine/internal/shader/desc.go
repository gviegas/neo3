// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Descriptor management.
//
// For portability, the following restrictions apply:
//
//	DescHeap per DescTable           | 4 (max)
//	DTexture/DSampler descriptors    | 16 (max)
//	DConstant descriptors            | 12 (max)
//	DImage/DBuffer descriptors       | 4 (max)
//	DConstant/DBuffer data alignment | 256 bytes (min)
//	DConstant/DBuffer data size      | 16 KiB (max)
//
// (the above names refer to the driver package).

package shader

import (
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
)

func constantDesc(nr int) driver.Descriptor {
	return driver.Descriptor{
		Type:   driver.DConstant,
		Stages: driver.SVertex | driver.SFragment,
		Nr:     nr,
		Len:    1,
	}
}

func textureDesc(nr int) driver.Descriptor {
	return driver.Descriptor{
		Type:   driver.DTexture,
		Stages: driver.SVertex | driver.SFragment,
		Nr:     nr,
		Len:    1,
	}
}

func samplerDesc(nr int) driver.Descriptor {
	return driver.Descriptor{
		Type:   driver.DSampler,
		Stages: driver.SVertex | driver.SFragment,
		Nr:     nr,
		Len:    1,
	}
}

// newDescHeap0 creates a new driver.DescHeap suitable for
// frame (FrameLayout), light (LightLayout) and shadow
// (ShadowLayout) data plus textures/samplers.
func newDescHeap0() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		// Frame.
		constantDesc(0),
		// Light.
		constantDesc(1),
		// Shadow.
		constantDesc(2),
		// Shadow map.
		// TODO: Texture array.
		textureDesc(3), samplerDesc(4),
	})
}

// newDescHeap1 creates a new driver.DescHeap suitable for
// drawable (DrawableLayout) data.
func newDescHeap1() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(0),
	})
}

// newDescHeap2 creates a new driver.DescHeap suitable for
// material (MaterialLayout) data plus texture/samplers.
func newDescHeap2() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(0),
		// Base color.
		textureDesc(1), samplerDesc(2),
		// Metallic-roughness.
		textureDesc(3), samplerDesc(4),
		// Normal map.
		textureDesc(5), samplerDesc(6),
		// Occlusion map.
		textureDesc(7), samplerDesc(8),
		// Emissive map.
		textureDesc(9), samplerDesc(10),
	})
}

// newDescHeap3 creates a new driver.DescHeap suitable for
// joint (JointLayout) data.
func newDescHeap3() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(0),
	})
}

// newDescTable creates a new driver.DescTable containing
// the heaps 0-3.
func newDescTable() (driver.DescTable, error) {
	dh0, err := newDescHeap0()
	if err != nil {
		return nil, err
	}
	dh1, err := newDescHeap1()
	if err != nil {
		dh0.Destroy()
		return nil, err
	}
	dh2, err := newDescHeap2()
	if err != nil {
		dh0.Destroy()
		dh1.Destroy()
		return nil, err
	}
	dh3, err := newDescHeap3()
	if err != nil {
		dh0.Destroy()
		dh1.Destroy()
		dh2.Destroy()
		return nil, err
	}
	return ctxt.GPU().NewDescTable([]driver.DescHeap{
		dh0,
		dh1,
		dh2,
		dh3,
	})
}

// freeDescTable destroys a driver.DescTable and every
// driver.DescHeap that it contains.
func freeDescTable(dt driver.DescTable) {
	dhs := make([]driver.DescHeap, dt.Len())
	for i := range dhs {
		dhs[i] = dt.Heap(i)
	}
	dt.Destroy()
	for i := range dhs {
		dhs[i].Destroy()
	}
}

// Table manages descriptor usage within a single
// driver.DescTable.
type Table struct {
	dt   driver.DescTable
	cbuf driver.Buffer
}

const (
	globalHeap = iota
	drawableHeap
	materialHeap
	jointHeap
)

// These spans are given in number of blocks.
// Each block has blockSize bytes.
const (
	blockSize = 256

	frameSpan    = (unsafe.Sizeof(FrameLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	lightSpan    = (MaxLights*unsafe.Sizeof(LightLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	shadowSpan   = (MaxShadows*unsafe.Sizeof(ShadowLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	drawableSpan = (unsafe.Sizeof(DrawableLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	materialSpan = (unsafe.Sizeof(MaterialLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	jointSpan    = (MaxJoints*unsafe.Sizeof(JointLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
)

// NewTable creates a new descriptor table.
// Each parameter defines the number of heap copies to
// allocate for a given heap. Currently, the heaps are
// organized as follows:
//
//	global heap   | frame/light/shadow descriptors
//	drawable heap | drawable descriptors
//	material heap | material descriptors
//	joint heap    | joint descriptors
//
// For constant descriptors that are defined as static
// arrays in shaders, every heap copy will require
// enough buffer memory to store the whole array.
func NewTable(globalN, drawableN, materialN, jointN int) (*Table, error) {
	dt, err := newDescTable()
	if err != nil {
		return nil, err
	}
	// NOTE: The order here must match the
	// heap indices.
	for i, n := range [4]int{globalN, drawableN, materialN, jointN} {
		if n < 0 {
			panic("descriptor heap allocation with negative count")
		}
		if err := dt.Heap(i).New(n); err != nil {
			return nil, err
		}
	}
	return &Table{dt, nil}, nil
}

// ConstSize returns the number of bytes consumed by
// all constant descriptors of t.
func (t *Table) ConstSize() int {
	spn0 := t.dt.Heap(globalHeap).Len() * int(frameSpan+lightSpan+shadowSpan)
	spn1 := t.dt.Heap(drawableHeap).Len() * int(drawableSpan)
	spn2 := t.dt.Heap(materialHeap).Len() * int(materialSpan)
	spn3 := t.dt.Heap(jointHeap).Len() * int(jointSpan)
	return (spn0 + spn1 + spn2 + spn3) * blockSize
}

// Free invalidates t and destroys the driver resources.
func (t *Table) Free() {
	if t.dt != nil {
		freeDescTable(t.dt)
	}
	//if t.cbuf != nil {
	//	t.cbuf.Destroy()
	//}
	*t = Table{}
}
