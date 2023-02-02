// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package node implements the scene's graph.
package node

import (
	"github.com/gviegas/scene/internal/bitm"
	"github.com/gviegas/scene/linear"
)

// Interface of a node.
type Interface interface {
	// Local returns the local transform of the node.
	// It must not return nil.
	Local() *linear.M4

	// Changed returns whether the local transform
	// has changed.
	Changed() bool
}

// Node identifies a node in a Graph.
type Node int

// Nil represents an invalid Node.
const Nil Node = 0

type node struct {
	next Node
	prev Node
	sub  Node
	data int
}

type data struct {
	local Interface
	world linear.M4
	node  Node
}

// Graph is a node graph.
type Graph struct {
	next    Node
	world   linear.M4
	changed bool
	nodes   []node
	nodeMap bitm.Bitm[uint32]
	data    []data
}

// Insert inserts a new node as descendant of prev.
func (g *Graph) Insert(node Interface, prev Node) Node {
	panic("not implemented")
}

// Remove removes a node.
func (g *Graph) Remove(node Node) Interface {
	panic("not implemented")
}
