// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package node

import (
	"fmt"
	"testing"
)

// fmt.Stringer for testing only.
// n.Name must have been set in order to produce
// meaningful output.
func (n *Node) String() string {
	const s = `
(%5s) <-> (%5s) <-> (%5s)
               |
               v
            (%5s)
`
	nd := [4]*Node{n.prev, n, n.next, n.sub}
	nm := [4]string{}
	for i := range nd {
		if nd[i] != nil {
			nm[i] = nd[i].Name
		} else {
			nm[i] = "<nil>"
		}
	}
	return fmt.Sprintf(s, nm[0], nm[1], nm[2], nm[3])
}

// logGraph outputs the scene graph whose root is n.
func (n *Node) logGraph(t *testing.T) {
	s := n.String()
	n.ForEach(func(n *Node) {
		s += n.String()
	})
	t.Log(s)
}

// testInsert calls n.Insert and checks that it works
// as expected.
func (n *Node) testInsert(sub *Node, t *testing.T) {
	n.Insert(sub)
	if n.sub != sub {
		t.Fatalf("n.Insert: n.sub\nhave %p\nwant %p\n%v", n.sub, sub, n)
	}
	if sub.prev != n {
		t.Fatalf("n.Insert: sub.prev\nhave %p\nwant %p\n%v", sub.prev, n, sub)
	}
}

// testRemove calls n.Remove and checks that it works
// as expected.
func (n *Node) testRemove(t *testing.T) {
	var anc, sub *Node
	if x := n.prev; x != nil && n == x.sub {
		anc = x
		sub = n.next
	}
	n.Remove()
	if n.next != nil {
		t.Fatalf("n.Remove: n.next\nhave %p\nwant nil\n%v", n.next, n)
	}
	if n.prev != nil {
		t.Fatalf("n.Remove: n.prev\nhave %p\nwant nil\n%v", n.prev, n)
	}
	if anc != nil && anc.sub != sub {
		t.Fatalf("n.Remove: anc.sub\nhave %p\nwant %p\n%v", anc.sub, sub, anc)
	}
}

func TestNode(t *testing.T) {
	n1 := New()
	n2 := New()
	n3 := New()
	n4 := New()
	n5 := New()
	n1.Name = "n1"
	n2.Name = "n2"
	n3.Name = "n3"
	n4.Name = "n4"
	n5.Name = "n5"

	n1.testInsert(n2, t)
	n1.testInsert(n3, t)
	n1.testInsert(n4, t)
	n3.testInsert(n5, t)
	n1.logGraph(t)
	n2.testRemove(t)
	n3.testRemove(t)
	n1.testRemove(t)
	n5.testRemove(t)
	n4.testRemove(t)
	n1.logGraph(t)
	n3.logGraph(t)

	n5.testInsert(n4, t)
	n4.testInsert(n3, t)
	n3.testInsert(n2, t)
	n2.testInsert(n1, t)
	n5.logGraph(t)
	n1.testRemove(t)
	n2.testRemove(t)
	n3.testRemove(t)
	n4.testRemove(t)
	n5.logGraph(t)
	n4.logGraph(t)
	n3.logGraph(t)
	n2.logGraph(t)
	n1.logGraph(t)

	n1.testInsert(n2, t)
	n2.testInsert(n3, t)
	n1.testInsert(n2, t)
	n1.logGraph(t)
	n1.testInsert(n3, t)
	n1.logGraph(t)
	n2.logGraph(t)
	n2.testRemove(t)
	n3.testInsert(n2, t)
	n1.logGraph(t)

	// Circular references break the graph.
	// In particular, it causes ForEach to
	// execute forever.
	//n1.testInsert(n2, t)
	//n2.testInsert(n1, t)
	//n1.logGraph(t)
}
