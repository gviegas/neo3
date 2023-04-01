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
	"github.com/gviegas/scene/internal/bitm"
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
	cmap bitm.Bitm[uint32]
}

const (
	globalHeap = iota
	drawableHeap
	materialHeap
	jointHeap
)

const (
	blockSize = 256
	nbit      = 32
)

// These spans are given in number of blocks.
// Each block has blockSize bytes.
const (
	frameSpan    = (unsafe.Sizeof(FrameLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	lightSpan    = (MaxLights*unsafe.Sizeof(LightLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	shadowSpan   = (MaxShadows*unsafe.Sizeof(ShadowLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	drawableSpan = (unsafe.Sizeof(DrawableLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	materialSpan = (unsafe.Sizeof(MaterialLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	jointSpan    = (MaxJoints*unsafe.Sizeof(JointLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
)

// NewTable creates a new descriptor table.
func NewTable() (*Table, error) {
	dt, err := newDescTable()
	if err != nil {
		return nil, err
	}
	return &Table{dt, nil, bitm.Bitm[uint32]{}}, nil
}

func (t *Table) heapAlloc(idx, n int) error {
	switch {
	// TODO: Consider deallocating when n is 0.
	case n == 0, n == t.dt.Heap(idx).Len():
		return nil
	case n < 0:
		panic("descriptor heap allocation with negative count")
	}
	if err := t.dt.Heap(idx).New(n); err != nil {
		return err
	}
	if err := t.constAlloc(idx, n); err != nil {
		t.dt.Heap(idx).New(0)
		return err
	}
	return nil
}

func (t *Table) constAlloc(idx, n int) error {
	// TODO: Unset range for this heap.
	switch idx {
	case globalHeap:
		n *= int(frameSpan + lightSpan + shadowSpan)
	case drawableHeap:
		n *= int(drawableSpan)
	case materialHeap:
		n *= int(materialSpan)
	case jointHeap:
		n *= int(jointSpan)
	default:
		// Should never happen.
		panic("not a valid heap index")
	}
	// TODO: Set range for this heap.
	_, ok := t.cmap.SearchRange(n)
	if !ok {
		// TODO: This can lead to large spans
		// of unused buffer memory (e.g., when
		// resizing a heap that is not in the
		// end of the buffer).
		// Contents should be moved within the
		// buffer when that happens.
		var sz int64
		var nplus int
		if t.cbuf != nil {
			sz = (t.cbuf.Cap()/blockSize + int64(n) + nbit - 1) &^ (nbit - 1)
			nplus = (int(sz) - t.cmap.Len()) / nbit
			sz *= blockSize
		} else {
			nplus = (n + nbit - 1) &^ (nbit - 1)
			sz = int64(nplus) * blockSize
			nplus /= nbit
		}
		cbuf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
		if err != nil {
			return err
		}
		if t.cbuf != nil {
			// TODO: Consider doing this copy
			// through the GPU.
			copy(cbuf.Bytes(), t.cbuf.Bytes())
			t.cbuf.Destroy()
		}
		t.cbuf = cbuf
		_ = t.cmap.Grow(nplus)
	}
	return nil
}

// AllocGlobal resizes the global heap to contain n
// copies of global descriptors. It also allocates
// buffer memory as necessary to store the constants
// of each copy.
//
// This heap stores frame, light and shadow descriptors.
//
// One should need exactly one global heap copy per
// in-flight frame.
func (t *Table) AllocGlobal(n int) error { return t.heapAlloc(globalHeap, n) }

// AllocDrawable resizes the drawable heap to contain
// n copies of drawable descriptors. It also allocates
// buffer memory as necessary to store the constants
// of each copy.
//
// This heap only stores drawable descriptors.
//
// This data is expected to be highly dynamic. One may
// want to retain one drawable slot per in-flight frame
// for every drawable entity.
func (t *Table) AllocDrawable(n int) error { return t.heapAlloc(drawableHeap, n) }

// AllocMaterial resizes the material heap to contain
// n copies of material descriptors. It also allocates
// buffer memory as necessary to store the constants
// of each copy.
//
// This heap only stores material descriptors.
//
// This data is expected to remain unchanged in the
// common case. One should attempt to share it between
// in-flight frames as much as possible.
func (t *Table) AllocMaterial(n int) error { return t.heapAlloc(materialHeap, n) }

// Allocjoint resizes the joint heap to contain n
// copies of joint descriptors. It also allocates
// buffer memory as necessary to store the constants
// of each copy.
//
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
	if t.cbuf != nil {
		t.cbuf.Destroy()
	}
	*t = Table{}
}
