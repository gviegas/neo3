// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

// Q is a quaternion of float32.
type Q struct {
	V V3
	R float32
}

// Mul sets q to contain l ⋅ r.
func (q *Q) Mul(l, r *Q) {
	var v, w V3
	v.Scale(r.R, &l.V)
	w.Scale(l.R, &r.V)
	v.Add(&v, &w)
	w.Cross(&l.V, &r.V)
	d := l.V.Dot(&r.V)
	q.V.Add(&v, &w)
	q.R = l.R*r.R - d
}
