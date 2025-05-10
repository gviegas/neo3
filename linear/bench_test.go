// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

import (
	"testing"
)

func BenchmarkDot(b *testing.B) {
	v := V3{-2, 3, 9}
	w := V3{6, -3, 7}
	var d, e float32
	b.Run("V3.Dot", func(b *testing.B) {
		for b.Loop() {
			d = v.Dot(&w)
		}
	})
	b.Run("V3.bDotValue", func(b *testing.B) {
		for b.Loop() {
			e = v.bDotValue(w)
		}
	})
	b.Log(d, e)
}

// v and w passed on the stack.
func (v V3) bDotValue(w V3) (d float32) {
	for i := range v {
		d += v[i] * w[i]
	}
	return
}

func BenchmarkCross(b *testing.B) {
	l := V3{1, 0, 0}
	r := V3{0, 1, 0}
	var v, u, w V3
	b.Run("V3.Cross", func(b *testing.B) {
		for b.Loop() {
			v.Cross(&l, &r)
		}
	})
	b.Run("bCrossValue", func(b *testing.B) {
		for b.Loop() {
			u = bCrossValue(l, r)
		}
	})
	b.Run("V3.bCrossNoAlias", func(b *testing.B) {
		for b.Loop() {
			w.bCrossNoAlias(&l, &r)
		}
	})
	b.Log(v, u, w)
}

// l, r and v passed on the stack.
func bCrossValue(l, r V3) (v V3) {
	v[0] = l[1]*r[2] - l[2]*r[1]
	v[1] = l[2]*r[0] - l[0]*r[2]
	v[2] = l[0]*r[1] - l[1]*r[0]
	return
}

// v updated in place.
func (v *V3) bCrossNoAlias(l, r *V3) {
	v[0] = l[1]*r[2] - l[2]*r[1]
	v[1] = l[2]*r[0] - l[0]*r[2]
	v[2] = l[0]*r[1] - l[1]*r[0]
}

func BenchmarkMulV3(b *testing.B) {
	m := M3{
		{-1, 4, -7},
		{2, -5, 8},
		{-3, 6, -9},
	}
	w := V3{3, 2, -1}
	var v, u V3
	b.Run("V3.Mul", func(b *testing.B) {
		for b.Loop() {
			v.Mul(&m, &w)
		}
	})
	b.Run("V3.bMulNoAlias", func(b *testing.B) {
		for b.Loop() {
			u.bMulNoAlias(&m, &w)
		}
	})
	b.Log(v, u)
}

// v updated in place.
func (v *V3) bMulNoAlias(m *M3, w *V3) {
	*v = V3{}
	for i := range v {
		for j := range v {
			v[i] += m[j][i] * w[j]
		}
	}
}

func BenchmarkMulV4(b *testing.B) {
	m := M4{
		{-1, 5, -9, -13},
		{2, 6, 10, -14},
		{-3, 7, -11, 15},
		{4, -8, 12, -16},
	}
	w := V4{-4, 3, 2, -1}
	var v, u V4
	b.Run("V4.Mul", func(b *testing.B) {
		for b.Loop() {
			v.Mul(&m, &w)
		}
	})
	b.Run("V4.bMulNoAlias", func(b *testing.B) {
		for b.Loop() {
			u.bMulNoAlias(&m, &w)
		}
	})
	b.Log(v, u)
}

// v updated in place.
func (v *V4) bMulNoAlias(m *M4, w *V4) {
	*v = V4{}
	for i := range v {
		for j := range v {
			v[i] += m[j][i] * w[j]
		}
	}
}

func BenchmarkMulM3(b *testing.B) {
	l := M3{
		{1, 4, 7},
		{2, 5, 8},
		{3, 6, 9},
	}
	r := M3{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	var m, n M3
	b.Run("M3.Mul", func(b *testing.B) {
		for b.Loop() {
			m.Mul(&l, &r)
		}
	})
	b.Run("M3.bMulNoAlias", func(b *testing.B) {
		for b.Loop() {
			n.bMulNoAlias(&l, &r)
		}
	})
	b.Log("\n", m, "\n", n)
}

// m updated in place.
func (m *M3) bMulNoAlias(l, r *M3) {
	for i := range m {
		for j := range m {
			m[i][j] = 0
			for k := range m {
				m[i][j] += l[k][j] * r[i][k]
			}
		}
	}
}

func BenchmarkMulM4(b *testing.B) {
	l := M4{
		{1, 5, 9, 13},
		{2, 6, 10, 14},
		{3, 7, 11, 15},
		{4, 8, 12, 16},
	}
	r := M4{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
		{13, 14, 15, 16},
	}
	var m, n M4
	b.Run("M4.Mul", func(b *testing.B) {
		for b.Loop() {
			m.Mul(&l, &r)
		}
	})
	b.Run("M4.bMulNoAlias", func(b *testing.B) {
		for b.Loop() {
			n.bMulNoAlias(&l, &r)
		}
	})
	b.Log("\n", m, "\n", n)
}

// m updated in place.
func (m *M4) bMulNoAlias(l, r *M4) {
	for i := range m {
		for j := range m {
			m[i][j] = 0
			for k := range m {
				m[i][j] += l[k][j] * r[i][k]
			}
		}
	}
}

func BenchmarkMulQ(b *testing.B) {
	l := Q{
		V: V3{0.5, 0.5, -0.5},
		R: 0.5,
	}
	r := Q{
		V: V3{-0.5, -0.5, 0.5},
		R: 0.5,
	}
	var q, p Q
	b.Run("Q.Mul", func(b *testing.B) {
		for b.Loop() {
			q.Mul(&l, &r)
		}
	})
	b.Run("Q.bMulQValue", func(b *testing.B) {
		for b.Loop() {
			p = bMulQValue(l, r)
		}
	})
	b.Log(q, p)
}

// l, r and q passed on the stack.
func bMulQValue(l, r Q) (q Q) {
	var v, w V3
	v.Scale(r.R, &l.V)
	w.Scale(l.R, &r.V)
	v.Add(&v, &w)
	w.Cross(&l.V, &r.V)
	d := l.V.Dot(&r.V)
	q.V.Add(&v, &w)
	q.R = l.R*r.R - d
	return
}

func BenchmarkInvert3(b *testing.B) {
	h := M3{
		{0, 1, 1},
		{3, 0, -1},
		{-1, 1, 0},
	}
	var m, n M3
	b.Run("M3.Invert", func(b *testing.B) {
		for b.Loop() {
			m.Invert(&h)
		}
	})
	b.Run("M3.bInvertNoAlias", func(b *testing.B) {
		for b.Loop() {
			n.bInvertNoAlias(&h)
		}
	})
	b.Log("\n", m, "\n", n)
}

// m updated in place.
func (m *M3) bInvertNoAlias(n *M3) {
	s0 := n[1][1]*n[2][2] - n[1][2]*n[2][1]
	s1 := n[1][0]*n[2][2] - n[1][2]*n[2][0]
	s2 := n[1][0]*n[2][1] - n[1][1]*n[2][0]
	idet := 1 / (n[0][0]*s0 - n[0][1]*s1 + n[0][2]*s2)
	m[0][0] = s0 * idet
	m[0][1] = -(n[0][1]*n[2][2] - n[0][2]*n[2][1]) * idet
	m[0][2] = (n[0][1]*n[1][2] - n[0][2]*n[1][1]) * idet
	m[1][0] = -s1 * idet
	m[1][1] = (n[0][0]*n[2][2] - n[0][2]*n[2][0]) * idet
	m[1][2] = -(n[0][0]*n[1][2] - n[0][2]*n[1][0]) * idet
	m[2][0] = s2 * idet
	m[2][1] = -(n[0][0]*n[2][1] - n[0][1]*n[2][0]) * idet
	m[2][2] = (n[0][0]*n[1][1] - n[0][1]*n[1][0]) * idet
}

func BenchmarkInvert4(b *testing.B) {
	h := M4{
		{0, 1, 1, -3},
		{3, 0, -1, 0},
		{-1, 1, 0, 3},
		{1, 0, -3, 0},
	}
	var m, n M4
	b.Run("M4.Invert", func(b *testing.B) {
		for b.Loop() {
			m.Invert(&h)
		}
	})
	b.Run("M4.bInvertNoAlias", func(b *testing.B) {
		for b.Loop() {
			n.bInvertNoAlias(&h)
		}
	})
	b.Log("\n", m, "\n", n)
}

// m updated in place.
func (m *M4) bInvertNoAlias(n *M4) {
	s0 := n[0][0]*n[1][1] - n[0][1]*n[1][0]
	s1 := n[0][0]*n[1][2] - n[0][2]*n[1][0]
	s2 := n[0][0]*n[1][3] - n[0][3]*n[1][0]
	s3 := n[0][1]*n[1][2] - n[0][2]*n[1][1]
	s4 := n[0][1]*n[1][3] - n[0][3]*n[1][1]
	s5 := n[0][2]*n[1][3] - n[0][3]*n[1][2]
	c0 := n[2][0]*n[3][1] - n[2][1]*n[3][0]
	c1 := n[2][0]*n[3][2] - n[2][2]*n[3][0]
	c2 := n[2][0]*n[3][3] - n[2][3]*n[3][0]
	c3 := n[2][1]*n[3][2] - n[2][2]*n[3][1]
	c4 := n[2][1]*n[3][3] - n[2][3]*n[3][1]
	c5 := n[2][2]*n[3][3] - n[2][3]*n[3][2]
	idet := 1 / (s0*c5 - s1*c4 + s2*c3 + s3*c2 - s4*c1 + s5*c0)
	m[0][0] = (c5*n[1][1] - c4*n[1][2] + c3*n[1][3]) * idet
	m[0][1] = (-c5*n[0][1] + c4*n[0][2] - c3*n[0][3]) * idet
	m[0][2] = (s5*n[3][1] - s4*n[3][2] + s3*n[3][3]) * idet
	m[0][3] = (-s5*n[2][1] + s4*n[2][2] - s3*n[2][3]) * idet
	m[1][0] = (-c5*n[1][0] + c2*n[1][2] - c1*n[1][3]) * idet
	m[1][1] = (c5*n[0][0] - c2*n[0][2] + c1*n[0][3]) * idet
	m[1][2] = (-s5*n[3][0] + s2*n[3][2] - s1*n[3][3]) * idet
	m[1][3] = (s5*n[2][0] - s2*n[2][2] + s1*n[2][3]) * idet
	m[2][0] = (c4*n[1][0] - c2*n[1][1] + c0*n[1][3]) * idet
	m[2][1] = (-c4*n[0][0] + c2*n[0][1] - c0*n[0][3]) * idet
	m[2][2] = (s4*n[3][0] - s2*n[3][1] + s0*n[3][3]) * idet
	m[2][3] = (-s4*n[2][0] + s2*n[2][1] - s0*n[2][3]) * idet
	m[3][0] = (-c3*n[1][0] + c1*n[1][1] - c0*n[1][2]) * idet
	m[3][1] = (c3*n[0][0] - c1*n[0][1] + c0*n[0][2]) * idet
	m[3][2] = (-s3*n[3][0] + s1*n[3][1] - s0*n[3][2]) * idet
	m[3][3] = (s3*n[2][0] - s1*n[2][1] + s0*n[2][2]) * idet
}

func BenchmarkRotate(b *testing.B) {
	var m, o M3
	var n, p M4
	var q Q
	angle := float32(3.14159 / 6)
	axis := V3{0, 10, 0} // will be normalized
	b.Run("M3.Rotate", func(b *testing.B) {
		for b.Loop() {
			m.Rotate(angle, &axis)
		}
	})
	b.Run("M4.Rotate", func(b *testing.B) {
		for b.Loop() {
			n.Rotate(angle, &axis)
		}
	})
	b.Run("Q.Rotate", func(b *testing.B) {
		for b.Loop() {
			q.Rotate(angle, &axis)
		}
	})
	b.Run("M3.RotateQ", func(b *testing.B) {
		for b.Loop() {
			o.RotateQ(&q)
		}
	})
	b.Run("M4.RotateQ", func(b *testing.B) {
		for b.Loop() {
			p.RotateQ(&q)
		}
	})
	b.Log("\n", m, "\n", n, "\n", q, "\n", o, "\n", p)
}
