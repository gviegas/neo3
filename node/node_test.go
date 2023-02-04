// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package node

import (
	"strconv"
	"testing"
	"unsafe"

	"github.com/gviegas/scene/linear"
)

// inode implements Interface for testing.
type inode struct {
	name    string
	local   linear.M4
	changed bool
}

func (n *inode) Local() *linear.M4 { return &n.local }
func (n *inode) Changed() bool     { return n.changed }

// want describes expected Graph state.
type want struct {
	next    Node
	nodeLen int
	nodeRem int
	dataLen int
}

// check checks whether the graph's state matches w.
func (g *Graph) check(w want, t *testing.T) {
	if g.next != w.next {
		t.Fatalf("Graph.next:\nhave %d\nwant %d", g.next, w.next)
	}
	if n := len(g.nodes); n != w.nodeLen {
		t.Fatalf("len(Graph.nodes):\nhave %d\nwant %d", n, w.nodeLen)
	}
	if r := g.nodeMap.Rem(); r != w.nodeRem {
		t.Fatalf("Graph.nodeMap.Rem:\nhave %d\nwant %d", r, w.nodeRem)
	}
	if n := len(g.data); n != w.dataLen {
		t.Fatalf("len(Graph.data):\nhave %d\nwant %d", n, w.dataLen)
	}
}

// checkNode checks whether the node's state matches w.
func (g *Graph) checkNode(n Node, w node, t *testing.T) {
	if nd := g.nodes[n-1]; nd != w {
		t.Fatalf("Graph.nodes[%d]:\nhave %v\nwant %v", n-1, nd, w)
	} else if self := g.data[w.data].node; self != n {
		t.Fatalf("Graph.data[%d].node:\nhave %v\nwant %v", w.data, self, n)
	}
}

// checkRemoval checks that ns has the expected length and that
// its elements have the expected names.
func (g *Graph) checkRemoval(ns []Interface, wantLen int, wantNames []string, t *testing.T) {
	if len(ns) != wantLen {
		t.Fatalf("len(Graph.Remove()):\nhave %d\nwant %d", len(ns), wantLen)
	}
	max := len(wantNames)
	if x := len(ns); x < max {
		max = x
	}
	for i := 0; i < max; i++ {
		if s := ns[i].(*inode).name; s != wantNames[i] {
			t.Fatalf("Graph.Remove()[%d].name:\nhave %s\nwant %s", i, s, wantNames[i])
		}
	}
}

func TestInsertRemove(t *testing.T) {
	var g Graph
	var w want
	var m linear.M4
	g.check(w, t)

	// Insert a node, remove it and then reinsert.
	n1 := g.Insert(&inode{"/1", m, true}, Nil)
	w = want{n1, 32, 31, 1}
	g.check(w, t)
	g.checkNode(n1, node{}, t)
	in := g.Remove(n1)
	g.checkRemoval(in, 1, []string{"/1"}, t)
	w = want{Nil, 32, 32, 0}
	g.check(w, t)
	n1 = g.Insert(&inode{"/1", m, true}, Nil)
	w = want{n1, 32, 31, 1}
	g.check(w, t)
	g.checkNode(n1, node{}, t)

	// Insert a 2nd root node and remove the 1st.
	n2 := g.Insert(&inode{"/2", m, true}, Nil)
	w.next = n2
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n2, node{next: n1, data: 1}, t)
	g.checkNode(n1, node{prev: n2}, t)
	in = g.Remove(n1)
	g.checkRemoval(in, 1, []string{"/1"}, t)
	w.nodeRem++
	w.dataLen--
	g.check(w, t)
	g.checkNode(n2, node{}, t)

	// Insert a descendant node, remove it and reinsert.
	n21 := g.Insert(&inode{"/2/1", m, true}, n2)
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n21, node{prev: n2, data: 1}, t)
	g.checkNode(n2, node{sub: n21}, t)
	in = g.Remove(n21)
	g.checkRemoval(in, 1, []string{"/2/1"}, t)
	w.nodeRem++
	w.dataLen--
	g.check(w, t)
	g.checkNode(n2, node{}, t)
	n21 = g.Insert(&inode{"/2/1", m, true}, n2)
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n21, node{prev: n2, data: 1}, t)
	g.checkNode(n2, node{sub: n21}, t)

	// Insert another descendant node and remove it.
	n22 := g.Insert(&inode{"/2/2", m, true}, n2)
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n22, node{next: n21, prev: n2, data: 2}, t)
	g.checkNode(n21, node{prev: n22, data: 1}, t)
	g.checkNode(n2, node{sub: n22}, t)
	in = g.Remove(n22)
	g.checkRemoval(in, 1, []string{"/2/2"}, t)
	w.nodeRem++
	w.dataLen--
	g.check(w, t)
	g.checkNode(n21, node{prev: n2, data: 1}, t)
	g.checkNode(n2, node{sub: n21}, t)

	// Insert another root node.
	n1 = g.Insert(&inode{"/1", m, true}, Nil)
	w.next = n1
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n2, node{prev: n1, sub: n21, data: 0}, t)
	g.checkNode(n1, node{next: n2, data: 2}, t)

	// Insert another descendant node and remove the other.
	n22 = g.Insert(&inode{"/2/2", m, true}, n2)
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n22, node{next: n21, prev: n2, data: 3}, t)
	g.checkNode(n21, node{prev: n22, data: 1}, t)
	g.checkNode(n2, node{prev: n1, sub: n22, data: 0}, t)
	g.checkNode(n1, node{next: n2, data: 2}, t)
	in = g.Remove(n21)
	g.checkRemoval(in, 1, []string{"/2/1"}, t)
	w.nodeRem++
	w.dataLen--
	g.check(w, t)
	g.checkNode(n22, node{prev: n2, data: 1}, t)
	g.checkNode(n2, node{prev: n1, sub: n22, data: 0}, t)
	g.checkNode(n1, node{next: n2, data: 2}, t)

	// This is valid and does nothing.
	in = g.Remove(Nil)
	g.checkRemoval(in, 0, nil, t)
	g.check(w, t)
	g.checkNode(n22, node{prev: n2, data: 1}, t)
	g.checkNode(n2, node{prev: n1, sub: n22, data: 0}, t)
	g.checkNode(n1, node{next: n2, data: 2}, t)

	// Insert a descendant of a descendant.
	n221 := g.Insert(&inode{"/2/2/1", m, true}, n22)
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n221, node{prev: n22, data: 3}, t)
	g.checkNode(n22, node{prev: n2, sub: n221, data: 1}, t)
	g.checkNode(n2, node{prev: n1, sub: n22, data: 0}, t)
	g.checkNode(n1, node{next: n2, data: 2}, t)

	// Remove the descendant's sub-graph.
	in = g.Remove(n22)
	g.checkRemoval(in, 2, []string{"/2/2", "/2/2/1"}, t)
	w.nodeRem += 2
	w.dataLen -= 2
	g.check(w, t)
	g.checkNode(n2, node{prev: n1, data: 0}, t)
	g.checkNode(n1, node{next: n2, data: 1}, t)

	// Insert a descendant node and remove by the root.
	n11 := g.Insert(&inode{"/1/1", m, true}, n1)
	w.nodeRem--
	w.dataLen++
	g.check(w, t)
	g.checkNode(n2, node{prev: n1, data: 0}, t)
	g.checkNode(n11, node{prev: n1, data: 2}, t)
	g.checkNode(n1, node{next: n2, sub: n11, data: 1}, t)
	in = g.Remove(n1)
	g.checkRemoval(in, 2, []string{"/1", "/1/1"}, t)
	w.next = n2
	w.nodeRem += 2
	w.dataLen -= 2
	g.check(w, t)
	g.checkNode(n2, node{}, t)

	// Remove the last node.
	in = g.Remove(n2)
	g.checkRemoval(in, 1, []string{"/2"}, t)
	w = want{Nil, 32, 32, 0}
	g.check(w, t)

	// This is valid and does nothing.
	in = g.Remove(Nil)
	g.checkRemoval(in, 0, nil, t)
	g.check(w, t)
}

func TestNodesGrowth(t *testing.T) {
	var g Graph

	// Graph.nodes needs to grow by a multiple of
	// 32 elements due to bitmap granularity.
	// This means an extra 1KB per bitmap word
	// when int is 64-bit.
	g.check(want{}, t)
	n1 := g.Insert(new(inode), Nil)
	g.check(want{next: n1, nodeLen: 32, nodeRem: 31, dataLen: 1}, t)
	g.checkRemoval(g.Remove(n1), 1, nil, t)
	g.check(want{nodeLen: 32, nodeRem: 32}, t)
	n1 = g.Insert(new(inode), Nil)
	for g.nodeMap.Rem() > 0 {
		g.Insert(new(inode), n1)
	}
	g.check(want{next: n1, nodeLen: 32, nodeRem: 0, dataLen: 32}, t)
	g.Insert(new(inode), n1)
	g.check(want{next: n1, nodeLen: 64, nodeRem: 31, dataLen: 33}, t)
	g.Insert(new(inode), n1)
	g.check(want{next: n1, nodeLen: 64, nodeRem: 30, dataLen: 34}, t)
	for g.nodeMap.Rem() > 0 {
		g.Insert(new(inode), n1)
	}
	g.check(want{next: n1, nodeLen: 64, nodeRem: 0, dataLen: 64}, t)
	g.checkRemoval(g.Remove(n1), 64, nil, t)
	g.check(want{nodeLen: 64, nodeRem: 64}, t)
	g.Insert(new(inode), Nil)
	g.check(want{next: n1, nodeLen: 64, nodeRem: 63, dataLen: 1}, t)

	t.Logf("Graph.nodes granularity is 32 elements (%d bytes)", unsafe.Sizeof(node{})*32)
}

func TestDepth(t *testing.T) {
	var g Graph
	var acc int
	insert := func(prev Node) Node {
		acc++
		return g.Insert(&inode{strconv.Itoa(acc), linear.M4{}, true}, prev)
	}
	const cnt = 10000
	n1 := insert(Nil)
	nx := n1
	names := make([]string, cnt)
	names[0] = "1"
	for i := 1; i < cnt; i++ {
		nx = insert(nx)
		names[i] = strconv.Itoa(i + 1)
	}
	const gran = (cnt + 31) &^ 31
	g.check(want{n1, gran, gran - cnt, cnt}, t)
	g.checkRemoval(g.Remove(n1), cnt, names, t)
	g.check(want{nodeLen: gran, nodeRem: gran}, t)
}

func TestBreadth(t *testing.T) {
	var g Graph
	const cnt = 10987
	nodes := make([]Node, cnt)
	for i := 0; i < cnt; i++ {
		nodes[i] = g.Insert(&inode{strconv.Itoa(i + 1), linear.M4{}, true}, Nil)
	}
	const gran = (cnt + 31) &^ 31
	g.check(want{nodes[cnt-1], gran, gran - cnt, cnt}, t)
	var acc int64
	for i, x := range nodes {
		in := g.Remove(x)
		g.checkRemoval(in, 1, []string{strconv.Itoa(i + 1)}, t)
		if d, err := strconv.ParseInt(in[0].(*inode).name, 10, 64); err != nil {
			t.Fatal(err)
		} else {
			acc += d
		}
	}
	if x := int64(cnt*cnt+cnt) / 2; x != acc {
		t.Fatalf("Graph.Remove: wrong accumulated value\nhave %d\nwant %d", acc, x)
	}
	g.check(want{nodeLen: gran, nodeRem: gran}, t)
}
