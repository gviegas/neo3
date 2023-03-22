// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package material implements the material model used in
// the engine.
package material

import (
	"github.com/gviegas/scene/engine/texture"
)

// Material.
type Material struct {
	// TODO
}

// TexRef identifies a particular view of a 2D texture
// and its sampler, with sampling operations using a
// given UV set.
type TexRef struct {
	Texture *texture.Texture
	View    int
	Sampler *texture.Sampler
	UVSet   int
}

// UV sets matching mesh.TexCoord* semantics.
const (
	// mesh.TexCoord0.
	UVSet0 = iota
	// mesh.TexCoord1.
	UVSet1
)

// BaseColor is the material's base color.
type BaseColor struct {
	TexRef
	Factor [4]float32
}

// MetalRough is the material's matallic-roughness.
type MetalRough struct {
	TexRef
	Metalness float32
	Roughness float32
}

// Normal is the material's normal map.
type Normal struct {
	TexRef
	Scale float32
}

// Occlusion is the material's occlusion map.
type Occlusion struct {
	TexRef
	Strength float32
}

// Emissive is the material's emissive map.
type Emissive struct {
	TexRef
	Factor [3]float32
}

// Alpha modes.
const (
	AlphaOpaque = iota
	AlphaBlend
	AlphaMask
)
