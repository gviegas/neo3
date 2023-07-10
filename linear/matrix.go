// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

import (
	"math"
)

// M3 is a column-major 3x3 matrix of float32.
type M3 [3]V3

// I3 returns M3's identity.
func I3() M3 { return M3{{1}, {1: 1}, {2: 1}} }

// I makes m an identity matrix.
func (m *M3) I() { *m = I3() }

// Mul sets m to contain l ⋅ r.
func (m *M3) Mul(l, r *M3) {
	var n M3
	for i := range m {
		for j := range m {
			for k := range m {
				n[i][j] += l[k][j] * r[i][k]
			}
		}
	}
	*m = n
}

// Transpose sets m to contain the transpose of n.
func (m *M3) Transpose(n *M3) {
	for i := range m {
		m[i][i] = n[i][i]
		for j := i + 1; j < len(m); j++ {
			m[i][j], m[j][i] = n[j][i], n[i][j]
		}
	}
}

// Invert sets m to contain the inverse of n.
func (m *M3) Invert(n *M3) {
	var h M3
	s0 := n[1][1]*n[2][2] - n[1][2]*n[2][1]
	s1 := n[1][0]*n[2][2] - n[1][2]*n[2][0]
	s2 := n[1][0]*n[2][1] - n[1][1]*n[2][0]
	idet := 1 / (n[0][0]*s0 - n[0][1]*s1 + n[0][2]*s2)
	h[0][0] = s0 * idet
	h[0][1] = -(n[0][1]*n[2][2] - n[0][2]*n[2][1]) * idet
	h[0][2] = (n[0][1]*n[1][2] - n[0][2]*n[1][1]) * idet
	h[1][0] = -s1 * idet
	h[1][1] = (n[0][0]*n[2][2] - n[0][2]*n[2][0]) * idet
	h[1][2] = -(n[0][0]*n[1][2] - n[0][2]*n[1][0]) * idet
	h[2][0] = s2 * idet
	h[2][1] = -(n[0][0]*n[2][1] - n[0][1]*n[2][0]) * idet
	h[2][2] = (n[0][0]*n[1][1] - n[0][1]*n[1][0]) * idet
	*m = h
}

// Scale sets m to contain a scale in the x, y and z axe.
func (m *M3) Scale(x, y, z float32) { *m = M3{{x}, {1: y}, {2: z}} }

// Rotate sets m to contain a rotation of angle radians
// around axis.
func (m *M3) Rotate(angle float32, axis *V3) {
	var v V3
	v.Norm(axis)
	x, y, z := v[0], v[1], v[2]
	xx, yy, zz := x*x, y*y, z*z
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	ic := 1 - c
	icxy, icxz, icyz := ic*x*y, ic*x*z, ic*y*z
	sx, sy, sz := s*x, s*y, s*z
	m[0][0] = c + ic*xx
	m[0][1] = icxy + sz
	m[0][2] = icxz - sy
	m[1][0] = icxy - sz
	m[1][1] = c + ic*yy
	m[1][2] = icyz + sx
	m[2][0] = icxz + sy
	m[2][1] = icyz - sx
	m[2][2] = c + ic*zz
}

// RotateQ sets m to contain the rotation that q describes.
func (m *M3) RotateQ(q *Q) {
	v := V4{q.V[0], q.V[1], q.V[2], q.R}
	v.Norm(&v)
	x, y, z, w := v[0], v[1], v[2], v[3]
	xx2, xy2, xz2, xw2 := 2*x*x, 2*x*y, 2*x*z, 2*x*w
	yy2, yz2, yw2 := 2*y*y, 2*y*z, 2*y*w
	zz2, zw2 := 2*z*z, 2*z*w
	m[0][0] = 1 - yy2 - zz2
	m[0][1] = xy2 + zw2
	m[0][2] = xz2 - yw2
	m[1][0] = xy2 - zw2
	m[1][1] = 1 - xx2 - zz2
	m[1][2] = yz2 + xw2
	m[2][0] = xz2 + yw2
	m[2][1] = yz2 - xw2
	m[2][2] = 1 - xx2 - yy2
}

// M4 is a column-major 4x4 matrix of float32.
type M4 [4]V4

// I4 returns M4's identity.
func I4() M4 { return M4{{1}, {1: 1}, {2: 1}, {3: 1}} }

// I makes m an identity matrix.
func (m *M4) I() { *m = I4() }

// Mul sets m to contain l ⋅ r.
func (m *M4) Mul(l, r *M4) {
	var n M4
	for i := range m {
		for j := range m {
			for k := range m {
				n[i][j] += l[k][j] * r[i][k]
			}
		}
	}
	*m = n
}

// Transpose sets m to contain the transpose of n.
func (m *M4) Transpose(n *M4) {
	for i := range m {
		m[i][i] = n[i][i]
		for j := i + 1; j < len(m); j++ {
			m[i][j], m[j][i] = n[j][i], n[i][j]
		}
	}
}

// Invert sets m to contain the inverse of n.
func (m *M4) Invert(n *M4) {
	var h M4
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
	h[0][0] = (c5*n[1][1] - c4*n[1][2] + c3*n[1][3]) * idet
	h[0][1] = (-c5*n[0][1] + c4*n[0][2] - c3*n[0][3]) * idet
	h[0][2] = (s5*n[3][1] - s4*n[3][2] + s3*n[3][3]) * idet
	h[0][3] = (-s5*n[2][1] + s4*n[2][2] - s3*n[2][3]) * idet
	h[1][0] = (-c5*n[1][0] + c2*n[1][2] - c1*n[1][3]) * idet
	h[1][1] = (c5*n[0][0] - c2*n[0][2] + c1*n[0][3]) * idet
	h[1][2] = (-s5*n[3][0] + s2*n[3][2] - s1*n[3][3]) * idet
	h[1][3] = (s5*n[2][0] - s2*n[2][2] + s1*n[2][3]) * idet
	h[2][0] = (c4*n[1][0] - c2*n[1][1] + c0*n[1][3]) * idet
	h[2][1] = (-c4*n[0][0] + c2*n[0][1] - c0*n[0][3]) * idet
	h[2][2] = (s4*n[3][0] - s2*n[3][1] + s0*n[3][3]) * idet
	h[2][3] = (-s4*n[2][0] + s2*n[2][1] - s0*n[2][3]) * idet
	h[3][0] = (-c3*n[1][0] + c1*n[1][1] - c0*n[1][2]) * idet
	h[3][1] = (c3*n[0][0] - c1*n[0][1] + c0*n[0][2]) * idet
	h[3][2] = (-s3*n[3][0] + s1*n[3][1] - s0*n[3][2]) * idet
	h[3][3] = (s3*n[2][0] - s1*n[2][1] + s0*n[2][2]) * idet
	*m = h
}

// Scale sets m to contain a scale in the x, y and z axe.
func (m *M4) Scale(x, y, z float32) { *m = M4{{x}, {1: y}, {2: z}, {3: 1}} }

// Rotate sets m to contain a rotation of angle radians
// around axis.
func (m *M4) Rotate(angle float32, axis *V3) {
	var v V3
	v.Norm(axis)
	x, y, z := v[0], v[1], v[2]
	xx, yy, zz := x*x, y*y, z*z
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	ic := 1 - c
	icxy, icxz, icyz := ic*x*y, ic*x*z, ic*y*z
	sx, sy, sz := s*x, s*y, s*z
	m[0][0] = c + ic*xx
	m[0][1] = icxy + sz
	m[0][2] = icxz - sy
	m[0][3] = 0
	m[1][0] = icxy - sz
	m[1][1] = c + ic*yy
	m[1][2] = icyz + sx
	m[1][3] = 0
	m[2][0] = icxz + sy
	m[2][1] = icyz - sx
	m[2][2] = c + ic*zz
	m[2][3] = 0
	m[3] = V4{0, 0, 0, 1}
}

// RotateQ sets m to contain the rotation that q describes.
func (m *M4) RotateQ(q *Q) {
	v := V4{q.V[0], q.V[1], q.V[2], q.R}
	v.Norm(&v)
	x, y, z, w := v[0], v[1], v[2], v[3]
	xx2, xy2, xz2, xw2 := 2*x*x, 2*x*y, 2*x*z, 2*x*w
	yy2, yz2, yw2 := 2*y*y, 2*y*z, 2*y*w
	zz2, zw2 := 2*z*z, 2*z*w
	m[0][0] = 1 - yy2 - zz2
	m[0][1] = xy2 + zw2
	m[0][2] = xz2 - yw2
	m[0][3] = 0
	m[1][0] = xy2 - zw2
	m[1][1] = 1 - xx2 - zz2
	m[1][2] = yz2 + xw2
	m[1][3] = 0
	m[2][0] = xz2 + yw2
	m[2][1] = yz2 - xw2
	m[2][2] = 1 - xx2 - yy2
	m[2][3] = 0
	m[3] = V4{0, 0, 0, 1}
}

// Translate sets m to contain a translation in the x, y and z axe.
func (m *M4) Translate(x, y, z float32) {
	*m = M4{{1}, {1: 1}, {2: 1}, {x, y, z, 1}}
}

// LookAt sets m to contain a view transform.
func (m *M4) LookAt(center, eye, up *V3) {
	var f, s, u V3
	f.Sub(center, eye)
	f.Norm(&f)
	s.Cross(&f, up)
	s.Norm(&s)
	u.Cross(&f, &s)
	*m = M4{
		{s[0], u[0], -f[0]},
		{s[1], u[1], -f[1]},
		{s[2], u[2], -f[2]},
		{-s.Dot(eye), -u.Dot(eye), f.Dot(eye), 1},
	}
}

// Perspective sets m to contain a perspective projection.
func (m *M4) Perspective(yfov, aspectRatio, znear, zfar float32) {
	ct := 1 / float32(math.Tan(float64(yfov/2)))
	*m = M4{
		{0: ct / aspectRatio},
		{1: ct},
		{2: (zfar + znear) / (znear - zfar), 3: -1},
		{2: (zfar * znear * 2) / (znear - zfar)},
	}
}

// InfPerspective sets m to contain an infinite perspective projection.
func (m *M4) InfPerspective(yfov, aspectRatio, znear float32) {
	ct := 1 / float32(math.Tan(float64(yfov/2)))
	*m = M4{
		{0: ct / aspectRatio},
		{1: ct},
		{2: -1, 3: -1},
		{2: znear * -2},
	}
}

// Ortho sets m to contain a nonperspective projection.
func (m *M4) Ortho(xmag, ymag, znear, zfar float32) {
	*m = M4{
		{0: 1 / xmag},
		{1: 1 / ymag},
		{2: 2 / (znear - zfar)},
		{2: (zfar + znear) / (znear - zfar), 3: 1},
	}
}
