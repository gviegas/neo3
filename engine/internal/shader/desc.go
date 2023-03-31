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
	dt driver.DescTable
	// TODO: Buffer; bitmap.
}

const (
	globalHeap = iota
	drawableHeap
	materialHeap
	jointHeap
)

// NewTable creates a new descriptor table.
func NewTable() (*Table, error) {
	dt, err := newDescTable()
	if err != nil {
		return nil, err
	}
	return &Table{dt}, nil
}

func (t *Table) heapAlloc(idx, n int) error {
	// TODO: Buffer.
	n += t.dt.Heap(idx).Len()
	return t.dt.Heap(idx).New(n)
}

// AllocGlobal allocates n copies in the heap of
// global data.
// This heap stores frame, light and shadow descriptors.
//
// One should need exactly one global heap copy per
// in-flight frame.
func (t *Table) AllocGlobal(n int) error { return t.heapAlloc(globalHeap, n) }

// AllocDrawable allocates n copies in the heap of
// drawable data.
// This heap only stores drawable descriptors.
//
// This data is expected to be highly dynamic. One may
// want to retain one drawable slot per in-flight frame
// for every drawable entity.
func (t *Table) AllocDrawable(n int) error { return t.heapAlloc(drawableHeap, n) }

// AllocMaterial allocates n copies in the heap of
// material data.
// This heap only stores material descriptors.
//
// This data is expected to remain unchanged in the
// common case. One should attempt to share it between
// in-flight frames as much as possible.
func (t *Table) AllocMaterial(n int) error { return t.heapAlloc(materialHeap, n) }

// AllocJoint allocates n copies in the heap of
// joint data.
// This heap only stores joint descriptors.
//
// This data may be used for more than one drawable in
// certain cases, so it is not allocated alongside
// drawable descriptors (for now).
func (t *Table) AllocJoint(n int) error { return t.heapAlloc(jointHeap, n) }

// Free invalidates t and destroys the driver resources.
func (t *Table) Free() {
	if t.dt != nil {
		freeDescTable(t.dt)
	}
	*t = Table{}
}
