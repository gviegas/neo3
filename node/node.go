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
func (g *Graph) Insert(n Interface, prev Node) Node {
	if g.nodeMap.Rem() == 0 {
		// TODO: Grow exponentially.
		var elems [32]node
		g.nodes = append(g.nodes, elems[:]...)
		g.nodeMap.Grow(1)
	}
	var newn Node
	if idx, ok := g.nodeMap.Search(); ok {
		g.nodeMap.Set(idx)
		newn = Node(idx + 1)
	} else {
		// Should never happen.
		panic("unexpected failure from bitm.Bitm.Search")
	}
	if prev != Nil {
		if sub := g.nodes[prev-1].sub; sub != Nil {
			g.nodes[newn-1].next = sub
			g.nodes[sub-1].prev = newn
		} else {
			g.nodes[newn-1].next = Nil
		}
		g.nodes[newn-1].prev = prev
		g.nodes[prev-1].sub = newn
	} else {
		if g.next != Nil {
			g.nodes[g.next-1].prev = newn
			g.nodes[newn-1].next = g.next
		} else {
			g.nodes[newn-1].next = Nil
		}
		g.nodes[newn-1].prev = Nil
		g.next = newn
	}
	g.nodes[newn-1].sub = Nil
	g.nodes[newn-1].data = len(g.data)
	var world linear.M4
	world.I()
	g.data = append(g.data, data{n, world, newn})
	return newn
}

// Remove removes a node.
func (g *Graph) Remove(n Node) Interface {
	panic("not implemented")
}
