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
	// Graph methods may call it multiple times.
	// This pointer is never written to.
	Local() *linear.M4

	// Changed returns whether the local transform
	// has changed.
	// It is called by Graph.Update, exactly once,
	// to decide whether the node's world transform
	// needs to be recomputed.
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
// The zero value defines an initialized, empty graph.
type Graph struct {
	next    Node
	world   linear.M4
	wasSet  bool
	changed bool
	nodes   []node
	nodeMap bitm.Bitm[uint32]
	data    []data
	cache   struct {
		nodes   []Node
		data    []int
		changed []bool
	}
}

// nodeCache retrieves the initialized Node cache.
// It is re-sliced to have length 0.
func (g *Graph) nodeCache() []Node {
	if g.cache.nodes == nil {
		g.cache.nodes = make([]Node, 0, 1)
	}
	return g.cache.nodes[:0]
}

// dataCache retrieves the initialized int cache.
// It is re-sliced to have length 0.
func (g *Graph) dataCache() []int {
	if g.cache.data == nil {
		g.cache.data = make([]int, 0, 1)
	}
	return g.cache.data[:0]
}

// changedCache retrieves the initialized bool cache.
// It is re-sliced to have length 0.
func (g *Graph) changedCache() []bool {
	if g.cache.changed == nil {
		g.cache.changed = make([]bool, 0, 1)
	}
	return g.cache.changed[:0]
}

// Insert inserts a new node as descendant of prev.
// n must not be nil.
// prev can be Nil, in which case n is inserted into the
// graph as an unconnected node.
// It returns a Node value that identifies n in g.
// If prev is not Nil, it must belong to g.
func (g *Graph) Insert(n Interface, prev Node) Node {
	if n == nil {
		panic("cannot insert node.Interface(nil)")
	}
	if g.nodeMap.Rem() == 0 {
		switch x := g.nodeMap.Len(); {
		case x > 0:
			cnt := 1 + (x-31)/32
			g.nodes = append(g.nodes, g.nodes...)
			g.nodeMap.Grow(cnt)
		default:
			var elems [32]node
			g.nodes = append(g.nodes, elems[:]...)
			g.nodeMap.Grow(1)
		}
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
// It returns a slice containing the Interface of every
// removed Node, or nil if n was the Nil Node.
// The returned slice is such that, for every node x,
// x is at a lower index than all of its descendants
// (this means that, if n is not Nil, the Interface at
// index 0 is that of n itself).
// The sub-graph rooted at n becomes invalid afterwards,
// so neither n nor the Node of any of its descendants
// are valid for further use.
// If n is not Nil, it must belong to g.
func (g *Graph) Remove(n Node) []Interface {
	if n == Nil {
		return nil
	}
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
	g.nodes[n-1] = node{}
	g.nodeMap.Unset(int(n - 1))
	if sub != Nil {
		stk := append(g.nodeCache(), sub)
		for last := len(stk) - 1; last >= 0; last = len(stk) - 1 {
			cur := stk[last]
			stk = stk[:last]
			data := g.nodes[cur-1].data
			ns = append(ns, g.data[data].local)
			removeData(data)
			if next := g.nodes[cur-1].next; next != Nil {
				stk = append(stk, next)
			}
			if sub := g.nodes[cur-1].sub; sub != Nil {
				stk = append(stk, sub)
			}
			g.nodes[cur-1] = node{}
			g.nodeMap.Unset(int(cur - 1))
		}
		g.cache.nodes = stk
	}
	return ns
}

// Get returns the Interface of a given Node.
// It returns nil if n is the Nil Node.
// If n is not Nil, it must belong to g.
func (g *Graph) Get(n Node) Interface {
	if n == Nil {
		return nil
	}
	data := g.nodes[n-1].data
	return g.data[data].local
}

// World returns the world transform of a given Node.
// When n is Nil, it returns the global world, which will
// be equal to linear.M4{} if SetWorld was never called
// (it is not applied in this case).
// This matrix is not necessarily up to date.
func (g *Graph) World(n Node) *linear.M4 {
	if n == Nil {
		return &g.world
	}
	data := g.nodes[n-1].data
	return &g.data[data].world
}

// SetWorld sets the global world transform.
// Since the global world transform applies to every
// unconnected node, calling this method will
// invalidate the whole world.
func (g *Graph) SetWorld(w linear.M4) {
	g.world = w
	g.wasSet = true
	g.changed = true
}

// Update updates the graph to reflect the state of
// its nodes' transforms.
func (g *Graph) Update() {
	// Do a depth traversal of every unconnected
	// (root) node and update any sub-graph
	// rooted at a node that has changed.
	for n := g.next; n != Nil; n = g.nodes[n-1].next {
		data := g.nodes[n-1].data
		// Evaluate Interface.Changed exactly once.
		changed := g.data[data].local.Changed() || g.changed
		if changed {
			local := g.data[data].local.Local()
			if g.wasSet {
				g.data[data].world.Mul(&g.world, local)
			} else {
				g.data[data].world = *local
			}
		}
		sub := g.nodes[n-1].sub
		if sub == Nil {
			continue
		}
		// These three stacks will behave as one.
		nstk := append(g.nodeCache(), sub)
		dstk := append(g.dataCache(), data)
		cstk := append(g.changedCache(), changed)
		for last := len(nstk) - 1; last >= 0; last = len(nstk) - 1 {
			// Some descendant of the unconnected n.
			nsub := nstk[last]
			nstk = nstk[:last]
			// Data of the immediate ancestor of nsub.
			prevd := dstk[last]
			dstk = dstk[:last]
			// Whether prevd did change.
			chgd := cstk[last]
			cstk = cstk[:last]
			for {
				if next := g.nodes[nsub-1].next; next != Nil {
					nstk = append(nstk, next)
					dstk = append(dstk, prevd)
					cstk = append(cstk, chgd)
				}
				data := g.nodes[nsub-1].data
				// This will only affect descendants
				// since the next sibling (if any)
				// is already on the stack.
				chgd = g.data[data].local.Changed() || chgd
				if chgd {
					prevw := &g.data[prevd].world
					local := g.data[data].local.Local()
					g.data[data].world.Mul(prevw, local)
				}
				if sub := g.nodes[nsub-1].sub; sub != Nil {
					nsub = sub
					prevd = data
				} else {
					break
				}
			}
		}
		g.cache.nodes = nstk
		g.cache.data = dstk
		g.cache.changed = cstk
	}
	// World is now up to date.
	// New unconnected nodes will rely on g.wasSet
	// (which is never unset) to decide whether
	// they should apply g.world.
	g.changed = false
}

// Len returns the number of nodes in the graph.
func (g *Graph) Len() int { return len(g.data) }
