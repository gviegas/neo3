// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"sync"

	"gviegas/neo3/engine/internal/shader"
	"gviegas/neo3/linear"
)

// Global drawables.
// TODO: These should live elsewhere.
var (
	drawableMap dataMap[Drawable, drawable]
	drawableMu  sync.Mutex
)

// drawable is what drawableMap stores.
type drawable struct {
	mesh   *Mesh
	matl   []*Material
	skin   *Skin
	layout shader.DrawableLayout
	// TODO...
}

// Drawable identifies an object to be rendered.
type Drawable int

// DrawParam describes how to render a Drawable.
// Matl is a list of non-nil materials where each
// element corresponds to a primitive in Mesh.
// Skinning is optional so Skin need not be set.
type DrawParam struct {
	World  linear.M4
	Normal linear.M3
	Mesh   *Mesh
	Matl   []*Material
	Skin   *Skin
	// TODO...
}
