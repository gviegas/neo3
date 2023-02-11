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
		storage.prims[prev.index].next = next.index
		// TODO: primitive.prev probably won't
		// have any use; consider removing it.
		storage.prims[next.index].prev = prev.index
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
	panic("not implemented")
}
