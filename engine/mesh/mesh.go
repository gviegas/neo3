// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package mesh implements the mesh data representation used
// in the engine's renderer.
package mesh

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"unsafe"

	"github.com/gviegas/scene/driver"
)

const prefix = "mesh: "

// Mesh is a collection of primitives.
// Each primitive defines the data for a draw call.
type Mesh struct {
	bufIdx  int
	primIdx int
	primLen int
}

// Len returns the number of primitives in m.
func (m *Mesh) Len() int { return m.primLen }

// Inputs returns a driver.VertexIn slice describing the
// vertex input layout of the primitive at index prim.
// If prim is out of bounds, it returns a nil slice.
// Inputs are ordered by the Semantic value they represent.
// driver.VertexIn.Nr is set to Semantic.I().
func (m *Mesh) Inputs(prim int) []driver.VertexIn {
	if prim >= m.primLen || prim < 0 {
		return nil
	}
	storage.RLock()
	defer storage.RUnlock()
	idx := m.primIdx
	for i := 0; i < prim; i++ {
		idx, _ = storage.next(idx)
	}
	p := &storage.prims[idx]
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

// Draw sets the vertex/index buffers and draws the primitive
// identified by prim.
// The caller is responsible for setting up cb as to be valid
// for drawing the primitive. In particular, it assumes that
// cb has an active render pass and that a compatible graphics
// pipeline has been set (a compatible pipeline is one whose
// vertex inputs match m.Inputs(prim)).
// If prim is out of bounds, the call is silently ignored.
func (m *Mesh) Draw(prim int, cb driver.CmdBuffer, instCnt int) {
	if prim >= m.primLen || prim < 0 {
		return
	}
	if instCnt < 1 {
		instCnt = 1
	}
	storage.RLock()
	defer storage.RUnlock()
	idx := prim
	for i := 0; i < prim; i++ {
		idx, _ = storage.next(idx)
	}
	p := &storage.prims[idx]
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
		buf[n] = storage.buf
		off[n] = int64(p.vertex[i].byteStart())
		n++
	}
	cb.SetVertexBuf(0, buf[:n], off[:n])
	if p.index.start >= p.index.end {
		cb.Draw(p.count, instCnt, 0, 0)
	} else {
		cb.SetIndexBuf(p.index.format, storage.buf, int64(p.index.byteStart()))
		cb.DrawIndexed(p.count, instCnt, 0, 0, 0)
	}
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
	}
	if _, err := io.ReadFull(src, b); err != nil {
		return nil, err
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

	var err = errors.New(prefix + "unsupported vertex format for " + s.String())
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

// SemanticData describes how to fetch semantic data from Data.Srcs.
type SemanticData struct {
	Format driver.VertexFmt
	Offset int64
	Src    int
}

// IndexData describes how to fetch index data from Data.Srcs.
type IndexData struct {
	Format driver.IndexFmt
	Offset int64
	Src    int
}

// PrimitiveData describes the data layout of a mesh's primitive
// and how to fetch such data from Data.Srcs.
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

// Data defines the data layout of a whole mesh and provides
// the data sources to read from.
type Data struct {
	Primitives []PrimitiveData
	Srcs       []io.ReadSeeker
}

// New creates a new mesh.
func New(data *Data) (m *Mesh, err error) {
	err = validateData(data)
	if err != nil {
		return
	}
	// TODO: Check whether locking/unlocking at
	// call sites improves performance.
	storage.Lock()
	defer storage.Unlock()
	var prim, next, prev int
	prim, err = storage.newEntry(&data.Primitives[0], data.Srcs)
	if err != nil {
		return
	}
	prev = prim
	for i := 1; i < len(data.Primitives); i++ {
		next, err = storage.newEntry(&data.Primitives[i], data.Srcs)
		if err != nil {
			prev = prim
			for {
				next, ok := storage.next(prev)
				storage.freeEntry(prev)
				if !ok {
					break
				}
				prev = next
			}
			return
		}
		storage.prims[prev].next = next
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
func validateData(data *Data) error {
	newErr := func(reason string) error { return errors.New(prefix + reason) }

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
