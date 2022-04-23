// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package linear implements math for 3D graphics.
package linear

import (
	"math"
)

// V3 is a 3-component vector of float32.
type V3 [3]float32

// Add sets v to contain l + r.
func (v *V3) Add(l, r *V3) {
	for i := range v {
		v[i] = l[i] + r[i]
	}
}

// Sub sets v to contain l - r.
func (v *V3) Sub(l, r *V3) {
	for i := range v {
		v[i] = l[i] - r[i]
	}
}

// Scale sets v to contain s ⋅ w.
func (v *V3) Scale(s float32, w *V3) {
	for i := range v {
		v[i] = s * w[i]
	}
}

// Dot returns v ⋅ w.
func (v *V3) Dot(w *V3) (d float32) {
	for i := range v {
		d += v[i] * w[i]
	}
	return
}

// Len returns the length of v.
func (v *V3) Len() float32 { return float32(math.Sqrt(float64(v.Dot(v)))) }

// Norm sets v to contain w normalized.
func (v *V3) Norm(w *V3) { v.Scale(1/w.Len(), w) }

// Cross sets v to contain l × r.
func (v *V3) Cross(l, r *V3) {
	v[0] = l[1]*r[2] - l[2]*r[1]
	v[1] = l[2]*r[0] - l[0]*r[2]
	v[2] = l[0]*r[1] - l[1]*r[0]
	return
}

// Mul sets v to contain m ⋅ w.
func (v *V3) Mul(m *M3, w *V3) {
	*v = V3{}
	for i := range v {
		for j := range v {
			v[i] += m[j][i] * w[j]
		}
	}
}

// V4 is a 4-component vector of float32.
type V4 [4]float32

// Add sets v to contain l + r.
func (v *V4) Add(l, r *V4) {
	for i := range v {
		v[i] = l[i] + r[i]
	}
}

// Sub sets v to contain l - r.
func (v *V4) Sub(l, r *V4) {
	for i := range v {
		v[i] = l[i] - r[i]
	}
}

// Scale sets v to contain s ⋅ w.
func (v *V4) Scale(s float32, w *V4) {
	for i := range v {
		v[i] = s * w[i]
	}
}

// Dot returns v ⋅ w.
func (v *V4) Dot(w *V4) (d float32) {
	for i := range v {
		d += v[i] * w[i]
	}
	return
}

// Len returns the length of v.
func (v *V4) Len() float32 { return float32(math.Sqrt(float64(v.Dot(v)))) }

// Norm sets v to contain w normalized.
func (v *V4) Norm(w *V4) { v.Scale(1/w.Len(), w) }

// Mul sets v to contain m ⋅ w.
func (v *V4) Mul(m *M4, w *V4) {
	*v = V4{}
	for i := range v {
		for j := range v {
			v[i] += m[j][i] * w[j]
		}
	}
}
