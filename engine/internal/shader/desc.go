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

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
)

const (
	GlobalHeap = iota
	DrawableHeap
	MaterialHeap
	JointHeap

	maxHeap
)

const (
	frameNr     = 0
	lightNr     = 1
	shadowNr    = 2
	shdwTexNr   = 3
	shdwSplrNr  = 4
	irradTexNr  = 5
	irradSplrNr = 6
	ldTexNr     = 7
	ldSplrNr    = 8
	dfgTexNr    = 9
	dfgSplrNr   = 10

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

func constantDesc(nr int, stages driver.Stage) driver.Descriptor {
	return driver.Descriptor{
		Type:   driver.DConstant,
		Stages: stages,
		Nr:     nr,
		Len:    1,
	}
}

func textureDesc(nr int, stages driver.Stage) driver.Descriptor {
	return driver.Descriptor{
		Type:   driver.DTexture,
		Stages: stages,
		Nr:     nr,
		Len:    1,
	}
}

func samplerDesc(nr int, stages driver.Stage) driver.Descriptor {
	return driver.Descriptor{
		Type:   driver.DSampler,
		Stages: stages,
		Nr:     nr,
		Len:    1,
	}
}

// newGlobalHeap creates a new driver.DescHeap suitable
// for frame (FrameLayout), light (LightLayout) and
// shadow (ShadowLayout) data plus textures/samplers.
//
// TODO: Texture arrays.
func newGlobalHeap() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		// TODO: driver.SCompute may be necessary.
		constantDesc(frameNr, driver.SVertex|driver.SFragment),
		constantDesc(lightNr, driver.SFragment),
		constantDesc(shadowNr, driver.SVertex|driver.SFragment),
		textureDesc(shdwTexNr, driver.SFragment),
		samplerDesc(shdwSplrNr, driver.SFragment),
		textureDesc(irradTexNr, driver.SFragment),
		samplerDesc(irradSplrNr, driver.SFragment),
		textureDesc(ldTexNr, driver.SFragment),
		samplerDesc(ldSplrNr, driver.SFragment),
		textureDesc(dfgTexNr, driver.SFragment),
		samplerDesc(dfgSplrNr, driver.SFragment),
	})
}

// newDrawableHeap creates a new driver.DescHeap suitable
// for drawable (DrawableLayout) data.
func newDrawableHeap() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		// TODO: driver.SFragment may be unnecessary.
		constantDesc(drawableNr, driver.SVertex|driver.SFragment),
	})
}

// newMaterialHeap creates a new driver.DescHeap suitable
// for material (MaterialLayout) data plus
// textures/samplers.
func newMaterialHeap() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(materialNr, driver.SFragment),
		textureDesc(colorTexNr, driver.SFragment),
		samplerDesc(colorSplrNr, driver.SFragment),
		textureDesc(metalTexNr, driver.SFragment),
		samplerDesc(metalSplrNr, driver.SFragment),
		textureDesc(normTexNr, driver.SFragment),
		samplerDesc(normSplrNr, driver.SFragment),
		textureDesc(occTexNr, driver.SFragment),
		samplerDesc(occSplrNr, driver.SFragment),
		textureDesc(emisTexNr, driver.SFragment),
		samplerDesc(emisSplrNr, driver.SFragment),
	})
}

// newJointHeap creates a new driver.DescHeap suitable
// for joint (JointLayout) data.
func newJointHeap() (driver.DescHeap, error) {
	return ctxt.GPU().NewDescHeap([]driver.Descriptor{
		constantDesc(jointNr, driver.SVertex),
	})
}

// newDrawTable creates a new driver.DescTable containing
// the global, drawable, material and joint heaps.
func newDrawTable() (driver.DescTable, error) {
	var heaps [maxHeap]driver.DescHeap
	for i, f := range [maxHeap]func() (driver.DescHeap, error){
		GlobalHeap:   newGlobalHeap,
		DrawableHeap: newDrawableHeap,
		MaterialHeap: newMaterialHeap,
		JointHeap:    newJointHeap,
	} {
		dh, err := f()
		if err != nil {
			for j := range i {
				heaps[j].Destroy()
			}
			return nil, err
		}
		heaps[i] = dh
	}
	return ctxt.GPU().NewDescTable(heaps[:])
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

// DrawTable manages descriptor usage within a single
// driver.DescTable.
type DrawTable struct {
	dt driver.DescTable
	// Cached heap copy counts.
	// These counts will not change during the
	// lifetime of the table, so this avoids
	// having to query the driver needlessly.
	dcpy [maxHeap]int
	cbuf driver.Buffer
	// Offsets into the constant buffer.
	// The location of heap data in the buffer
	// is ordered by heap index, and within a
	// heap, by heap copy index. Every copy of
	// a lower numbered heap comes before the
	// copies of subsequent heaps.
	coff [maxHeap]int64
	// Cached cbuf.Bytes().
	// Note that it has no offset applied.
	cs []byte
}

// NewDrawTable creates a new descriptor table.
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
func NewDrawTable(globalN, drawableN, materialN, jointN int) (*DrawTable, error) {
	dt, err := newDrawTable()
	if err != nil {
		return nil, err
	}
	// NOTE: The order here must match the
	// heap indices.
	dcpy := [maxHeap]int{globalN, drawableN, materialN, jointN}
	for i, n := range dcpy {
		if n < 0 {
			panic("descriptor heap allocation with negative count")
		}
		if err := dt.Heap(i).New(n); err != nil {
			return nil, err
		}
	}
	return &DrawTable{dt: dt, dcpy: dcpy}, nil
}

// SetGraph calls cb.SetDescTableGraph to set the given
// heap copies.
// cb must be recording commands.
func (t *DrawTable) SetGraph(cb driver.CmdBuffer, start int, cpy []int) {
	if start < GlobalHeap || start > JointHeap || len(cpy) == 0 || len(cpy) > maxHeap-start {
		panic("invalid descriptor heap indexing")
	}
	for i, x := range cpy {
		t.validateHeapCopy(start+i, x)
	}
	cb.SetDescTableGraph(t.dt, start, cpy)
}

// ConstSize returns the number of bytes consumed by
// all constant descriptors of t.
func (t *DrawTable) ConstSize() int {
	spn0 := t.dt.Heap(GlobalHeap).Len() * int(frameSpan+lightSpan+shadowSpan)
	spn1 := t.dt.Heap(DrawableHeap).Len() * int(drawableSpan)
	spn2 := t.dt.Heap(MaterialHeap).Len() * int(materialSpan)
	spn3 := t.dt.Heap(JointHeap).Len() * int(jointSpan)
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
func (t *DrawTable) SetConstBuf(buf driver.Buffer, off int64) (driver.Buffer, int64) {
	var cs []byte
	switch {
	case buf == nil:
		off = 0

	case off&(blockSize-1) != 0:
		panic("misaligned constant buffer offset")

	case buf.Cap()-off < int64(t.ConstSize()):
		panic("constant buffer range out of bounds")

	default:
		cs = buf.Bytes()
		var dh driver.DescHeap
		var n int
		buf, off, sz := []driver.Buffer{buf}, []int64{off}, []int64{0}

		// Global heap constants:
		//	0 | FrameLayout
		//	1 | [MaxLight]LightLayout
		//	2 | [MaxShadow]ShadowLayout
		dh = t.dt.Heap(GlobalHeap)
		n = dh.Len()
		for i := range n {
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
		dh = t.dt.Heap(DrawableHeap)
		n = dh.Len()
		t.coff[DrawableHeap] = off[0]
		sz[0] = int64(drawableSpan * blockSize)
		for i := range n {
			dh.SetBuffer(i, drawableNr, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Material heap constants:
		//	0 | MaterialLayout
		dh = t.dt.Heap(MaterialHeap)
		n = dh.Len()
		t.coff[MaterialHeap] = off[0]
		sz[0] = int64(materialSpan * blockSize)
		for i := 0; i < n; i++ {
			dh.SetBuffer(i, materialNr, 0, buf, off, sz)
			off[0] += sz[0]
		}

		// Joint heap constants:
		//	0 | [MaxJoint]JointLayout
		dh = t.dt.Heap(JointHeap)
		n = dh.Len()
		t.coff[JointHeap] = off[0]
		sz[0] = int64(jointSpan * blockSize)
		for i := range n {
			dh.SetBuffer(i, jointNr, 0, buf, off, sz)
			off[0] += sz[0]
		}
	}

	pbuf := t.cbuf
	poff := t.coff[GlobalHeap]
	t.cbuf = buf
	t.coff[GlobalHeap] = off
	t.cs = cs
	return pbuf, poff
}

// SetShadowMap sets a shadow texture/sampler pair in the
// global heap.
// tex.Image() must support driver.UShaderSample.
// splr must support depth comparison.
func (t *DrawTable) SetShadowMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(GlobalHeap, cpy, tex, splr)
	t.dt.Heap(GlobalHeap).SetImage(cpy, shdwTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(GlobalHeap).SetSampler(cpy, shdwSplrNr, 0, []driver.Sampler{splr})
}

// SetIrradiance sets a diffuse irradiance texture/sampler
// pair in the global heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetIrradiance(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(GlobalHeap, cpy, tex, splr)
	t.dt.Heap(GlobalHeap).SetImage(cpy, irradTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(GlobalHeap).SetSampler(cpy, irradSplrNr, 0, []driver.Sampler{splr})
}

// SetLD sets a specular LD texture/sampler pair in the
// global heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetLD(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(GlobalHeap, cpy, tex, splr)
	t.dt.Heap(GlobalHeap).SetImage(cpy, ldTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(GlobalHeap).SetSampler(cpy, ldSplrNr, 0, []driver.Sampler{splr})
}

// SetDFG sets a specular DFG texture/sampler pair in
// the global heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetDFG(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(GlobalHeap, cpy, tex, splr)
	t.dt.Heap(GlobalHeap).SetImage(cpy, dfgTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(GlobalHeap).SetSampler(cpy, dfgSplrNr, 0, []driver.Sampler{splr})
}

// SetBaseColor sets a base color texture/sampler pair in
// the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetBaseColor(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(MaterialHeap, cpy, tex, splr)
	t.dt.Heap(MaterialHeap).SetImage(cpy, colorTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(MaterialHeap).SetSampler(cpy, colorSplrNr, 0, []driver.Sampler{splr})
}

// SetMetalRough sets a metallic-roughness texture/sampler
// pair in the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetMetalRough(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(MaterialHeap, cpy, tex, splr)
	t.dt.Heap(MaterialHeap).SetImage(cpy, metalTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(MaterialHeap).SetSampler(cpy, metalSplrNr, 0, []driver.Sampler{splr})
}

// SetNormalMap sets a normal texture/sampler pair in the
// material heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetNormalMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(MaterialHeap, cpy, tex, splr)
	t.dt.Heap(MaterialHeap).SetImage(cpy, normTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(MaterialHeap).SetSampler(cpy, normSplrNr, 0, []driver.Sampler{splr})
}

// SetOcclusionMap sets an occlusion texture/sampler pair
// in the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetOcclusionMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(MaterialHeap, cpy, tex, splr)
	t.dt.Heap(MaterialHeap).SetImage(cpy, occTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(MaterialHeap).SetSampler(cpy, occSplrNr, 0, []driver.Sampler{splr})
}

// SetEmissiveMap sets an emissive texture/sampler pair in
// the material heap.
// tex.Image() must support driver.UShaderSample.
func (t *DrawTable) SetEmissiveMap(cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateTexSplr(MaterialHeap, cpy, tex, splr)
	t.dt.Heap(MaterialHeap).SetImage(cpy, emisTexNr, 0, []driver.ImageView{tex}, nil)
	t.dt.Heap(MaterialHeap).SetSampler(cpy, emisSplrNr, 0, []driver.Sampler{splr})
}

// Frame returns a pointer to GPU memory mapping to a
// given FrameLayout of the global heap.
// A valid constant buffer must be set when this method
// is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *DrawTable) Frame(cpy int) *FrameLayout {
	t.validateHeapCopy(GlobalHeap, cpy)
	off := t.coff[GlobalHeap] + int64(frameSpan+lightSpan+shadowSpan)*blockSize*int64(cpy)
	s := t.cs[off:]
	return (*FrameLayout)(unsafe.Pointer(unsafe.SliceData(s)))
}

// Light returns a pointer to GPU memory mapping to a
// given LightLayout array of the global heap.
// A valid constant buffer must be set when this method
// is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *DrawTable) Light(cpy int) *[MaxLight]LightLayout {
	t.validateHeapCopy(GlobalHeap, cpy)
	off := t.coff[GlobalHeap] + int64(frameSpan+lightSpan+shadowSpan)*blockSize*int64(cpy)
	off += int64(frameSpan) * blockSize
	s := t.cs[off:]
	return (*[MaxLight]LightLayout)(unsafe.Pointer(unsafe.SliceData(s)))
}

// Shadow returns a pointer to GPU memory mapping to a
// given ShadowLayout array of the global heap.
// A valid constant buffer must be set when this method
// is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *DrawTable) Shadow(cpy int) *[MaxShadow]ShadowLayout {
	t.validateHeapCopy(GlobalHeap, cpy)
	off := t.coff[GlobalHeap] + int64(frameSpan+lightSpan+shadowSpan)*blockSize*int64(cpy)
	off += int64(frameSpan+lightSpan) * blockSize
	s := t.cs[off:]
	return (*[MaxShadow]ShadowLayout)(unsafe.Pointer(unsafe.SliceData(s)))
}

// Drawable returns a pointer to GPU memory mapping to a
// given DrawableLayout of the drawable heap.
// A valid constant buffer must be set when this method
// is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *DrawTable) Drawable(cpy int) *DrawableLayout {
	t.validateHeapCopy(DrawableHeap, cpy)
	off := t.coff[DrawableHeap] + int64(drawableSpan)*blockSize*int64(cpy)
	s := t.cs[off:]
	return (*DrawableLayout)(unsafe.Pointer(unsafe.SliceData(s)))
}

// Material returns a pointer to GPU memory mapping to a
// given MaterialLayout of the material heap.
// A valid constant buffer must be set when this method
// is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *DrawTable) Material(cpy int) *MaterialLayout {
	t.validateHeapCopy(MaterialHeap, cpy)
	off := t.coff[MaterialHeap] + int64(materialSpan)*blockSize*int64(cpy)
	s := t.cs[off:]
	return (*MaterialLayout)(unsafe.Pointer(unsafe.SliceData(s)))
}

// Joint returns a pointer to GPU memory mapping to a
// given JointLayout array of the joint heap.
// A valid constant buffer must be set when this method
// is called.
// Calling t.SetConstBuf invalidates any pointers
// returned by this method.
func (t *DrawTable) Joint(cpy int) *[MaxJoint]JointLayout {
	t.validateHeapCopy(JointHeap, cpy)
	off := t.coff[JointHeap] + int64(jointSpan)*blockSize*int64(cpy)
	s := t.cs[off:]
	return (*[MaxJoint]JointLayout)(unsafe.Pointer(unsafe.SliceData(s)))
}

// Free invalidates t and destroys the driver resources.
//
// NOTE: The constant buffer is not destroyed by this
// method; one can retrieve the buffer by calling
// t.SetConstBuf(nil, _) prior to calling t.Free.
func (t *DrawTable) Free() {
	if t.dt != nil {
		freeDescTable(t.dt)
	}
	*t = DrawTable{}
}

// NOTE: Tests will fail if the panic message changes.
func (t *DrawTable) validateHeapCopy(heap int, cpy int) {
	if uint(t.dcpy[heap]) <= uint(cpy) {
		panic("descriptor heap copy out of bounds")
	}
}

// NOTE: Tests will fail if the panic message changes.
func (t *DrawTable) validateTexSplr(heap int, cpy int, tex driver.ImageView, splr driver.Sampler) {
	t.validateHeapCopy(heap, cpy)
	switch {
	case tex == nil:
		panic("nil texture")
	case splr == nil:
		panic("nil sampler")
	}
}
