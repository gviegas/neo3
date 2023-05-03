// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package node

import (
	"strconv"
	"testing"

	"gviegas/neo3/linear"
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

	// Node growth is exponential.
	for len(g.data) < 64 {
		n1 = g.Insert(new(inode), Nil)
	}
	g.check(want{next: n1, nodeLen: 64, nodeRem: 0, dataLen: 64}, t)
	n1 = g.Insert(new(inode), Nil)
	g.check(want{next: n1, nodeLen: 128, nodeRem: 63, dataLen: 65}, t)
	for len(g.data) < 4096 {
		g.Insert(new(inode), n1)
	}
	g.check(want{next: n1, nodeLen: 4096, nodeRem: 0, dataLen: 4096}, t)
	g.Insert(new(inode), n1)
	g.check(want{next: n1, nodeLen: 8192, nodeRem: 4095, dataLen: 4097}, t)
	g.Insert(new(inode), g.Insert(new(inode), n1))
	g.check(want{next: n1, nodeLen: 8192, nodeRem: 4093, dataLen: 4099}, t)
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
	gran := 5
	for ; 1<<gran < cnt; gran++ {
	}
	gran = 1 << gran
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
	gran := 5
	for ; 1<<gran < cnt; gran++ {
	}
	gran = 1 << gran
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

func TestGet(t *testing.T) {
	var g Graph
	check := func(have, want Interface) {
		if have != want {
			t.Fatalf("Graph.Get:\nhave %v\nwant %v", have, want)
		}
	}
	check(g.Get(Nil), nil)

	i1 := inode{name: "/1"}
	n1 := g.Insert(&i1, Nil)
	check(g.Get(n1), &i1)

	i2 := inode{name: "/2"}
	n2 := g.Insert(&i2, Nil)
	check(g.Get(n2), &i2)

	i3 := inode{name: "/3"}
	n3 := g.Insert(&i3, Nil)
	check(g.Get(n3), &i3)

	i31 := inode{name: "/3/1"}
	n31 := g.Insert(&i31, n3)
	check(g.Get(n31), &i31)

	g.checkRemoval(g.Remove(n2), 1, []string{i2.name}, t)
	check(g.Get(n31), &i31)
	check(g.Get(n3), &i3)
	check(g.Get(n1), &i1)

	i311 := inode{name: "3/1/1"}
	n311 := g.Insert(&i311, n31)
	check(g.Get(n311), &i311)

	i11 := inode{name: "/1/1"}
	n11 := g.Insert(&i11, n1)
	check(g.Get(n311), &i311)
	check(g.Get(n31), &i31)
	check(g.Get(n3), &i3)
	check(g.Get(n11), &i11)
	check(g.Get(n1), &i1)

	g.checkRemoval(g.Remove(n31), 2, []string{"/3/1", "3/1/1"}, t)
	check(g.Get(n3), &i3)
	check(g.Get(n11), &i11)
	check(g.Get(n1), &i1)
	check(g.Get(Nil), nil)
}

func TestWorld(t *testing.T) {
	var g Graph

	// Global world.
	if w := g.World(Nil); *w != (linear.M4{}) {
		t.Fatalf("Graph.World:\nhave %v\nwant %v", *w, linear.M4{})
	}
	if g.changed {
		t.Fatal("Graph.changed:\nhave true\nwant false")
	}

	var m linear.M4
	m.I()
	g.SetWorld(m)
	if w := g.World(Nil); *w != m {
		t.Fatalf("Graph.World:\nhave %v\nwant %v", *w, m)
	}
	if !g.changed {
		t.Fatal("Graph.changed:\nhave false\nwant true")
	}

	m.Translate(1, 2, -3)
	g.SetWorld(m)
	if w := g.World(Nil); *w != m {
		t.Fatalf("Graph.World:\nhave %v\nwant %v", *w, m)
	}
	if !g.changed {
		t.Fatal("Graph.changed:\nhave false\nwant true")
	}

	g.SetWorld(linear.M4{})
	if w := g.World(Nil); *w != (linear.M4{}) {
		t.Fatalf("Graph.World:\nhave %v\nwant %v", *w, linear.M4{})
	}
	if !g.changed {
		t.Fatal("Graph.changed:\nhave false\nwant true")
	}

	a := [4]linear.V4{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, -8, -7, -6}, {-5, -4, -3, -2}}
	g.SetWorld(a)
	if w := g.World(Nil); *w != a {
		t.Fatalf("Graph.World:\nhave %v\nwant %v", *w, a)
	}
	if !g.changed {
		t.Fatal("Graph.changed:\nhave false\nwant true")
	}

	// Node's world.
	var n Node
	ns := make([]Node, 0, 4)
	for i := 0; i < cap(ns); i++ {
		ns = append(ns, g.Insert(new(inode), n))
		n = ns[i]
	}
	m.I()
	for _, n := range ns {
		world := g.World(n)
		if *world != m {
			t.Fatalf("Graph.World(%d):\nhave %v\nwant %v", n, *world, m)
		}
		data := g.nodes[n-1].data
		if w := &g.data[data].world; w != world {
			t.Fatalf("Graph.World(%d):\nhave %p\nwant %p", n, world, w)
		}
	}
}

func TestUpdate(t *testing.T) {
	var g Graph
	g.Update()
	var id linear.M4
	id.I()
	checkWorld := func(n Node, w linear.M4) {
		world := g.World(n)
		if *world != w {
			t.Fatalf("Graph.World(%d):\nhave %v\nwant %v", n, *world, w)
		}
		if *world == (linear.M4{}) {
			t.Fatalf("Graph.World(%d): unexpected zero matrix", n)
		}
	}

	// New unconnected - global world not applied (!g.wasSet).
	var m1 linear.M4
	m1.Scale(2, 2, 2)
	i1 := inode{"/1", m1, true}
	n1 := g.Insert(&i1, Nil)
	checkWorld(n1, id)
	g.Update()
	checkWorld(n1, m1)

	// Global world set and thus applied.
	g.SetWorld(linear.M4{{0: -1}, {1: -1}, {2: -1}, {3: 1}})
	checkWorld(n1, m1)
	g.Update()
	m1.Mul(&g.world, &m1)
	checkWorld(n1, m1)

	// New unconnected - global world applied.
	var m2 linear.M4
	m2.Scale(2, 4, 8)
	i2 := inode{"/2", m2, true}
	n2 := g.Insert(&i2, Nil)
	checkWorld(n2, id)
	g.Update()
	m2.Mul(&g.world, &m2)
	checkWorld(n2, m2)

	// Global world set and thus applied anew.
	g.SetWorld(id)
	checkWorld(n2, m2)
	checkWorld(n1, m1)
	g.Update()
	m2[0][0] = -m2[0][0]
	m2[1][1] = -m2[1][1]
	m2[2][2] = -m2[2][2]
	checkWorld(n2, m2)
	m1[0][0] = -m1[0][0]
	m1[1][1] = -m1[1][1]
	m1[2][2] = -m1[2][2]
	checkWorld(n1, m1)

	// Skip update of n2 (!Changed()).
	m1.Translate(10, 20, 30)
	i1.local = m1
	i2.local = m1
	i2.changed = false
	g.Update()
	checkWorld(n2, m2)
	checkWorld(n1, m1)

	// Do not skip update of n2 (Changed()).
	g.Update()
	checkWorld(n2, m2)
	checkWorld(n1, m1)
	i2.changed = true
	g.Update()
	checkWorld(n2, m1)
	checkWorld(n1, m1)
	m2.Translate(60, 50, 40)
	i2.local = m2
	g.Update()
	checkWorld(n2, m2)
	checkWorld(n1, m1)

	// Insert unchanged descendants.
	var m21, m22, m211 linear.M4
	m21.Scale(2, 2, 2)
	m22.Scale(4, 4, 4)
	m211.Scale(8, 8, 8)
	i21 := inode{"/2/1", m21, false}
	i22 := inode{"/2/2", m22, false}
	i211 := inode{"/2/1/1", m211, false}
	n21 := g.Insert(&i21, n2)
	n22 := g.Insert(&i22, n2)
	n211 := g.Insert(&i211, n21)
	i2.changed = false
	g.Update()
	checkWorld(n211, id)
	checkWorld(n22, id)
	checkWorld(n21, id)
	checkWorld(n2, m2)
	checkWorld(n1, m1)

	// Update one descendant.
	var tmp linear.M4
	tmp.Mul(g.World(n2), &m22)
	i22.changed = true
	i2.local.Scale(-1, -1, -1)
	g.Update()
	checkWorld(n211, id)
	checkWorld(n22, tmp)
	checkWorld(n21, id)
	checkWorld(n2, m2)
	checkWorld(n1, m1)

	// Update n21's sub-graph.
	i21.changed = true
	g.Update()
	checkWorld(n22, tmp) // Not affected.
	tmp.Mul(g.World(n21), &m211)
	checkWorld(n211, tmp)
	tmp.Mul(g.World(n2), &m21)
	checkWorld(n21, tmp)
	checkWorld(n2, m2)
	checkWorld(n1, m1)

	// Update n2's sub-graph.
	m2 = i2.local
	i21.changed = false // Shouldn't matter.
	i2.changed = true
	g.Update()
	tmp.Mul(g.World(n21), &m211)
	checkWorld(n211, tmp)
	tmp.Mul(&m2, &m22)
	checkWorld(n22, tmp)
	tmp.Mul(&m2, &m21)
	checkWorld(n21, tmp)
	checkWorld(n2, m2)
	checkWorld(n1, m1)

	// Depth update.
	ms := [5]linear.M4{
		{{0: 2}, {1: 1}, {2: 1}, {1, 6, 9, 1}},
		{{0: 3}, {1: 1}, {2: 1}, {2, 7, 8, 1}},
		{{0: 4}, {1: 1}, {2: 1}, {3, 8, 7, 1}},
		{{0: 5}, {1: 1}, {2: 1}, {4, 9, 6, 1}},
		{{0: 6}, {1: 1}, {2: 1}, {5, 0, 5, 1}},
	}
	ns := make([]Node, 0, len(ms))
	var n Node
	for _, m := range ms {
		n = g.Insert(&inode{local: m}, n)
		ns = append(ns, n)
	}
	g.Update()
	for _, n := range ns {
		checkWorld(n, id)
	}
	for i := 2; i < 4; i++ {
		g.Get(ns[i]).(*inode).changed = true
	}
	g.Update()
	g.Update()
	tmp.I()
	for i := 2; i < len(ns); i++ {
		tmp.Mul(&tmp, g.Get(ns[i]).Local())
		checkWorld(ns[i], tmp)
	}
	for i := 0; i < 2; i++ {
		checkWorld(ns[i], id)
		g.Get(ns[i]).(*inode).changed = true
	}
	g.Update()
	tmp.I()
	for _, n := range ns {
		tmp.Mul(&tmp, g.Get(n).Local())
		checkWorld(n, tmp)
	}
	for _, n := range ns {
		g.Get(n).(*inode).changed = false
	}
	g.Update()
	tmp.I()
	for _, n := range ns {
		tmp.Mul(&tmp, g.Get(n).Local())
		checkWorld(n, tmp)
	}

	// Breadth update.
	n = ns[len(ns)/2]
	ns = ns[:0]
	for _, m := range ms {
		ns = append(ns, g.Insert(&inode{local: m}, n))
	}
	g.Update()
	g.Update()
	for _, n := range ns {
		checkWorld(n, id)
	}
	g.Get(n).(*inode).changed = true
	g.Update()
	world := g.World(n)
	for _, n := range ns {
		tmp.Mul(world, g.Get(n).Local())
		checkWorld(n, tmp)
	}

	// Recompute the whole graph.
	for i := range g.data {
		g.data[i].world = linear.M4{}
	}
	tmp.Scale(0.25, 0.25, 0.25)
	g.SetWorld(tmp)
	g.Update()
	ns = ns[:0]
	for n := g.next; n != Nil; n = g.nodes[n-1].next {
		ns = append(ns, n)
	}
	for len(ns) > 0 {
		n := ns[0]
		for sub := g.nodes[n-1].sub; sub != Nil; sub = g.nodes[sub-1].next {
			tmp.Mul(g.World(n), g.Get(sub).Local())
			checkWorld(sub, tmp)
			// Now back to top.
			tmp.Mul(g.Get(n).Local(), g.Get(sub).Local())
			cur := n
			prev := g.nodes[cur-1].prev
			for prev != Nil {
				if psub := g.nodes[prev-1].sub; psub == cur {
					tmp.Mul(g.Get(prev).Local(), &tmp)
				}
				cur = prev
				prev = g.nodes[prev-1].prev
			}
			tmp.Mul(&g.world, &tmp)
			checkWorld(sub, tmp)
			ns = append(ns, sub)
		}
		ns = ns[1:]
	}
}

func TestCaching(t *testing.T) {
	var g Graph

	// Graph.Remove uses the Node cache
	// when removing multiple nodes.
	n1 := g.Insert(new(inode), Nil)
	n2 := g.Insert(new(inode), Nil)
	n21 := g.Insert(new(inode), n2)
	funcs := [3]func(){
		func() { g.Remove(n1) },
		func() { g.Remove(n21) }, // Note the ordering.
		func() { g.Remove(n2) },
	}
	for _, f := range funcs {
		f()
		if g.cache.nodes != nil {
			t.Fatal("Graph.cache.nodes: unexpected non-nil slice")
		}
		if g.cache.data != nil {
			t.Fatal("Graph.cache.data: unexpected non-nil slice")
		}
		if g.cache.changed != nil {
			t.Fatal("Graph.cache.changed: unexpected non-nil slice")
		}
	}
	n1 = g.Insert(new(inode), Nil)
	n2 = g.Insert(new(inode), Nil)
	n21 = g.Insert(new(inode), n2)
	g.checkRemoval(g.Remove(n1), 1, nil, t)
	if g.cache.nodes != nil {
		t.Fatal("Graph.cache.nodes: unexpected non-nil slice")
	}
	if g.cache.data != nil {
		t.Fatal("Graph.cache.data: unexpected non-nil slice")
	}
	if g.cache.changed != nil {
		t.Fatal("Graph.cache.changed: unexpected non-nil slice")
	}
	g.checkRemoval(g.Remove(n2), 2, nil, t)
	if g.cache.nodes == nil {
		t.Fatal("Graph.cache.nodes: unexpected nil slice")
	}
	if g.cache.data != nil {
		t.Fatal("Graph.cache.data: unexpected non-nil slice")
	}
	if g.cache.changed != nil {
		t.Fatal("Graph.cache.changed: unexpected non-nil slice")
	}

	// Graph.Update uses the Node, int and
	// bool caches when depth is greater
	// than one (i.e., descendants exist).
	ncap := cap(g.cache.nodes)
	nptr := (*[0]Node)(g.cache.nodes)
	n1 = g.Insert(new(inode), Nil)
	n2 = g.Insert(new(inode), Nil)
	g.Update()
	if c := cap(g.cache.nodes); c != ncap {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant %d", c, ncap)
	}
	if p := (*[0]Node)(g.cache.nodes); p != nil && p != nptr {
		t.Fatalf("&Graph.cache.nodes[0]:\nhave %p\nwant %p", p, nptr)
	}
	if g.cache.data != nil {
		t.Fatal("Graph.cache.data: unexpected non-nil slice")
	}
	if g.cache.changed != nil {
		t.Fatal("Graph.cache.changed: unexpected non-nil slice")
	}
	n21 = g.Insert(new(inode), n2)
	g.Update()
	if c := cap(g.cache.nodes); c != ncap {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant %d", c, ncap)
	}
	if p := (*[0]Node)(g.cache.nodes); p == nil || p != nptr {
		t.Fatalf("&Graph.cache.nodes[0]:\nhave %p\nwant %p", p, nptr)
	}
	if g.cache.data == nil {
		t.Fatal("Graph.cache.data: unexpected nil slice")
	}
	if g.cache.changed == nil {
		t.Fatal("Graph.cache.changed: unexpected nil slice")
	}

	// Graph.Remove and Graph.Update share
	// the Node cache.
	g = Graph{}
	n := Nil
	const cnt = 500
	for i := 0; i < cnt; i++ {
		n = g.Insert(new(inode), n)
	}
	// In this case, Graph.Remove can do
	// with a single Node in the cache.
	g.checkRemoval(g.Remove(g.next), cnt, nil, t)
	if c := cap(g.cache.nodes); c != 1 {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant 1", c)
	}
	n = g.Insert(new(inode), Nil)
	g.Insert(new(inode), n)
	g.Insert(new(inode), g.Insert(new(inode), n))
	g.checkRemoval(g.Remove(n1), 4, nil, t)
	if c := cap(g.cache.nodes); c < 2 {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant >= 2", c)
	}
	if g.cache.data != nil {
		t.Fatal("Graph.cache.data: unexpected non-nil slice")
	}
	if g.cache.changed != nil {
		t.Fatal("Graph.cache.changed: unexpected non-nil slice")
	}
	n = g.Insert(new(inode), Nil)
	for i := 0; i < cnt/2; i++ {
		g.Insert(new(inode), n)
		n = g.Insert(new(inode), n)
	}
	nptr = (*[0]Node)(g.cache.nodes)
	// This update should grow all caches.
	g.Update()
	ncap = cap(g.cache.nodes)
	dcap := cap(g.cache.data)
	ccap := cap(g.cache.changed)
	if (*[0]Node)(g.cache.nodes) == nptr {
		t.Log("(*[0])(Graph.cache.nodes): underlying array should likely have changed")
	}
	if ncap < cnt/2 {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant >= %d", ncap, cnt/2)
	}
	if dcap < cnt/2 {
		t.Fatalf("cap(Graph.cache.data):\nhave %d\nwant >= %d", dcap, cnt/2)
	}
	if ccap < cnt/2 {
		t.Fatalf("cap(Graph.cache.changed):\nhave %d\nwant >= %d", ccap, cnt/2)
	}
	nptr = (*[0]Node)(g.cache.nodes)
	dptr := (*[0]int)(g.cache.data)
	cptr := (*[0]bool)(g.cache.changed)
	// This update should not grow the caches.
	g.Update()
	if p := (*[0]Node)(g.cache.nodes); p != nptr {
		t.Fatalf("(*[0])(Graph.cache.nodes):\nhave %p\nwant %p", p, nptr)
	}
	if p := (*[0]int)(g.cache.data); p != dptr {
		t.Fatalf("(*[0])(Graph.cache.data):\nhave %p\nwant %p", p, dptr)
	}
	if p := (*[0]bool)(g.cache.changed); p != cptr {
		t.Fatalf("(*[0])(Graph.cache.changed):\nhave %p\nwant %p", p, cptr)
	}
	if x := cap(g.cache.nodes); x != ncap {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant %d", x, ncap)
	}
	if x := cap(g.cache.data); x != dcap {
		t.Fatalf("cap(Graph.cache.data):\nhave %d\nwant %d", x, dcap)
	}
	if x := cap(g.cache.changed); x != ccap {
		t.Fatalf("cap(Graph.cache.changed):\nhave %d\nwant %d", x, ccap)
	}
	n = g.Insert(new(inode), Nil)
	for i := 0; i < cnt; i++ {
		g.Insert(new(inode), n)
		n = g.Insert(new(inode), n)
	}
	// In this case, Graph.Remove should need
	// a cache that is roughly half the size
	// of the removed sub-graph.
	g.checkRemoval(g.Remove(g.next), 1+cnt*2, nil, t)
	if x := cap(g.cache.nodes); x <= ncap {
		t.Fatalf("cap(Graph.cache.nodes):\nhave %d\nwant > %d", x, ncap)
	}
}

func TestLen(t *testing.T) {
	var g Graph
	check := func(want int) {
		if len := g.Len(); len != want {
			t.Fatalf("Graph.Len:\nhave %d\nwant %d", len, want)
		}
	}
	check(0)
	n1 := g.Insert(new(inode), Nil)
	check(1)
	g.checkRemoval(g.Remove(n1), 1, nil, t)
	check(0)
	n1 = g.Insert(new(inode), Nil)
	check(1)
	n11 := g.Insert(new(inode), n1)
	check(2)
	g.checkRemoval(g.Remove(n11), 1, nil, t)
	check(1)
	n11 = g.Insert(new(inode), n1)
	check(2)
	g.checkRemoval(g.Remove(n1), 2, nil, t)
	check(0)
	for i := 0; i < 10; i++ {
		g.Insert(new(inode), g.Insert(new(inode), Nil))
	}
	check(20)
	for i := 1; i <= 5; i++ {
		g.checkRemoval(g.Remove(g.next), 2, nil, t)
		check(20 - i*2)
	}
	for next := g.next; g.nodes[next-1].next != Nil; next = g.next {
		g.checkRemoval(g.Remove(next), 2, nil, t)
	}
	check(2)
	g.checkRemoval(g.Remove(g.nodes[g.next-1].sub), 1, nil, t)
	check(1)
	g.checkRemoval(g.Remove(g.next), 1, nil, t)
	check(0)
}

func TestRemovalOrder(t *testing.T) {
	var g Graph
	n := g.Insert(&inode{name: "0"}, Nil)

	n3 := g.Insert(&inode{name: "11"}, n)
	n31 := g.Insert(&inode{name: "12"}, n3)
	g.Insert(&inode{name: "16"}, n31)
	n311 := g.Insert(&inode{name: "13"}, n31)
	g.Insert(&inode{name: "15"}, n311)
	g.Insert(&inode{name: "14"}, n311)

	n2 := g.Insert(&inode{name: "9"}, n)
	g.Insert(&inode{name: "10"}, n2)

	n1 := g.Insert(&inode{name: "1"}, n)
	n12 := g.Insert(&inode{name: "4"}, n1)
	n121 := g.Insert(&inode{name: "5"}, n12)
	g.Insert(&inode{name: "8"}, n121)
	g.Insert(&inode{name: "7"}, n121)
	g.Insert(&inode{name: "6"}, n121)
	n11 := g.Insert(&inode{name: "2"}, n1)
	g.Insert(&inode{name: "3"}, n11)

	ns := g.Remove(n)
	for i := range ns {
		switch x, err := strconv.Atoi(ns[i].(*inode).name); {
		case err != nil:
			t.Fatal(err)
		case x != i:
			t.Fatalf("Graph.Remove: invalid order\nhave %d\nwant %d", x, i)
		}
	}
}

func TestNilInterface(t *testing.T) {
	var g Graph
	defer func() {
		const s = "cannot insert node.Interface(nil)"
		switch recover() {
		case s:
		default:
			t.Fatalf("Graph.Insert: expected to panic with:\n%#v", s)
		}
	}()
	g.Insert(nil, Nil)
}

func TestIgnored(t *testing.T) {
	var g Graph

	var m1, m2, m3, m4, m5, m6 linear.M4
	m1.Scale(0.5, 0.5, 0.5)
	m2.Translate(-1, -2, -3)
	m3.Scale(2, 2, 2)
	m4.Translate(16, 64, 256)
	m5.Scale(0.25, 0.25, 0.25)
	m6.Translate(30, 20, 10)

	n1 := g.Insert(&inode{"/1", m1, true}, Nil)
	n11 := g.Insert(&inode{"/1/1", m2, true}, n1)
	n12 := g.Insert(&inode{"/1/2", m3, true}, n1)
	n121 := g.Insert(&inode{"1/2/1", m4, true}, n12)

	n2 := g.Insert(&inode{"/2", m2, true}, Nil)

	n3 := g.Insert(&inode{"/3", m3, true}, Nil)
	n31 := g.Insert(&inode{"/3/1", m4, true}, n3)
	n311 := g.Insert(&inode{"/3/1/1", m5, true}, n31)
	n3111 := g.Insert(&inode{"/3/1/1/1", m6, true}, n311)

	check := func(n Node, ms ...linear.M4) {
		var world linear.M4
		world.I()
		for _, m := range ms {
			world.Mul(&world, &m)
		}
		data := g.nodes[n-1].data
		if w := g.data[data].world; w != world {
			t.Fatalf("Graph.Ignore: invalid world for node %d\nhave %v\nwant %v", n, w, world)
		}
	}

	g.Update()
	check(n1, m1)
	check(n11, m1, m2)
	check(n12, m1, m3)
	check(n121, m1, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n1, true)
	g.Update()
	check(n1, m1)
	check(n11, m1, m2)
	check(n12, m1, m3)
	check(n121, m1, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Get(n1).(*inode).local = m6
	g.Update()
	check(n1, m1)
	check(n11, m1, m2)
	check(n12, m1, m3)
	check(n121, m1, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n1, false)
	g.Update()
	check(n1, m6)
	check(n11, m6, m2)
	check(n12, m6, m3)
	check(n121, m6, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n11, true)
	g.Get(n11).(*inode).local = m1
	g.Update()
	check(n1, m6)
	check(n11, m6, m2)
	check(n12, m6, m3)
	check(n121, m6, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n11, false)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m3)
	check(n121, m6, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n12, true)
	g.Get(n12).(*inode).local = m2
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m3)
	check(n121, m6, m3, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n12, false)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m4)
	check(n311, m3, m4, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n3111, true)
	g.Ignore(n2, true)
	g.Get(n31).(*inode).local = m2
	g.Get(n2).(*inode).local = m1
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m2)
	check(n3, m3)
	check(n31, m3, m2)
	check(n311, m3, m2, m5)
	check(n3111, m3, m4, m5, m6)

	g.Ignore(n2, false)
	g.Ignore(n3111, false)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m1)
	check(n3, m3)
	check(n31, m3, m2)
	check(n311, m3, m2, m5)
	check(n3111, m3, m2, m5, m6)

	g.SetWorld(m1)
	g.Ignore(n1, true)
	g.Ignore(n2, true)
	g.Ignore(n311, true)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m1)
	check(n3, m1, m3)
	check(n31, m1, m3, m2)
	check(n311, m3, m2, m5)
	check(n3111, m3, m2, m5, m6)

	g.Ignore(n2, false)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m1, m1)
	check(n3, m1, m3)
	check(n31, m1, m3, m2)
	check(n311, m3, m2, m5)
	check(n3111, m3, m2, m5, m6)

	g.Ignore(n311, false)
	g.Ignore(n3111, true)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m1, m1)
	check(n3, m1, m3)
	check(n31, m1, m3, m2)
	check(n311, m1, m3, m2, m5)
	check(n3111, m3, m2, m5, m6)

	g.Ignore(n3111, false)
	g.Update()
	check(n1, m6)
	check(n11, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m1, m1)
	check(n3, m1, m3)
	check(n31, m1, m3, m2)
	check(n311, m1, m3, m2, m5)
	check(n3111, m1, m3, m2, m5, m6)

	g.Ignore(n1, false)
	g.Ignore(n12, true)
	g.Update()
	check(n1, m1, m6)
	check(n11, m1, m6, m1)
	check(n12, m6, m2)
	check(n121, m6, m2, m4)
	check(n2, m1, m1)
	check(n3, m1, m3)
	check(n31, m1, m3, m2)
	check(n311, m1, m3, m2, m5)
	check(n3111, m1, m3, m2, m5, m6)

	g.Ignore(n12, false)
	g.Update()
	check(n1, m1, m6)
	check(n11, m1, m6, m1)
	check(n12, m1, m6, m2)
	check(n121, m1, m6, m2, m4)
	check(n2, m1, m1)
	check(n3, m1, m3)
	check(n31, m1, m3, m2)
	check(n311, m1, m3, m2, m5)
	check(n3111, m1, m3, m2, m5, m6)
}
