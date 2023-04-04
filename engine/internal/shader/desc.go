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

const (
	globalHeap = iota
	drawableHeap
	materialHeap
	jointHeap

	maxHeap
)

const (
	frameNr    = 0
	lightNr    = 1
	shadowNr   = 2
	shdwTexNr  = 3
	shdwSplrNr = 4

	drawableNr = 0

	materialNr  = 0
	colorTexNr  = 1
	colorSplrNr = 2
	metalTexNr  = 3
	metalSplrNr = 4
	normTexNr   = 5
	normSplrNr  = 6
	occTexNr    = 7
	occSplrNr   = 8
	emisTexNr   = 9
	emisSplrNr  = 10

	jointNr = 0
)

// These spans are given in number of blocks.
// Each block has blockSize bytes.
const (
	blockSize = 256

	frameSpan    = (unsafe.Sizeof(FrameLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	lightSpan    = (MaxLight*unsafe.Sizeof(LightLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	shadowSpan   = (MaxShadow*unsafe.Sizeof(ShadowLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	drawableSpan = (unsafe.Sizeof(DrawableLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	materialSpan = (unsafe.Sizeof(MaterialLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
	jointSpan    = (MaxJoint*unsafe.Sizeof(JointLayout{}) + blockSize - 1) &^ (blockSize - 1) / blockSize
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
		constantDesc(frameNr),
		// Light.
		constantDesc(lightNr),
		// Shadow.
		constantDesc(shadowNr),
		// Shadow map.
		// TODO: Texture array.
		textureDesc(shdwTexNr),
		samplerDesc(shdwSplrNr),
	})
}

// newDescHeap1 creates a new driver.DescHeap suitable for
// drawable (DrawableLayout) data.
func newDescHeap1() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(drawableNr),
	})
}

// newDescHeap2 creates a new driver.DescHeap suitable for
// material (MaterialLayout) data plus texture/samplers.
func newDescHeap2() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(materialNr),
		// Base color.
		textureDesc(colorTexNr),
		samplerDesc(colorSplrNr),
		// Metallic-roughness.
		textureDesc(metalTexNr),
		samplerDesc(metalSplrNr),
		// Normal map.
		textureDesc(normTexNr),
		samplerDesc(normSplrNr),
		// Occlusion map.
		textureDesc(occTexNr),
		samplerDesc(occSplrNr),
		// Emissive map.
		textureDesc(emisTexNr),
		samplerDesc(emisSplrNr),
	})
}

// newDescHeap3 creates a new driver.DescHeap suitable for
// joint (JointLayout) data.
func newDescHeap3() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(jointNr),
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
	coff [maxHeap]int64
}

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
		//	0 | FrameLayout
		//	1 | [MaxLight]LightLayout
		//	2 | [MaxShadow]ShadowLayout
		dh = t.dt.Heap(globalHeap)
		n = dh.Len()
		for i := 0; i < n; i++ {
			sz[0] = int64(frameSpan * blockSize)
			dh.SetBuffer(i, frameNr, 0, buf, off, sz)
			off[0] += sz[0]
			sz[0] = int64(lightSpan * blockSize)
			dh.SetBuffer(i, lightNr, 0, buf, off, sz)
			off[0] += sz[0]
			sz[0] = int64(shadowSpan * blockSize)
			dh.SetBuffer(i, shadowNr, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Drawable heap constants:
		//	0 | DrawableLayout
		dh = t.dt.Heap(drawableHeap)
		n = dh.Len()
		t.coff[drawableHeap] = off[0]
		sz[0] = int64(drawableSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, drawableNr, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Material heap constants:
		//	0 | MaterialLayout
		dh = t.dt.Heap(materialHeap)
		n = dh.Len()
		t.coff[materialHeap] = off[0]
		sz[0] = int64(materialSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, materialNr, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Joint heap constants:
		//	0 | [MaxJoint]JointLayout
		dh = t.dt.Heap(jointHeap)
		n = dh.Len()
		t.coff[jointHeap] = off[0]
		sz[0] = int64(jointSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, jointNr, 0, buf, off, sz)
			off[0] += sz[0]
		}
	}

	pbuf := t.cbuf
	poff := t.coff[globalHeap]
	t.cbuf = buf
	t.coff[globalHeap] = off
	return pbuf, poff
}

// SetShadowMap sets a shadow texture/sampler pair in the
// global heap.
// tex.Image() must support driver.UShaderSample.
func (t *Table) SetShadowMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.dt.Heap(globalHeap).SetImage(cpy, shdwTexNr, 0, []driver.ImageView{tex})
	t.dt.Heap(globalHeap).SetSampler(cpy, shdwSplrNr, 0, []driver.Sampler{splr})
}

// SetBaseColor sets a base color texture/sampler pair in
// the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *Table) SetBaseColor(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.dt.Heap(materialHeap).SetImage(cpy, colorTexNr, 0, []driver.ImageView{tex})
	t.dt.Heap(materialHeap).SetSampler(cpy, colorSplrNr, 0, []driver.Sampler{splr})
}

// SetMetalRough sets a metallic-roughness texture/sampler
// pair in the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *Table) SetMetalRough(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.dt.Heap(materialHeap).SetImage(cpy, metalTexNr, 0, []driver.ImageView{tex})
	t.dt.Heap(materialHeap).SetSampler(cpy, metalSplrNr, 0, []driver.Sampler{splr})
}

// SetNormalMap sets a normal texture/sampler pair in the
// material heap.
// tex.Image() must support driver.UShaderSample.
func (t *Table) SetNormalMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.dt.Heap(materialHeap).SetImage(cpy, normTexNr, 0, []driver.ImageView{tex})
	t.dt.Heap(materialHeap).SetSampler(cpy, normSplrNr, 0, []driver.Sampler{splr})
}

// SetOcclusionMap sets an occlusion texture/sampler pair
// in the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *Table) SetOcclusionMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.dt.Heap(materialHeap).SetImage(cpy, occTexNr, 0, []driver.ImageView{tex})
	t.dt.Heap(materialHeap).SetSampler(cpy, occSplrNr, 0, []driver.Sampler{splr})
}

// SetEmissiveMap sets an emissive texture/sampler pair in
// the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *Table) SetEmissiveMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.dt.Heap(materialHeap).SetImage(cpy, emisTexNr, 0, []driver.ImageView{tex})
	t.dt.Heap(materialHeap).SetSampler(cpy, emisSplrNr, 0, []driver.Sampler{splr})
}

// Frame returns a pointer to GPU memory mapping to a
// given FrameLayout of the global heap.
// A valid constant buffer must be set at the time this
// method is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *Table) Frame(cpy int) *FrameLayout {
	off := t.coff[globalHeap] + int64(frameSpan+lightSpan+shadowSpan)*blockSize*int64(cpy)
	b := t.cbuf.Bytes()[off:]
	return (*FrameLayout)(unsafe.Pointer(unsafe.SliceData(b)))
}

// Light returns a pointer to GPU memory mapping to a
// given LightLayout array of the global heap.
// A valid constant buffer must be set at the time this
// method is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *Table) Light(cpy int) *[MaxLight]LightLayout {
	off := t.coff[globalHeap] + int64(frameSpan+lightSpan+shadowSpan)*blockSize*int64(cpy)
	off += int64(frameSpan) * blockSize
	b := t.cbuf.Bytes()[off:]
	return (*[MaxLight]LightLayout)(unsafe.Pointer(unsafe.SliceData(b)))
}

// Shadow returns a pointer to GPU memory mapping to a
// given ShadowLayout array of the global heap.
// A valid constant buffer must be set at the time this
// method is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *Table) Shadow(cpy int) *[MaxShadow]ShadowLayout {
	off := t.coff[globalHeap] + int64(frameSpan+lightSpan+shadowSpan)*blockSize*int64(cpy)
	off += int64(frameSpan+lightSpan) * blockSize
	b := t.cbuf.Bytes()[off:]
	return (*[MaxShadow]ShadowLayout)(unsafe.Pointer(unsafe.SliceData(b)))
}

// Drawable returns a pointer to GPU memory mapping to a
// given DrawableLayout of the drawable heap.
// A valid constant buffer must be set at the time this
// method is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *Table) Drawable(cpy int) *DrawableLayout {
	off := t.coff[drawableHeap] + int64(drawableSpan)*blockSize*int64(cpy)
	b := t.cbuf.Bytes()[off:]
	return (*DrawableLayout)(unsafe.Pointer(unsafe.SliceData(b)))
}

// Material returns a pointer to GPU memory mapping to a
// given MaterialLayout of the material heap.
// A valid constant buffer must be set at the time this
// method is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *Table) Material(cpy int) *MaterialLayout {
	off := t.coff[materialHeap] + int64(materialSpan)*blockSize*int64(cpy)
	b := t.cbuf.Bytes()[off:]
	return (*MaterialLayout)(unsafe.Pointer(unsafe.SliceData(b)))
}

// Joint returns a pointer to GPU memory mapping to a
// given JointLayout array of the joint heap.
// A valid constant buffer must be set at the time this
// method is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *Table) Joint(cpy int) *[MaxJoint]JointLayout {
	off := t.coff[jointHeap] + int64(jointSpan)*blockSize*int64(cpy)
	b := t.cbuf.Bytes()[off:]
	return (*[MaxJoint]JointLayout)(unsafe.Pointer(unsafe.SliceData(b)))
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
