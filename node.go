// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package scene

import ()

// Node represents a single node in a scene graph.
// Nodes have at most one immediate ancestor and
// an arbitrary number of immediate descendants.
type Node struct {
	next *Node
	prev *Node
	sub  *Node
}

// NewNode creates an initialized node.
func NewNode() *Node { return new(Node).Init() }

// Init initializes node n.
// TODO: This is expected to be necessary when
// more fields are added to Node.
func (n *Node) Init() *Node { return n }

func (n *Node) Insert(sub *Node) {
	panic("not implemented")
}

func (n *Node) Remove() {
	panic("not implemented")
}

func (n *Node) ForEach(f func(*Node) bool) {
	panic("not implemented")
}
