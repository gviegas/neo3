// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

import (
	"math"
	"testing"
)

func TestV3(t *testing.T) {
	var u V3
	v := V3{1, 2, 4}
	w := V3{0, -1, 2}

	if u.Add(&v, &w); u != (V3{1, 1, 6}) {
		t.Fatalf("V3.Add\nhave %v\nwant [1 1 6]", u)
	}
	if u.Sub(&v, &w); u != (V3{1, 3, 2}) {
		t.Fatalf("V3.Sub\nhave %v\nwant [1 3 2]", u)
	}
	if u.Scale(-1, &v); u != (V3{-1, -2, -4}) {
		t.Fatalf("V3.Scale\nhave %v\nwant [-1 -2 -4]", u)
	}
	if u.Scale(2, &w); u != (V3{0, -2, 4}) {
		t.Fatalf("V3.Scale\nhave %v\nwant [0 -2 4]", u)
	}
	if d := v.Dot(&w); d != 6 {
		t.Fatalf("V3.Dot\nhave %v\nwant 6\n", d)
	}
	if d := v.Dot(&v); d != 21 {
		t.Fatalf("V3.Dot\nhave %v\nwant 21\n", d)
	}
	if l := v.Len(); l != float32(math.Sqrt(21)) {
		t.Fatalf("V3.Len\nhave %v\nwant %v\n", l, math.Sqrt(21))
	}
	if l := w.Len(); l != float32(math.Sqrt(5)) {
		t.Fatalf("V3.Len\nhave %v\nwant %v\n", l, math.Sqrt(5))
	}

	v = V3{0, 0, -2}
	w = V3{0, 4, 0}

	if v.Norm(&v); v != (V3{0, 0, -1}) {
		t.Fatalf("V3.Norm\nhave %v\nwant [0 0 -1]", v)
	}
	if w.Norm(&w); w != (V3{0, 1, 0}) {
		t.Fatalf("V3.Norm\nhave %v\nwant [0 1 0]", w)
	}
	if u.Cross(&v, &w); u != (V3{1, 0, 0}) {
		t.Fatalf("V3.Cross\nhave %v\nwant [1 0 0]", u)
	}
	if u.Cross(&w, &v); u != (V3{-1, 0, 0}) {
		t.Fatalf("V3.Cross\nhave %v\nwant [-1 0 0]", u)
	}

	m := M3{
		{2, 0, 1},
		{1, 3, 2},
		{4, 2, 3},
	}
	v = V3{-1, 0, 1}

	if u.Mul(&m, &v); u != (V3{2, 2, 2}) {
		t.Fatalf("V3.Mul\nhave %v\nwant [2 2 2]", u)
	}
	m.I()
	if u.Mul(&m, &v); u != v {
		t.Fatalf("V3.Mul\nhave %v\nwant %v", u, v)
	}
}

func TestV4(t *testing.T) {
	var u V4
	v := V4{1, 2, 4, -2}
	w := V4{0, -1, 2, 3}

	if u.Add(&v, &w); u != (V4{1, 1, 6, 1}) {
		t.Fatalf("V4.Add\nhave %v\nwant [1 1 6 1]", u)
	}
	if u.Sub(&v, &w); u != (V4{1, 3, 2, -5}) {
		t.Fatalf("V4.Sub\nhave %v\nwant [1 3 2 -5]", u)
	}
	if u.Scale(-1, &v); u != (V4{-1, -2, -4, 2}) {
		t.Fatalf("V4.Scale\nhave %v\nwant [-1 -2 -4 2]", u)
	}
	if u.Scale(2, &w); u != (V4{0, -2, 4, 6}) {
		t.Fatalf("V4.Scale\nhave %v\nwant [0 -2 4 6]", u)
	}
	if d := v.Dot(&w); d != 0 {
		t.Fatalf("V4.Dot\nhave %v\nwant 0\n", d)
	}
	if d := v.Dot(&v); d != 25 {
		t.Fatalf("V4.Dot\nhave %v\nwant 25\n", d)
	}
	if l := v.Len(); l != float32(math.Sqrt(25)) {
		t.Fatalf("V4.Len\nhave %v\nwant %v\n", l, math.Sqrt(25))
	}
	if l := w.Len(); l != float32(math.Sqrt(14)) {
		t.Fatalf("V4.Len\nhave %v\nwant %v\n", l, math.Sqrt(14))
	}

	v = V4{0, 0, 0, -2}
	w = V4{4, 0, 0, 0}

	if v.Norm(&v); v != (V4{0, 0, 0, -1}) {
		t.Fatalf("V4.Norm\nhave %v\nwant [0 0 0 -1]", v)
	}
	if w.Norm(&w); w != (V4{1, 0, 0, 0}) {
		t.Fatalf("V4.Norm\nhave %v\nwant [1 0 0 0]", w)
	}

	m := M4{
		{2, 0, 1, -1},
		{1, 3, 2, -3},
		{0, -4, 2, 3},
		{1, 2, -1, 0},
	}
	v = V4{-1, 0, 1, 2}

	if u.Mul(&m, &v); u != (V4{0, 0, -1, 4}) {
		t.Fatalf("V4.Mul\nhave %v\nwant [0 0 -1 4]", u)
	}
	m.I()
	if u.Mul(&m, &v); u != v {
		t.Fatalf("V4.Mul\nhave %v\nwant %v", u, v)
	}
}

func TestM3(t *testing.T) {
	var l M3
	m := M3{
		{1, 4, 7},
		{2, 5, 8},
		{3, 6, 9},
	}
	n := M3{
		{0, 1, 0},
		{0, 0, 1},
		{1, 0, 0},
	}

	if l.I(); l != (M3{{1}, {0, 1}, {0, 0, 1}}) {
		t.Fatalf("M3.I\nhave %v\nwant [%v %v %v]", l, V3{1}, V3{0, 1}, V3{0, 0, 1})
	}
	if l.Mul(&m, &n); l != (M3{m[1], m[2], m[0]}) {
		t.Fatalf("M3.Mul\nhave %v\nwant [%v %v %v]", l, m[1], m[2], m[0])
	}
	if l.Mul(&n, &m); l != (M3{{7, 1, 4}, {8, 2, 5}, {9, 3, 6}}) {
		t.Fatalf("M3.Mul\nhave %v\nwant %v", l, M3{{7, 1, 4}, {8, 2, 5}, {9, 3, 6}})
	}
	if l.Transpose(&m); l != (M3{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}) {
		t.Fatalf("M3.Transpose\nhave %v\nwant %v", l, M3{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}})
	}
	if l.Invert(&n); l != (M3{n[1], n[2], n[0]}) {
		t.Fatalf("M3.Invert\nhave %v\nwant %v", l, M3{n[1], n[2], n[0]})
	}
}

func TestM4(t *testing.T) {
	var l M4
	m := M4{
		{1, 5, 9, 13},
		{2, 6, 10, 14},
		{3, 7, 11, 15},
		{4, 8, 12, 16},
	}
	n := M4{
		{0, 1, 0, 0},
		{1, 0, 0, 0},
		{0, 0, 0, 1},
		{0, 0, 1, 0},
	}
	o := M4{
		{4, 0, 0, 0},
		{0, 5, 0, 0},
		{0, 0, 2, 0},
		{3, -2, -5, 1},
	}

	if l.I(); l != (M4{{1}, {0, 1}, {0, 0, 1}, {0, 0, 0, 1}}) {
		t.Fatalf("M4.I\nhave %v\nwant [%v %v %v %v]", l, V4{1}, V4{0, 1}, V4{0, 0, 1}, V4{0, 0, 0, 1})
	}
	if l.Mul(&m, &n); l != (M4{m[1], m[0], m[3], m[2]}) {
		t.Fatalf("M4.Mul\nhave %v\nwant [%v %v %v %v]", l, m[1], m[0], m[3], m[2])
	}
	if l.Mul(&n, &m); l != (M4{{5, 1, 13, 9}, {6, 2, 14, 10}, {7, 3, 15, 11}, {8, 4, 16, 12}}) {
		t.Fatalf("M4.Mul\nhave %v\nwant %v", l, M4{{5, 1, 13, 9}, {6, 2, 14, 10}, {7, 3, 15, 11}, {8, 4, 16, 12}})
	}
	if l.Transpose(&m); l != (M4{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}, {13, 14, 15, 16}}) {
		t.Fatalf("M4.Transpose\nhave %v\nwant %v", l, M4{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}, {13, 14, 15, 16}})
	}
	if l.Invert(&o); l != (M4{
		{1 / o[0][0]},
		{1: 1 / o[1][1]},
		{2: 1 / o[2][2]},
		{-o[3][0] / o[0][0], -o[3][1] / o[1][1], -o[3][2] / o[2][2], o[3][3]},
	}) {
		t.Fatalf("M4.Invert\nhave %v\nwant %v", l, M4{
			{1 / o[0][0]},
			{1: 1 / o[1][1]},
			{2: 1 / o[2][2]},
			{-o[3][0] / o[0][0], -o[3][1] / o[1][1], -o[3][2] / o[2][2], o[3][3]},
		})
	}
}

func TestQ(t *testing.T) {
	var r Q
	q := Q{V: V3{1, 0, 0}, R: 3}
	p := Q{V: V3{0, 1, 0}, R: 3}

	if r.I(); r.V != (V3{}) || r.R != 1 {
		t.Fatalf("Q.I\nhave %v\nwant {[0 0 0] 1}", r)
	}
	if r.Mul(&q, &p); r.V != (V3{3, 3, 1}) || r.R != 9 {
		t.Fatalf("Q.Mul\nhave %v\nwant {[3 3 1] 9}", r)
	}
	if r.Mul(&p, &q); r.V != (V3{3, 3, -1}) || r.R != 9 {
		t.Fatalf("Q.Mul\nhave %v\nwant {[3 3 -1] 9}", r)
	}
	if q.Mul(&q, &q); q.V != (V3{6}) || q.R != 8 {
		t.Fatalf("Q.Mul\nhave %v\nwant {[6 0 0] 8}", q)
	}
}

func TestTRS(t *testing.T) {
	var x, r, s M4
	var q Q

	x.Translate(-1, -2, -3)
	q.Rotate(0, &V3{1})
	r.RotateQ(&q)
	s.Scale(5, 5, 5)
	x.Mul(&x, &r)
	x.Mul(&x, &s)
	if x != (M4{{5}, {1: 5}, {2: 5}, {-1, -2, -3, 1}}) {
		t.Fatalf("T*R*S\nhave %v\nwant %v", x, M4{{5}, {1: 5}, {2: 5}, {-1, -2, -3, 1}})
	}
	v := V4{1, 1, 1, 1}
	v.Mul(&x, &v)
	if v != (V4{4, 3, 2, 1}) {
		t.Fatalf("TRS*v\nhave %v\nwant %v", v, V4{4, 3, 2, 1})
	}
}

func TestMConv(t *testing.T) {
	i3 := I3()
	i4 := I4()

	var m3, n3 M3
	var m4, n4 M4

	m3.FromM4(&i4)
	m4.FromM3(&i3)
	if m3 != i3 {
		t.Fatalf("M3.FromM4()\nhave %v\nwant %v", m3, i3)
	}
	if m4 != i4 {
		t.Fatalf("M4.FromM3()\nhave %v\nwant %v", m4, i4)
	}

	m3.FromM4(&m4)
	m4.FromM3(&m3)
	if m3 != i3 {
		t.Fatalf("M3.FromM4()\nhave %v\nwant %v", m3, i3)
	}
	if m4 != i4 {
		t.Fatalf("M4.FromM3()\nhave %v\nwant %v", m4, i4)
	}

	m3.FromM4(&n4)
	m4.FromM3(&n3)
	if m3 != (M3{}) {
		t.Fatalf("M3.FromM4()\nhave %v\nwant %v", m3, M3{})
	}
	if m4 != (M4{{}, {}, {}, {0, 0, 0, 1}}) {
		t.Fatalf("M4.FromM3()\nhave %v\nwant %v", m4, M4{})
	}

	n4.Translate(-5, 10, -20)
	m3.FromM4(&n4)
	if m3 != i3 {
		t.Fatalf("M3.FromM4()\nhave %v\nwant %v", m3, i3)
	}

	n3.Scale(0.25, 1.5, -1)
	n4.Scale(n3[0][0], n3[1][1], n3[2][2])
	m4.FromM3(&n3)
	if m4 != n4 {
		t.Fatalf("M4.FromM3()\nhave %v\nwant %v", m4, n4)
	}
}
