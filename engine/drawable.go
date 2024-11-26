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

// Drawable identifies an entity to be rendered.
// A Drawable is always associated with a Renderer,
// thus there might be identical Drawable values
// that belong to different renderers.
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

// setLayout sets d.layout.
// TODO: This layout is expected to receive
// new additions.
func (d *drawable) setLayout(world *linear.M4, normal *linear.M3, id Drawable) {
	d.layout.SetWorld(world)
	d.layout.SetNormal(normal)
	d.layout.SetID(uint32(id))
}
