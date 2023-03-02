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
type Mesh struct {
	bufIdx  int
	primIdx int
	primLen int
}

// Primitive defines the data for a draw call.
type Primitive struct {
	bufIdx int
	index  int
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

	MaxSemantic = iota
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
		return "[!] invalid Semantic value"
	}
}

// format returns the driver.VertexFmt that the engine uses for
// storing s's data.
//
// TODO: Consider allowing alternative formats if it justifies
// the added complexity.
func (s Semantic) format() (f driver.VertexFmt) {
	switch s {
	case Position, Normal:
		f = driver.Float32x3
	case Tangent, Color0, Weights0:
		f = driver.Float32x4
	case TexCoord0, TexCoord1:
		f = driver.Float32x2
	case Joints0:
		f = driver.Uint16x4
	default:
		panic("invalid Semantic value")
	}
	return
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
		panic("invalid count on Semantic.conv call")
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
	var reason string
	switch {
	case data == nil:
		reason = "nil data"
	case len(data.Primitives) == 0:
		reason = "no primitive data"
	case len(data.Srcs) == 0:
		reason = "no data source"
	default:
		goto validData
	}
	err = errors.New(prefix + reason)
	return
validData:
	// TODO: Experiment with a more
	// fine-grained locking.
	storage.Lock()
	defer storage.Unlock()
	var prim, next, prev Primitive
	prim, err = newPrimitive(data, 0)
	if err != nil {
		return
	}
	prev = prim
	for i := 1; i < len(data.Primitives); i++ {
		next, err = newPrimitive(data, i)
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
		storage.link(prev, next)
	}
	m = &Mesh{
		bufIdx:  prim.bufIdx,
		primIdx: prim.index,
		primLen: len(data.Primitives),
	}
	return
}

// newPrimitive creates the primitive at data.Primitives[index].
func newPrimitive(data *Data, index int) (p Primitive, err error) {
	pdata := &data.Primitives[index]
	var reason string
	switch {
	case pdata.VertexCount < 0:
		reason = "invalid vertex count"
	case pdata.SemanticMask&Position == 0:
		reason = "no position semantic"
	case pdata.IndexCount > 0 && pdata.Index.Src >= len(data.Srcs):
		reason = "index data source out of bounds"
	default:
		var cnt = pdata.VertexCount
		if x := pdata.IndexCount; x > 0 {
			cnt = x
		}
		switch pdata.Topology {
		case driver.TPoint:
		case driver.TLine:
			if cnt&1 != 0 {
				reason = "invalid count for driver.TLine"
				goto invalidData
			}
		case driver.TLnStrip:
			if cnt < 2 {
				reason = "invalid count for driver.TLnStrip"
				goto invalidData
			}
		case driver.TTriangle:
			if cnt%3 != 0 {
				reason = "invalid count for driver.TTriangle"
				goto invalidData
			}
		case driver.TTriStrip:
			if cnt < 3 {
				reason = "invalid count for driver.TTriStrip"
				goto invalidData
			}
		}
		for i := range pdata.Semantics {
			if pdata.SemanticMask&(1<<i) == 0 {
				continue
			}
			if pdata.Semantics[i].Src >= len(data.Srcs) {
				reason = "semantic data source out of bounds"
				goto invalidData
			}
		}
		goto validData
	}
invalidData:
	err = errors.New(prefix + reason)
	return
validData:
	p, err = storage.newEntry(&data.Primitives[index], data.Srcs)
	return
}
