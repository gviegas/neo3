// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/internal/bitm"
)

const meshPrefix = "mesh: "

// Mesh is a collection of primitives.
// Each primitive defines the data for a draw call.
type Mesh struct {
	bufIdx  int
	primIdx int
	primLen int
}

// Len returns the number of primitives in m.
func (m *Mesh) Len() int { return m.primLen }

// inputs returns a driver.VertexIn slice describing the
// vertex input layout of the primitive at index prim.
// If prim is out of bounds, it returns a nil slice.
// Inputs are ordered by the Semantic value they represent.
// driver.VertexIn.Nr is set to Semantic.I().
func (m *Mesh) inputs(prim int) []driver.VertexIn {
	if prim >= m.primLen || prim < 0 {
		return nil
	}
	meshes.RLock()
	defer meshes.RUnlock()
	idx := m.primIdx
	for i := 0; i < prim; i++ {
		idx, _ = meshes.next(idx)
	}
	p := &meshes.prims[idx]
	var vin [MaxSemantic]driver.VertexIn
	var n int
	for i := 0; i < MaxSemantic; i++ {
		if p.mask&(1<<i) == 0 {
			continue
		}
		vin[n] = driver.VertexIn{
			Format: p.vertex[i].format,
			Stride: p.vertex[i].format.Size(),
			Nr:     i,
		}
		n++
	}
	return vin[:n]
}

// draw sets the vertex/index buffers and draws the primitive
// identified by prim.
// The caller is responsible for setting up cb as to be valid
// for drawing the primitive. In particular, it assumes that
// cb has an active render pass and that a compatible graphics
// pipeline has been set (a compatible pipeline is one whose
// vertex inputs match m.inputs(prim)).
// If prim is out of bounds, the call is silently ignored.
func (m *Mesh) draw(prim int, cb driver.CmdBuffer, instCnt int) {
	if prim >= m.primLen || prim < 0 {
		return
	}
	if instCnt < 1 {
		instCnt = 1
	}
	meshes.RLock()
	defer meshes.RUnlock()
	idx := prim
	for i := 0; i < prim; i++ {
		idx, _ = meshes.next(idx)
	}
	p := &meshes.prims[idx]
	// TODO: Consider computing these during
	// Mesh creation and storing alongside
	// the primitive (probably not worth it).
	var buf [MaxSemantic]driver.Buffer
	var off [MaxSemantic]int64
	var n int
	for i := 0; i < MaxSemantic; i++ {
		if p.mask&(1<<i) == 0 {
			continue
		}
		buf[n] = meshes.buf
		off[n] = int64(p.vertex[i].byteStart())
		n++
	}
	cb.SetVertexBuf(0, buf[:n], off[:n])
	if p.index.start >= p.index.end {
		cb.Draw(p.count, instCnt, 0, 0)
	} else {
		cb.SetIndexBuf(p.index.format, meshes.buf, int64(p.index.byteStart()))
		cb.DrawIndexed(p.count, instCnt, 0, 0, 0)
	}
}

// Free invalidates m and makes the GPU memory it holds
// available for new meshes.
func (m *Mesh) Free() {
	if m.primLen < 1 {
		return
	}
	meshes.Lock()
	defer meshes.Unlock()
	p := m.primIdx
	for {
		next, ok := meshes.next(p)
		meshes.freeEntry(p)
		if !ok {
			break
		}
		p = next
	}
	*m = Mesh{}
}

// Semantic specifies the intended use of a primitive's attribute.
type Semantic int

// Semantics.
const (
	Position Semantic = 1 << iota
	Normal
	Tangent
	TexCoord0
	TexCoord1
	Color0
	Joints0
	Weights0

	MaxSemantic int = iota
)

// I computes log₂(s).
// This value can be used to index into PrimitiveData.Semantics.
func (s Semantic) I() (i int) {
	for s > 1 {
		s >>= 1
		i++
	}
	return
}

// String implements fmt.Stringer.
func (s Semantic) String() string {
	switch s {
	case Position:
		return "Position"
	case Normal:
		return "Normal"
	case Tangent:
		return "Tangent"
	case TexCoord0:
		return "TexCoord0"
	case TexCoord1:
		return "TexCoord1"
	case Color0:
		return "Color0"
	case Joints0:
		return "Joints0"
	case Weights0:
		return "Weights0"
	default:
		return "!mesh.Semantic"
	}
}

// format returns the driver.VertexFmt that the engine uses for
// storing s's data.
//
// TODO: Consider allowing alternative formats if it justifies
// the added complexity.
func (s Semantic) format() driver.VertexFmt {
	switch s {
	case Position, Normal:
		return driver.Float32x3
	case Tangent, Color0, Weights0:
		return driver.Float32x4
	case TexCoord0, TexCoord1:
		return driver.Float32x2
	case Joints0:
		return driver.Uint16x4
	default:
		panic("undefined Semantic constant")
	}
}

// conv converts semantic data from a given driver.VertexFmt into
// the one which the engine expects. The following must hold:
//
//	cnt must be greater than 0 and
//	src encoding must be little-endian and
//	floating-point data must be IEEE-754.
//
// The following formats are supported:
//
//	Position:
//		driver.Float32x3 (no-op)
//	Normal:
//		driver.Float32x3 (no-op)
//	Tangent:
//		driver.Float32x4 (no-op)
//	TexCoord0,1:
//		driver.Float32x2 (no-op)
//		driver.Uint16x2
//		driver.Uint8x2
//	Color0:
//		driver.Float32x4 (no-op)
//		driver.Float32x3
//		driver.Uint16x4
//		driver.Uint16x3
//		driver.Uint8x4
//		driver.Uint8x3
//	Joints0:
//		driver.Uint16x4 (no-op)
//		driver.Uint8x4
//	Weights0:
//		driver.Float32x4 (no-op)
//		driver.Uint16x4
//		driver.Uint8x4
//
// If fmt is the expected format (i.e., s.format()), then nothing
// is done and it returns (src, nil).
func (s Semantic) conv(fmt driver.VertexFmt, src io.Reader, cnt int) (io.Reader, error) {
	if cnt < 1 {
		panic("Semantic.conv call has invalid count")
	}
	f := s.format()
	if f == fmt {
		return src, nil
	}

	// Read the whole data upfront.
	var b []byte
	switch x := src.(type) {
	case *bytes.Buffer:
		b = x.Bytes()[:cnt*fmt.Size()]
	default:
		b = make([]byte, cnt*fmt.Size())
		if _, err := io.ReadFull(src, b); err != nil {
			return nil, err
		}
	}

	// IEEE-754 binary => float32
	nextFloat32 := func() float32 {
		u := binary.LittleEndian.Uint32(b)
		b = b[4:]
		return math.Float32frombits(u)
	}
	// [0:65536) => [0.0:1.0]
	nextUnorm16AsFloat32 := func() float32 {
		u := binary.LittleEndian.Uint16(b)
		b = b[2:]
		return float32(u) / float32(^uint16(0))
	}
	// [0:256) => [0.0:1.0]
	nextUnorm8AsFloat32 := func() float32 {
		u := b[0]
		b = b[1:]
		return float32(u) / float32(^uint8(0))
	}
	// [0:256) => [0:256)
	nextUint8AsUint16 := func() uint16 {
		u := b[0]
		b = b[1:]
		return uint16(u)
	}

	var err = errors.New(meshPrefix + "unsupported vertex format for " + s.String())
	var p *byte

	switch s {
	case Position, Normal, Tangent:
		// These must match exactly.
		return nil, err

	case TexCoord0, TexCoord1:
		// Into driver.Float32x2.
		v := make([]float32, cnt*2)
		p = (*byte)(unsafe.Pointer(unsafe.SliceData(v)))
		switch fmt {
		case driver.Uint16x2:
			for len(v) > 0 {
				v[0] = nextUnorm16AsFloat32()
				v[1] = nextUnorm16AsFloat32()
				v = v[2:]
			}
		case driver.Uint8x2:
			for len(v) > 0 {
				v[0] = nextUnorm8AsFloat32()
				v[1] = nextUnorm8AsFloat32()
				v = v[2:]
			}
		default:
			return nil, err
		}

	case Color0:
		// Into driver.Float32x4.
		v := make([]float32, cnt*4)
		p = (*byte)(unsafe.Pointer(unsafe.SliceData(v)))
		switch fmt {
		case driver.Float32x3:
			for len(v) > 0 {
				v[0] = nextFloat32()
				v[1] = nextFloat32()
				v[2] = nextFloat32()
				v[3] = 1
				v = v[4:]
			}
		case driver.Uint16x4:
			for len(v) > 0 {
				v[0] = nextUnorm16AsFloat32()
				v[1] = nextUnorm16AsFloat32()
				v[2] = nextUnorm16AsFloat32()
				v[3] = nextUnorm16AsFloat32()
				v = v[4:]
			}
		case driver.Uint16x3:
			for len(v) > 0 {
				v[0] = nextUnorm16AsFloat32()
				v[1] = nextUnorm16AsFloat32()
				v[2] = nextUnorm16AsFloat32()
				v[3] = 1
				v = v[4:]
			}
		case driver.Uint8x4:
			for len(v) > 0 {
				v[0] = nextUnorm8AsFloat32()
				v[1] = nextUnorm8AsFloat32()
				v[2] = nextUnorm8AsFloat32()
				v[3] = nextUnorm8AsFloat32()
				v = v[4:]
			}
		case driver.Uint8x3:
			for len(v) > 0 {
				v[0] = nextUnorm8AsFloat32()
				v[1] = nextUnorm8AsFloat32()
				v[2] = nextUnorm8AsFloat32()
				v[3] = 1
				v = v[4:]
			}
		default:
			return nil, err
		}

	case Joints0:
		// Into driver.Uint16x4.
		v := make([]uint16, cnt*4)
		p = (*byte)(unsafe.Pointer(unsafe.SliceData(v)))
		switch fmt {
		case driver.Uint8x4:
			for len(v) > 0 {
				v[0] = nextUint8AsUint16()
				v[1] = nextUint8AsUint16()
				v[2] = nextUint8AsUint16()
				v[3] = nextUint8AsUint16()
				v = v[4:]
			}
		default:
			return nil, err
		}

	case Weights0:
		// Into driver.Float32x4.
		v := make([]float32, cnt*4)
		p = (*byte)(unsafe.Pointer(unsafe.SliceData(v)))
		switch fmt {
		case driver.Uint16x4:
			for len(v) > 0 {
				v[0] = nextUnorm16AsFloat32()
				v[1] = nextUnorm16AsFloat32()
				v[2] = nextUnorm16AsFloat32()
				v[3] = nextUnorm16AsFloat32()
				v = v[4:]
			}
		case driver.Uint8x4:
			for len(v) > 0 {
				v[0] = nextUnorm8AsFloat32()
				v[1] = nextUnorm8AsFloat32()
				v[2] = nextUnorm8AsFloat32()
				v[3] = nextUnorm8AsFloat32()
				v = v[4:]
			}
		default:
			return nil, err
		}

	default:
		// It would already have panicked due to the
		// s.format call above.
		panic("unreachable")
	}

	n := cnt * f.Size()
	return bytes.NewReader(unsafe.Slice(p, n)), nil
}

// SemanticData describes how to fetch semantic data
// from MeshData.Srcs.
type SemanticData struct {
	Format driver.VertexFmt
	Offset int64
	Src    int
}

// IndexData describes how to fetch index data from
// MeshData.Srcs.
type IndexData struct {
	Format driver.IndexFmt
	Offset int64
	Src    int
}

// PrimitiveData describes the data layout of a mesh's
// primitive and how to fetch such data from
// MeshData.Srcs.
type PrimitiveData struct {
	Topology    driver.Topology
	VertexCount int
	IndexCount  int
	// SemanticMask indicates which semantics
	// this primitive provides in the Semantics
	// array. Unused semantics need not be set.
	SemanticMask Semantic
	Semantics    [MaxSemantic]SemanticData
	// Index describes the index buffer's data.
	// It is ignored if IndexCount is less than
	// or equal to zero.
	Index IndexData
}

// MeshData defines the data layout of a whole mesh
// and provides the data sources to read from.
type MeshData struct {
	Primitives []PrimitiveData
	Srcs       []io.ReadSeeker
}

// NewMesh creates a new mesh.
func NewMesh(data *MeshData) (m *Mesh, err error) {
	err = validateMeshData(data)
	if err != nil {
		return
	}
	// TODO: Check whether locking/unlocking at
	// call sites improves performance.
	meshes.Lock()
	defer meshes.Unlock()
	var prim, next, prev int
	prim, err = meshes.newEntry(&data.Primitives[0], data.Srcs)
	if err != nil {
		return
	}
	prev = prim
	for i := 1; i < len(data.Primitives); i++ {
		next, err = meshes.newEntry(&data.Primitives[i], data.Srcs)
		if err != nil {
			prev = prim
			for {
				next, ok := meshes.next(prev)
				meshes.freeEntry(prev)
				if !ok {
					break
				}
				prev = next
			}
			return
		}
		meshes.prims[prev].next = next
		prev = next
	}
	m = &Mesh{
		// Currently, Mesh.bufIdx is not used.
		primIdx: prim,
		primLen: len(data.Primitives),
	}
	return
}

// validateData checks whether data is valid.
func validateMeshData(data *MeshData) error {
	newErr := func(reason string) error { return errors.New(meshPrefix + reason) }

	switch {
	case data == nil:
		return newErr("nil data")
	case len(data.Primitives) == 0:
		return newErr("no primitive data")
	case len(data.Srcs) == 0:
		return newErr("no data source")
	}

	for i := range data.Primitives {
		pdata := &data.Primitives[i]
		switch {
		case pdata.VertexCount < 0:
			return newErr("invalid vertex count")
		case pdata.SemanticMask&Position == 0:
			return newErr("no position semantic")
		case pdata.IndexCount > 0 && uint(pdata.Index.Src) >= uint(len(data.Srcs)):
			return newErr("index data source out of bounds")
		}
		// Back-ends usually can handle wrong counts
		// (e.g., by dropping excess vertices), but
		// this is likely a caller's mistake that
		// should be fixed.
		var cnt = pdata.VertexCount
		if x := pdata.IndexCount; x > 0 {
			cnt = x
		}
		switch pdata.Topology {
		case driver.TPoint:
		case driver.TLine:
			if cnt&1 != 0 {
				return newErr("invalid count for driver.TLine")
			}
		case driver.TLnStrip:
			if cnt < 2 {
				return newErr("invalid count for driver.TLnStrip")
			}
		case driver.TTriangle:
			if cnt%3 != 0 {
				return newErr("invalid count for driver.TTriangle")
			}
		case driver.TTriStrip:
			if cnt < 3 {
				return newErr("invalid count for driver.TTriStrip")
			}
		default:
			return newErr("undefined driver.Topology constant")
		}
		// It is fairly easy to make a primitive refer
		// to an invalid source, and we rather not
		// panic at out of bounds indexing.
		for i := range pdata.Semantics {
			if pdata.SemanticMask&(1<<i) == 0 {
				continue
			}
			if uint(pdata.Semantics[i].Src) >= uint(len(data.Srcs)) {
				return newErr("semantic data source out of bounds")
			}
		}
	}

	return nil
}

// Global mesh storage.
var meshes meshBuffer

// setMeshBuffer sets the GPU buffer into which mesh data
// will be stored.
// The buffer must be host-visible, its usage must include
// both driver.UVertexData and driver.UIndexData, and its
// capacity must be a multiple of 16384 bytes.
// It returns the replaced buffer, if any.
//
// NOTE: Calls to this function invalidate all previously
// created meshes.
//
// TODO: Review this functionality. It should be using a
// staging buffer on NUMA devices.
func setMeshBuffer(buf driver.Buffer) driver.Buffer {
	meshes.Lock()
	defer meshes.Unlock()
	switch buf {
	case meshes.buf:
		return nil
	case nil:
		meshes.spanMap = bitm.Bitm[uint32]{}
		meshes.primMap = bitm.Bitm[uint16]{}
		meshes.prims = nil
	default:
		c := buf.Cap()
		n := c / (spanBlock * spanMapNBit)
		if n > int64(^uint(0)>>1) || c != n*(spanBlock*spanMapNBit) {
			panic("invalid mesh buffer capacity")
		}
		meshes.spanMap = bitm.Bitm[uint32]{}
		meshes.spanMap.Grow(int(n))
		meshes.primMap = bitm.Bitm[uint16]{}
		meshes.prims = meshes.prims[:0]
	}
	prev := meshes.buf
	meshes.buf = buf
	return prev
}

// meshBuffer manages vertex/index data of created meshes.
type meshBuffer struct {
	buf     driver.Buffer
	spanMap bitm.Bitm[uint32]
	primMap bitm.Bitm[uint16]
	prims   []primitive
	sync.RWMutex
}

const (
	spanMapNBit = 32
	primMapNBit = 16
)

// store reads byteLen bytes from src and writes the data
// into the GPU buffer.
// It returns a span identifying the buffer range where
// the data was stored.
func (b *meshBuffer) store(src io.Reader, byteLen int) (span, error) {
	nb := (byteLen + (spanBlock - 1)) &^ (spanBlock - 1)
	ns := nb / spanBlock
	is, ok := b.spanMap.SearchRange(ns)
	if !ok {
		// TODO: Reconsider the growth strategy here.
		// Currently it assumes that SetMeshBuffer will
		// be called with a sensibly sized buffer and
		// that reallocations will not happen often,
		// so it optimizes for space.
		nplus := (ns + (spanMapNBit - 1)) / spanMapNBit
		bcap := int64(b.spanMap.Len()+nplus*spanMapNBit) * spanBlock
		buf, err := ctxt.GPU().NewBuffer(bcap, true, driver.UVertexData|driver.UIndexData)
		if err != nil {
			return span{}, err
		}
		if b.buf != nil {
			// TODO: Do this copy through the GPU
			// (requires driver.UCopySrc/UCopyDst).
			copy(buf.Bytes(), b.buf.Bytes())
			b.buf.Destroy()
		}
		b.buf = buf
		is = b.spanMap.Grow(nplus)
	}
	slc := b.buf.Bytes()[is*spanBlock : is*spanBlock+byteLen]
	for len(slc) > 0 {
		switch n, err := src.Read(slc); {
		case n > 0:
			slc = slc[n:]
		case err != nil:
			return span{}, err
		}
	}
	for i := 0; i < ns; i++ {
		b.spanMap.Set(is + i)
	}
	return span{is, is + ns}, nil
}

// newEntry creates a new entry in the buffer containing
// the primitive specified by data.
func (b *meshBuffer) newEntry(data *PrimitiveData, srcs []io.ReadSeeker) (p int, err error) {
	prim := primitive{
		topology: data.Topology,
		mask:     data.SemanticMask,
		next:     -1,
	}
	if data.IndexCount != 0 {
		prim.count = data.IndexCount
		prim.index.format = data.Index.Format
		var isz int
		switch prim.index.format {
		case driver.Index16:
			isz = 2
		case driver.Index32:
			isz = 4
		default:
			err = errors.New(meshPrefix + "undefined driver.IndexFmt constant")
		}
		src := srcs[data.Index.Src]
		off := data.Index.Offset
		if _, err = src.Seek(off, io.SeekStart); err != nil {
			return
		}
		if prim.index.span, err = b.store(src, prim.count*isz); err != nil {
			return
		}
	} else {
		prim.count = data.VertexCount
	}
	for i := range data.Semantics {
		sem := Semantic(1 << i)
		if data.SemanticMask&sem == 0 {
			continue
		}
		fmt := data.Semantics[i].Format
		src := srcs[data.Semantics[i].Src]
		off := data.Semantics[i].Offset
		if _, err = src.Seek(off, io.SeekStart); err != nil {
			b._freeEntry(&prim)
			return
		}
		var conv io.Reader
		if conv, err = sem.conv(fmt, src, data.VertexCount); err != nil {
			b._freeEntry(&prim)
			return
		}
		fmt = sem.format()
		prim.vertex[i].format = fmt
		if prim.vertex[i].span, err = b.store(conv, data.VertexCount*fmt.Size()); err != nil {
			b._freeEntry(&prim)
			return
		}
	}
	if i, ok := b.primMap.Search(); !ok {
		// TODO: Grow exponentially.
		var z [primMapNBit]primitive
		b.prims = append(b.prims, z[:]...)
		p = b.primMap.Grow(1)
	} else {
		p = i
	}
	b.primMap.Set(p)
	b.prims[p] = prim
	return
}

// next returns the next primitive in the list.
// If prim has no subsequent primitive (i.e., it was not
// linked to another primitive), then ok will be false.
// This is only relevant for meshes that contain multiple
// primitives.
func (b *meshBuffer) next(prim int) (p int, ok bool) {
	if i := b.prims[prim].next; i >= 0 {
		p = i
		ok = true
	}
	return
}

// freeEntry removes a primitive from the buffer.
// Any span held by prim is made available for use when
// creating new entries (it does not free GPU memory).
func (b *meshBuffer) freeEntry(prim int) {
	b.primMap.Unset(prim)
	b._freeEntry(&b.prims[prim])
}

func (b *meshBuffer) _freeEntry(prim *primitive) {
	// This ignores the mask and checks for
	// empty spans instead, so it is safe to
	// call from newEntry when it fails with
	// a partially set primitive.
	for i := range prim.vertex {
		for j := prim.vertex[i].start; j < prim.vertex[i].end; j++ {
			b.spanMap.Unset(j)
		}
	}
	for i := prim.index.start; i < prim.index.end; i++ {
		b.spanMap.Unset(i)
	}
	*prim = primitive{}
}

// primitive is an entry in a mesh buffer.
type primitive struct {
	topology driver.Topology
	count    int
	mask     Semantic
	vertex   [MaxSemantic]struct {
		format driver.VertexFmt
		span
	}
	index struct {
		format driver.IndexFmt
		span
	}
	// Index into meshBuffer.prims identifying
	// the next primitive of a mesh. Whether
	// this value is meaningful or not depends
	// on the Mesh.primLen field.
	next int
}

// span defines a buffer range in number of blocks.
type span struct {
	start int
	end   int
}

// span block size.
const spanBlock = 512

// byteStart computes the span's first byte.
func (s span) byteStart() int { return s.start * spanBlock }

// byteEnd computes the span's one-past-the-end byte.
func (s span) byteEnd() int { return s.end * spanBlock }

// byteLen computes the span's byte length.
func (s span) byteLen() int { return (s.end - s.start) * spanBlock }

// String implements fmt.Stringer.
func (s span) String() string {
	return fmt.Sprintf("{%d(%dB) %d(%dB)}", s.start, s.byteStart(), s.end, s.byteEnd())
}
