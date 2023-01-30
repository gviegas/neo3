// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package mesh implements the mesh data representation used
// in the engine's renderer.
package mesh

import (
	"io"

	"github.com/gviegas/scene/driver"
)

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

// I computes log2(s).
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
