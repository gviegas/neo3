// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
	"math"
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/linear"
)

func checkSlicesT(x, y []float32, t *testing.T, prefix string) {
	min := len(x)
	if n := len(y); n < min {
		min = n
	}
	for i := 0; i < min; i++ {
		if x[i] != y[i] {
			t.Fatalf("%s: slices differ at index %d\n%v != %v", prefix, i, x[i], y[i])
		}
	}
}

func TestFrameLayout(t *testing.T) {
	// [0:16]
	col := linear.V4{12, 34, 56, 78}
	vp := linear.M4{col, col, col, col}

	// [16:32]
	col = linear.V4{-12, -13, -14, -15}
	v := linear.M4{col, col, col, col}

	// [32:48]
	col = linear.V4{21, -43, 41, -87}
	p := linear.M4{col, col, col, col}

	// [48:49]
	tm := time.Now().Add(time.Second).Sub(time.Now())

	// [49:50]
	rnd := rand.Float32()

	// [50:56]
	bnd := driver.Viewport{X: 1, Y: 2, Width: 800, Height: 600, Znear: 0.1, Zfar: 100}

	var l FrameLayout
	l.SetVP(&vp)
	l.SetV(&v)
	l.SetP(&p)
	l.SetTime(tm)
	l.SetRand(rnd)
	l.SetBounds(&bnd)

	s := "FrameLayout."

	checkSlicesT(l[:16], unsafe.Slice((*float32)(unsafe.Pointer(&vp)), 16), t, s+"SetVP")
	checkSlicesT(l[16:32], unsafe.Slice((*float32)(unsafe.Pointer(&v)), 16), t, s+"SetV")
	checkSlicesT(l[32:48], unsafe.Slice((*float32)(unsafe.Pointer(&p)), 16), t, s+"SetP")
	if x := float32(tm.Seconds()); l[48] != x {
		t.Fatalf("%sSetTime:\nhave %f\nwant%f", s, l[48], x)
	}
	if l[49] != rnd {
		t.Fatalf("%sSetRand:\nhave %f\nwant%f", s, l[49], rnd)
	}
	if l[50] != bnd.X {
		t.Fatalf("%sSetBounds: Viewport.X\nhave %f\nwant%f", s, l[50], bnd.X)
	}
	if l[51] != bnd.Y {
		t.Fatalf("%sSetBounds: Viewport.Y\nhave %f\nwant%f", s, l[51], bnd.Y)
	}
	if l[52] != bnd.Width {
		t.Fatalf("%sSetBounds: Viewport.Width\nhave %f\nwant%f", s, l[52], bnd.Width)
	}
	if l[53] != bnd.Height {
		t.Fatalf("%sSetBounds: Viewport.Height\nhave %f\nwant%f", s, l[53], bnd.Height)
	}
	if l[54] != bnd.Znear {
		t.Fatalf("%sSetBounds: Viewport.Znear\nhave %f\nwant%f", s, l[54], bnd.Znear)
	}
	if l[55] != bnd.Zfar {
		t.Fatalf("%sSetBounds: Viewport.Zfar\nhave %f\nwant%f", s, l[55], bnd.Zfar)
	}
}

func TestLightLayout(t *testing.T) {
	// [0:1]
	unused := true

	// [1:2]
	typ := SpotLight

	// [2:3]
	intens := float32(1000)

	// [3:4]
	rng := float32(12)

	// [4:7]
	color := linear.V3{0.1, 0.2, 0.3}

	// [7:8]
	scale := float32(0.7854)

	// [8:11]
	pos := linear.V3{0.4, 0.5, 0.6}

	// [11:12]
	off := float32(1.0472)

	// [12:15]
	dir := linear.V3{0.5026, 0.5744, 0.6462}

	var l LightLayout
	l.SetUnused(unused)
	l.SetType(typ)
	l.SetIntensity(intens)
	l.SetRange(rng)
	l.SetColor(&color)
	l.SetAngScale(scale)
	l.SetPosition(&pos)
	l.SetAngOffset(off)
	l.SetDirection(&dir)

	s := "LightLayout."

	switch x := *(*int32)(unsafe.Pointer(&l[0])); x {
	case 0:
		if unused {
			t.Fatalf("%sSetUnused:\nhave false (0) \nwant true (1)", s)
		}
	case 1:
		if !unused {
			t.Fatalf("%sSetUnused:\nhave true (1) \nwant false (0)", s)
		}
	default:
		t.Fatalf("%sSetUnused: bad value\n%d", s, x)
	}
	if x := *(*int32)(unsafe.Pointer(&l[1])); x != typ {
		t.Fatalf("%sSetType:\nhave %d\nwant %d", s, x, typ)
	}
	if l[2] != intens {
		t.Fatalf("%sSetIntensity:\nhave %f\nwant %f", s, l[2], intens)
	}
	if l[3] != rng {
		t.Fatalf("%sSetRange:\nhave %f\nwant %f", s, l[3], rng)
	}
	checkSlicesT(l[4:7], color[:], t, s+"SetColor")
	if l[7] != scale {
		t.Fatalf("%sSetAngScale:\nhave %f\nwant %f", s, l[7], scale)
	}
	checkSlicesT(l[8:11], pos[:], t, s+"SetPosition")
	if l[11] != off {
		t.Fatalf("%sSetAngOffset:\nhave %f\nwant %f", s, l[11], off)
	}
	checkSlicesT(l[12:15], dir[:], t, s+"SetDirection")
}

func TestShadowLayout(t *testing.T) {
	// [0:1]
	unused := true

	// [16:32]
	var shdw linear.M4
	shdw.Ortho(-1, 1, -1, 1)
	shdw.Mul(&linear.M4{{0.5}, {1: -0.5}, {2: 1}, {0.5, 0.5, 0, 1}}, &shdw)

	var l ShadowLayout
	l.SetUnused(unused)
	l.SetShadow(&shdw)

	s := "ShadowLayout."

	switch x := *(*int32)(unsafe.Pointer(&l[0])); x {
	case 0:
		if unused {
			t.Fatalf("%sSetUnused:\nhave false (0) \nwant true (1)", s)
		}
	case 1:
		if !unused {
			t.Fatalf("%sSetUnused:\nhave true (1) \nwant false (0)", s)
		}
	default:
		t.Fatalf("%sSetUnused: bad value\n%d", s, x)
	}
	checkSlicesT(l[16:32], unsafe.Slice((*float32)(unsafe.Pointer(&shdw)), 16), t, s+"SetShadow")
}

func TestDrawableLayout(t *testing.T) {
	// [0:16]
	var wld linear.M4
	wld.Rotate(math.Pi/4, &linear.V3{0, 0.7071, 0.7071})

	// [16:28]
	var norm linear.M3
	norm.FromM4(&wld)
	norm.Invert(&norm)
	norm.Transpose(&norm)

	// [28:29]
	id := uint32(0x1d)

	var l DrawableLayout
	l.SetWorld(&wld)
	l.SetNormal(&norm)
	l.SetID(id)

	s := "DrawableLayout."

	checkSlicesT(l[:16], unsafe.Slice((*float32)(unsafe.Pointer(&wld)), 16), t, s+"SetWorld")
	checkSlicesT(l[16:28], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, s+"SetNormal")
	if x := *(*uint32)(unsafe.Pointer(&l[28])); x != id {
		t.Fatalf("%sSetID:\nhave %d\nwant %d", s, x, id)
	}
}

func TestMaterialLayout(t *testing.T) {
	// [0:4]
	color := linear.V4{0.1, 0.2, 0.3, 0.4}

	// [4:5], [5:6]
	metal, rough := float32(0.5), float32(0.6)

	// [6:7]
	scale := float32(0.7)

	// [7:8]
	strength := float32(0.8)

	// [8:11]
	emissive := linear.V3{0.9, 0.91, 0.92}

	// [11:12]
	cutoff := float32(0.93)

	// [12:13]
	flags := MatPBR | MatABlend | MatDoubleSided

	var l MaterialLayout
	l.SetColorFactor(&color)
	l.SetMetalRough(metal, rough)
	l.SetNormScale(scale)
	l.SetOccStrength(strength)
	l.SetEmisFactor(&emissive)
	l.SetAlphaCutoff(cutoff)
	l.SetFlags(flags)

	s := "MaterialLayout."

	checkSlicesT(l[:4], color[:], t, s+"SetColorFactor")
	if l[4] != metal || l[5] != rough {
		t.Fatalf("%sSetMetalRough:\nhave %f, %f\nwant %f, %f", s, l[4], l[5], metal, rough)
	}
	if l[6] != scale {
		t.Fatalf("%sSetNormScale:\nhave %f\nwant %f", s, l[6], scale)
	}
	if l[7] != strength {
		t.Fatalf("%sSetOccStrength:\nhave %f\nwant %f", s, l[7], strength)
	}
	checkSlicesT(l[8:11], emissive[:], t, s+"SetEmisFactor")
	if l[11] != cutoff {
		t.Fatalf("%sSetAlphaCutoff:\nhave %f\nwant %f", s, l[11], cutoff)
	}
	if x := *(*uint32)(unsafe.Pointer(&l[12])); x != flags {
		t.Fatalf("%sSetFlags:\nhave 0x%x\nwant 0x%x", s, x, flags)
	}
}

func TestJointLayout(t *testing.T) {
	// [0:12]
	var jnt linear.M4
	jnt.Rotate(math.Pi/3, &linear.V3{-0.7071, 0.7071, 0})

	// [12:24]
	var norm linear.M3
	norm.FromM4(&jnt)
	norm.Invert(&norm)
	norm.Transpose(&norm)

	var l JointLayout
	l.SetJoint(&jnt)
	l.SetNormal(&norm)

	s := "JointLayout."

	checkSlicesT(l[:12], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, s+"SetJoint")
	checkSlicesT(l[12:24], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, s+"SetNormal")
}
