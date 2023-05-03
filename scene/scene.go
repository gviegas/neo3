// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package scene provides functionality for creating and
// rendering scene graphs.
package scene

import (
	"gviegas/neo3/node"
)

// Scene defines a scene graph.
type Scene struct {
	graph node.Graph
	// TODO
}

// New creates an initialized scene.
func New() *Scene { return new(Scene).Init() }

// Init initializes a scene.
func (s *Scene) Init() *Scene {
	// TODO
	return s
}
