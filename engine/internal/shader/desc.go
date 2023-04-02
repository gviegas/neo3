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
	coff int64
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
	return &Table{dt: dt}, nil
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

// SetConstBuf sets the buffer for constant descriptors.
// This buffer must be host visible and must have been
// created with the driver.UShaderConst usage flag.
// The constants will consume exactly t.ConstSize()
// bytes from buf, starting at offset off (the caller
// must ensure that this range is within bounds).
// off must be aligned to 256 bytes.
// It returns the previously set buffer/offset, if any.
func (t *Table) SetConstBuf(buf driver.Buffer, off int64) (driver.Buffer, int64) {
	switch {
	case buf == nil:
		off = 0

	case off&(blockSize-1) != 0:
		panic("misaligned constant buffer offset")

	case buf.Cap()-off < int64(t.ConstSize()):
		panic("constant buffer range out of bounds")

	default:
		var dh driver.DescHeap
		var n int
		buf, off, sz := []driver.Buffer{buf}, []int64{off}, []int64{0}

		// Global heap constants:
		//	* FrameLayout
		//	* [MaxLights]LightLayout
		//	* [MaxShadows]ShadowLayout
		dh = t.dt.Heap(globalHeap)
		n = dh.Len()
		for i := 0; i < n; i++ {
			sz[0] = int64(frameSpan * blockSize)
			dh.SetBuffer(i, 0, 0, buf, off, sz)
			off[0] += sz[0]
			sz[0] = int64(lightSpan * blockSize)
			dh.SetBuffer(i, 1, 0, buf, off, sz)
			off[0] += sz[0]
			sz[0] = int64(shadowSpan * blockSize)
			dh.SetBuffer(i, 2, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Drawable heap constants:
		//	* DrawableLayout
		dh = t.dt.Heap(drawableHeap)
		n = dh.Len()
		sz[0] = int64(drawableSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, 0, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Material heap constants:
		//	* MaterialLayout
		dh = t.dt.Heap(materialHeap)
		n = dh.Len()
		sz[0] = int64(materialSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, 0, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Joint heap constants:
		//	* [MaxJoints]JointLayout
		dh = t.dt.Heap(jointHeap)
		n = dh.Len()
		sz[0] = int64(jointSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, 0, 0, buf, off, sz)
			off[0] += sz[0]
		}
	}

	pbuf := t.cbuf
	poff := t.coff
	t.cbuf = buf
	t.coff = off
	return pbuf, poff
}

// Free invalidates t and destroys the driver resources.
//
// NOTE: The constant buffer is not destroyed by this
// method; one can retrieve the buffer by calling
// t.SetConstBuf(nil, _) prior to calling t.Free.
func (t *Table) Free() {
	if t.dt != nil {
		freeDescTable(t.dt)
	}
	*t = Table{}
}
