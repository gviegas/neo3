// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package mesh implements the mesh data representation used
// in the engine's renderer.
package mesh

import (
	"errors"
	"io"

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
	case Position:
		f = driver.Float32x3
	case Normal:
		f = driver.Float32x3
	case Tangent:
		f = driver.Float32x4
	case TexCoord0:
		f = driver.Float32x2
	case TexCoord1:
		f = driver.Float32x2
	case Color0:
		f = driver.Float32x4
	case Joints0:
		f = driver.Uint16x4
	case Weights0:
		f = driver.Float32x4
	default:
		panic("invalid Semantic value")
	}
	return
}

// conv converts semantic data from a given driver.VertexFmt into
// the one which the engine expects.
// If fmt is the expected format (i.e., s.format()), then nothing
// is done and it returns (src, nil).
func (s Semantic) conv(fmt driver.VertexFmt, src io.Reader, cnt int) (io.Reader, error) {
	f := s.format()
	if f == fmt {
		return src, nil
	}
	panic("unfinished")
}

// PrimitiveData describes the data layout of a mesh's primitive.
type PrimitiveData struct {
	Topology    driver.Topology
	VertexCount int
	IndexCount  int
	// SemanticMask indicates which semantics
	// this primitive provides in the Semantics
	// array. Unused semantics need not be set.
	SemanticMask Semantic
	Semantics    [MaxSemantic]struct {
		Format driver.VertexFmt
		Offset int64
		Src    int
	}
	// Index describes the index buffer's data.
	// It is ignored if IndexCount is less than
	// or equal to zero.
	Index struct {
		Format driver.IndexFmt
		Offset int64
		Src    int
	}
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
	var prim, next, prev Primitive
	prim, err = newPrimitive(data, 0)
	if err != nil {
		return
	}
	prev = prim
	for i := 1; i < len(data.Primitives); i++ {
		next, err = newPrimitive(data, i)
		if err != nil {
			// TODO: Free primitives 0:i.
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
