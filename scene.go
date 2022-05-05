// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package scene provides functionality for creating and
// rendering scene graphs.
package scene

import (
	"github.com/gviegas/scene/node"
)

// Scene defines a scene graph.
type Scene struct {
	node.Node
}

// New creates an initialized scene.
func New() *Scene { return new(Scene).Init() }

// Init initializes a scene.
func (s *Scene) Init() *Scene {
	s.Node.Init()
	return s
}
