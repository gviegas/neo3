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
	var v, u V3
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
	b.Log(v, u)
}

// l, r and v passed on the stack.
func bCrossValue(l, r V3) (v V3) {
	v[0] = l[1]*r[2] - l[2]*r[1]
	v[1] = l[2]*r[0] - l[0]*r[2]
	v[2] = l[0]*r[1] - l[1]*r[0]
	return
}
