// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

// M3 is a column-major 3x3 matrix of float32.
type M3 [3]V3

// I makes m an identity matrix.
func (m *M3) I() { *m = M3{{1}, {0, 1}, {0, 0, 1}} }

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

// M4 is a column-major 4x4 matrix of float32.
type M4 [4]V4

// I makes m an identity matrix.
func (m *M4) I() { *m = M4{{1}, {0, 1}, {0, 0, 1}, {0, 0, 0, 1}} }

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
