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
		for i := 0; i < b.N; i++ {
			d = v.Dot(&w)
		}
	})
	b.Run("V3.bDotValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
			v.Cross(&l, &r)
		}
	})
	b.Run("bCrossValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			u = bCrossValue(l, r)
		}
	})
	b.Run("V3.bCrossNoAlias", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
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

// v updated in-place.
func (v *V3) bCrossNoAlias(l, r *V3) {
	v[0] = l[1]*r[2] - l[2]*r[1]
	v[1] = l[2]*r[0] - l[0]*r[2]
	v[2] = l[0]*r[1] - l[1]*r[0]
}

func BenchmarkMulV(b *testing.B) {
	m := M3{
		{-1, 4, -7},
		{2, -5, 8},
		{-3, 6, -9},
	}
	w := V3{3, 2, -1}
	var v, u V3
	b.Run("V3.Mul", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v.Mul(&m, &w)
		}
	})
	b.Run("V3.bMulNoAlias", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
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

func BenchmarkMulM(b *testing.B) {
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
		for i := 0; i < b.N; i++ {
			m.Mul(&l, &r)
		}
	})
	b.Run("M3.bMulNoAlias", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			n.bMulNoAlias(&l, &r)
		}
	})
	b.Log("\n", m, "\n", n)
}

// m updated in-place.
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

func BenchmarkInvert(b *testing.B) {
	h := M3{
		{0, 1, 1},
		{3, 0, -1},
		{-1, 1, 0},
	}
	var m, n M3
	b.Run("M3.Invert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.Invert(&h)
		}
	})
	b.Run("M3.bInvertNoAlias", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			n.bInvertNoAlias(&h)
		}
	})
	b.Log("\n", m, "\n", n)
}

// m updated in-place.
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
