// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package node provides the elements of the scene graph.
package node

// Node represents a single node in a scene graph.
// Nodes have at most one immediate ancestor and
// an arbitrary number of immediate descendants.
type Node struct {
	next *Node
	prev *Node
	sub  *Node

	// Name for the node.
	// It is not used by node code.
	Name string
}

// New creates an initialized node.
func New() *Node { return new(Node).Init() }

// Init initializes node n.
// TODO: This is expected to be necessary when
// more fields are added to Node.
func (n *Node) Init() *Node { return n }

// Insert inserts node sub as immediate descendant
// of node n.
// sub must be either a descendant of n or part of
// an unrelated graph - it must not be an ancestor
// of node n.
func (n *Node) Insert(sub *Node) {
	sub.Remove()
	sub.next = n.sub
	sub.prev = n
	if n.sub != nil {
		n.sub.prev = sub
	}
	n.sub = sub
}

// Remove removes node n from its immediate ancestor.
func (n *Node) Remove() {
	// Note that Node.prev is only nil when the node
	// has no ancestors, since the prev field of the
	// first immediate descendant is set to refer to
	// its immediate ancestor.
	if n.prev != nil {
		if n.prev.sub == n {
			n.prev.sub = n.next
		} else {
			n.prev.next = n.next
		}
		if n.next != nil {
			n.next.prev = n.prev
		}
		n.prev = nil
		n.next = nil
	}
}

// ForEach calls f for each descendant of node n.
// Ancestors are processed first.
// The scene graph must not be changed until this
// method returns.
func (n *Node) ForEach(f func(*Node)) {
	if n.sub == nil {
		return
	}
	que := []*Node{n.sub}
	for len(que) > 0 {
		for nd := que[0]; nd != nil; nd = nd.next {
			f(nd)
			if sub := nd.sub; sub != nil {
				que = append(que, sub)
			}
		}
		que = que[1:]
	}
}

// Until calls f for each descendant of node n.
// Ancestors are processed first. If f returns false,
// Until returns immediately.
// The scene graph must not be changed until this
// method returns.
func (n *Node) Until(f func(*Node) bool) {
	if n.sub == nil {
		return
	}
	que := []*Node{n.sub}
	for len(que) > 0 {
		for nd := que[0]; nd != nil; nd = nd.next {
			if !f(nd) {
				return
			}
			if sub := nd.sub; sub != nil {
				que = append(que, sub)
			}
		}
		que = que[1:]
	}
}
