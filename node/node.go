// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package node implements the scene's graph.
package node

import (
	"github.com/gviegas/scene/internal/bitm"
	"github.com/gviegas/scene/linear"
)

// Interface of a node.
// Types that implement this interface can be inserted
// into a node Graph.
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

// node is the graph's data structure.
// Indices of nodes stored in a Graph can be derived
// by decrementing Node values by 1.
type node struct {
	next Node
	prev Node
	sub  Node
	data int
}

// data is the node's data.
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
// prev can be Nil, in which case n is inserted into the
// graph as an unconnected node.
// It returns a Node value that identifies n in g.
// NOTE: If prev is not Nil, it must have been generated
// from a previous call to g.Insert.
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
		// Valid Node values start at 1 so Nil
		// can be 0.
		newn = Node(idx + 1)
	} else {
		// Should never happen.
		panic("unexpected failure from bitm.Bitm.Search")
	}
	// Do not assume that g.nodes[newn-1] is the
	// zero value.
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

// Remove removes a node and its descendants.
// The sub-graph rooted at n becomes invalid afterwards,
// so neither n nor the Node of any of its descendants
// are valid for further use.
func (g *Graph) Remove(n Node) []Interface {
	// Swap-removes g.data[d].
	removeData := func(d int) {
		last := len(g.data) - 1
		if d < last {
			swap := g.data[last].node
			g.nodes[swap-1].data = d
			g.data[d] = g.data[last]
		}
		g.data[last] = data{}
		g.data = g.data[:last]
	}
	next := g.nodes[n-1].next
	prev := g.nodes[n-1].prev
	sub := g.nodes[n-1].sub
	data := g.nodes[n-1].data
	if g.next == n {
		g.next = next
	}
	if prev != Nil {
		// node.prev in the first child
		// refers to its parent.
		if g.nodes[prev-1].sub == n {
			g.nodes[prev-1].sub = next
		} else {
			g.nodes[prev-1].next = next
		}
	}
	if next != Nil {
		g.nodes[next-1].prev = prev
	}
	ns := []Interface{g.data[data].local}
	removeData(data)
	if sub != Nil {
		// TODO: Node cache on Graph.
		que := []Node{n}
		for len(que) > 0 {
			cur := que[0]
			for sub := g.nodes[cur-1].sub; sub != Nil; sub = g.nodes[sub-1].next {
				data := g.nodes[sub-1].data
				ns = append(ns, g.data[data].local)
				removeData(data)
				que = append(que, sub)
			}
			g.nodes[cur-1] = node{}
			g.nodeMap.Unset(int(cur - 1))
			que = que[1:]
		}
	} else {
		g.nodes[n-1] = node{}
		g.nodeMap.Unset(int(n - 1))
	}
	return ns
}
