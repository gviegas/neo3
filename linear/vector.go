// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package linear implements math for 3D graphics.
package linear

import (
	"math"
)

// V3 is a 3-component vector of float32.
type V3 [3]float32

// AddV3 returns v + w.
func AddV3(v, w V3) (u V3) {
	for i := range u {
		u[i] = v[i] + w[i]
	}
	return
}

// SubV3 returns v - w.
func SubV3(v, w V3) (u V3) {
	for i := range u {
		u[i] = v[i] - w[i]
	}
	return
}

// ScaleV3 returns s ⋅ v.
func ScaleV3(s float32, v V3) (u V3) {
	for i := range u {
		u[i] = s * v[i]
	}
	return
}

// DotV3 returns v ⋅ w.
func DotV3(v, w V3) (d float32) {
	for i := range v {
		d += v[i] * w[i]
	}
	return
}

// LenV3 returns the length of v.
func LenV3(v V3) float32 {
	return float32(math.Sqrt(float64(DotV3(v, v))))
}

// NormV3 returns v normalized.
func NormV3(v V3) V3 {
	return ScaleV3(1/LenV3(v), v)
}

// Cross returns v × w.
func Cross(v, w V3) (u V3) {
	u[0] = v[1]*w[2] - v[2]*w[1]
	u[1] = v[2]*w[0] - v[0]*w[2]
	u[2] = v[0]*w[1] - v[1]*w[0]
	return
}

// V4 is a 4-component vector of float32.
type V4 [4]float32

// AddV4 returns v + w.
func AddV4(v, w V4) (u V4) {
	for i := range u {
		u[i] = v[i] + w[i]
	}
	return
}

// SubV4 returns v - w.
func SubV4(v, w V4) (u V4) {
	for i := range u {
		u[i] = v[i] - w[i]
	}
	return
}

// ScaleV4 returns s ⋅ v.
func ScaleV4(s float32, v V4) (u V4) {
	for i := range u {
		u[i] = s * v[i]
	}
	return
}

// DotV4 returns v ⋅ w.
func DotV4(v, w V4) (d float32) {
	for i := range v {
		d += v[i] * w[i]
	}
	return
}

// LenV4 returns the length of v.
func LenV4(v V4) float32 {
	return float32(math.Sqrt(float64(DotV4(v, v))))
}

// NormV4 returns v normalized.
func NormV4(v V4) V4 {
	return ScaleV4(1/LenV4(v), v)
}
