// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package mesh implements the mesh data representation
// used in the engine's renderer.
package mesh

import ()

// Mesh is a collection of primitives.
type Mesh struct {
	// TODO
}

// Primitive defines the data for a draw call.
type Primitive struct {
	// TODO
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
	TexCoord2
	Color0
	Color1
	Joints0
	Joints1
	Weights0
	Weights1
)
