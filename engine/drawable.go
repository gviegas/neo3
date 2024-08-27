// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"gviegas/neo3/engine/internal/shader"
	"gviegas/neo3/linear"
)

// drawableMap is a dataMap for drawables.
type drawableMap struct{ dataMap[Drawable, drawable] }

// drawable is what a drawableMap stores.
type drawable struct {
	mesh   *Mesh
	mat    []*Material
	skin   *Skin
	layout shader.DrawableLayout
	// TODO...
}

// Drawable identifies an object to be rendered.
type Drawable int

// DrawParam describes how to render a Drawable.
// Mat is a list of non-nil materials where each
// element corresponds to a primitive in Mesh.
// Skinning is optional so Skin need not be set.
type DrawParam struct {
	World  linear.M4
	Normal linear.M3
	Mesh   *Mesh
	Mat    []*Material
	Skin   *Skin
	// TODO...
}
