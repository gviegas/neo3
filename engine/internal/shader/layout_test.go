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
	for i := range vp {
		vp[i][i] += 1.0
	}

	// [16:32]
	col = linear.V4{-12, -13, -14, -15}
	v := linear.M4{col, col, col, col}
	for i := range vp {
		v[i][i] += 1.0
	}

	// [32:48]
	col = linear.V4{21, -43, 41, -87}
	p := linear.M4{col, col, col, col}
	for i := range vp {
		p[i][i] += 1.0
	}

	// [48:49]
	tm := time.Now().Add(time.Millisecond).Sub(time.Now())

	// [49:50]
	rnd := rand.Float32()

	// [50:56]
	bnd := driver.Viewport{X: 64, Y: 32, Width: 800, Height: 600, Znear: 1, Zfar: 1e-6}

	var l FrameLayout
	l.SetVP(&vp)
	l.SetV(&v)
	l.SetP(&p)
	l.SetTime(tm)
	l.SetRand(rnd)
	l.SetBounds(&bnd)

	s := "FrameLayout."

	checkSlicesT(l[:16], unsafe.Slice((*float32)(unsafe.Pointer(&vp)), 16), t, s+"SetVP")
	if x := l.VP(); x != vp {
		t.Fatalf("%sVP:\nhave %f\nwant %f", s, x, vp)
	}

	checkSlicesT(l[16:32], unsafe.Slice((*float32)(unsafe.Pointer(&v)), 16), t, s+"SetV")
	if x := l.V(); x != v {
		t.Fatalf("%sV:\nhave %f\nwant %f", s, x, v)
	}

	checkSlicesT(l[32:48], unsafe.Slice((*float32)(unsafe.Pointer(&p)), 16), t, s+"SetP")
	if x := l.P(); x != p {
		t.Fatalf("%sP:\nhave %f\nwant%f", s, x, p)
	}

	switch x, y := float64(l[48]), l.Time().Seconds(); {
	case math.Abs(x-tm.Seconds()) > 1e-6:
		t.Fatalf("%sSetTime:\nhave %f\nwant %f", s, x, tm.Seconds())
	case math.Abs(y-tm.Seconds()) > 1e-6:
		t.Fatalf("%sTime:\nhave %f\nwant %f", s, y, tm.Seconds())
	}

	switch x, y := l[49], l.Rand(); {
	case x != rnd:
		t.Fatalf("%sSetRand:\nhave %f\nwant %f", s, x, rnd)
	case y != rnd:
		t.Fatalf("%sRand:\nhave %f\nwant %f", s, y, rnd)
	}

	if l[50] != bnd.X {
		t.Fatalf("%sSetBounds: Viewport.X\nhave %f\nwant %f", s, l[50], bnd.X)
	}
	if l[51] != bnd.Y {
		t.Fatalf("%sSetBounds: Viewport.Y\nhave %f\nwant %f", s, l[51], bnd.Y)
	}
	if l[52] != bnd.Width {
		t.Fatalf("%sSetBounds: Viewport.Width\nhave %f\nwant %f", s, l[52], bnd.Width)
	}
	if l[53] != bnd.Height {
		t.Fatalf("%sSetBounds: Viewport.Height\nhave %f\nwant %f", s, l[53], bnd.Height)
	}
	if l[54] != bnd.Znear {
		t.Fatalf("%sSetBounds: Viewport.Znear\nhave %f\nwant %f", s, l[54], bnd.Znear)
	}
	if l[55] != bnd.Zfar {
		t.Fatalf("%sSetBounds: Viewport.Zfar\nhave %f\nwant %f", s, l[55], bnd.Zfar)
	}
	if x := l.Bounds(); x != bnd {
		t.Fatalf("%sBounds:\nhave %v\nwant %v", s, x, bnd)
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
			t.Fatalf("%sSetUnused:\nhave false (0)\nwant true (1)", s)
		}
		if l.Unused() {
			t.Fatalf("%sUnused:\nhave true\nwant false", s)
		}
	case 1:
		if !unused {
			t.Fatalf("%sSetUnused:\nhave true (1)\nwant false (0)", s)
		}
		if !l.Unused() {
			t.Fatalf("%sUnused:\nhave false\nwant true", s)
		}
	default:
		t.Fatalf("%sSetUnused: bad value\n%d", s, x)
	}

	switch x, y := *(*int32)(unsafe.Pointer(&l[1])), l.Type(); {
	case x != typ:
		t.Fatalf("%sSetType:\nhave %d\nwant %d", s, x, typ)
	case y != typ:
		t.Fatalf("%sType:\nhave %d\nwant %d", s, y, typ)
	}

	switch x, y := l[2], l.Intensity(); {
	case x != intens:
		t.Fatalf("%sSetIntensity:\nhave %f\nwant %f", s, x, intens)
	case y != intens:
		t.Fatalf("%sIntensity:\nhave %f\nwant %f", s, y, intens)
	}

	switch x, y := l[3], l.Range(); {
	case x != rng:
		t.Fatalf("%sSetRange:\nhave %f\nwant %f", s, x, rng)
	case y != rng:
		t.Fatalf("%sRange:\nhave %f\nwant %f", s, y, rng)
	}

	checkSlicesT(l[4:7], color[:], t, s+"SetColor")
	if x := l.Color(); x != color {
		t.Fatalf("%sColor:\nhave %v\nwant %v", s, x, color)
	}

	switch x, y := l[7], l.AngScale(); {
	case x != scale:
		t.Fatalf("%sSetAngScale:\nhave %f\nwant %f", s, x, scale)
	case y != scale:
		t.Fatalf("%sAngScale:\nhave %f\nwant %f", s, y, scale)
	}

	checkSlicesT(l[8:11], pos[:], t, s+"SetPosition")
	if x := l.Position(); x != pos {
		t.Fatalf("%sPosition:\nhave %v\nwant %v", s, x, pos)
	}

	switch x, y := l[11], l.AngOffset(); {
	case x != off:
		t.Fatalf("%sSetAngOffset:\nhave %f\nwant %f", s, x, off)
	case y != off:
		t.Fatalf("%sAngOffset:\nhave %f\nwant %f", s, y, off)
	}

	checkSlicesT(l[12:15], dir[:], t, s+"SetDirection")
	if x := l.Direction(); x != dir {
		t.Fatalf("%sDirection:\nhave %v\nwant %v", s, x, dir)
	}
}

func TestShadowLayout(t *testing.T) {
	// [0:1]
	unused := true

	// [16:32]
	var shdw linear.M4
	shdw.Frustum(-1, 1, -1, 1, 1, 100)
	shdw.Mul(&linear.M4{{0.5}, {1: -0.5}, {2: 1}, {0.5, 0.5, 0, 1}}, &shdw)

	var l ShadowLayout
	l.SetUnused(unused)
	l.SetShadow(&shdw)

	s := "ShadowLayout."

	switch x := *(*int32)(unsafe.Pointer(&l[0])); x {
	case 0:
		if unused {
			t.Fatalf("%sSetUnused:\nhave false (0)\nwant true (1)", s)
		}
		if l.Unused() {
			t.Fatalf("%sUnused:\nhave true\nwant false", s)
		}
	case 1:
		if !unused {
			t.Fatalf("%sSetUnused:\nhave true (1)\nwant false (0)", s)
		}
		if !l.Unused() {
			t.Fatalf("%sUnused:\nhave false\nwant true", s)
		}
	default:
		t.Fatalf("%sSetUnused: bad value\n%d", s, x)
	}

	checkSlicesT(l[16:32], unsafe.Slice((*float32)(unsafe.Pointer(&shdw)), 16), t, s+"SetShadow")
	if x := l.Shadow(); x != shdw {
		t.Fatalf("%sShadow:\nhave %v\nwant %v", s, x, shdw)
	}
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
	if x := l.World(); x != wld {
		t.Fatalf("%sWorld:\nhave %v\nwant %v", s, x, wld)
	}

	checkSlicesT(l[16:28], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, s+"SetNormal")
	if x := l.Normal(); x != norm {
		t.Fatalf("%sNormal:\nhave %v\nwant %v", s, x, norm)
	}

	switch x, y := *(*uint32)(unsafe.Pointer(&l[28])), l.ID(); {
	case x != id:
		t.Fatalf("%sSetID:\nhave %d\nwant %d", s, x, id)
	case y != id:
		t.Fatalf("%sID:\nhave %d\nwant %d", s, y, id)
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
	if x := l.ColorFactor(); x != color {
		t.Fatalf("%sColorFactor:\nhave %v\nwant %v", s, x, color)
	}

	if m, r := l[4], l[5]; m != metal || r != rough {
		t.Fatalf("%sSetMetalRough:\nhave %f,%f\nwant %f,%f", s, m, r, metal, rough)
	}
	if m, r := l.MetalRough(); m != metal || r != rough {
		t.Fatalf("%sMetalRough:\nhave %f,%f\nwant %f,%f", s, m, r, metal, rough)
	}

	switch x, y := l[6], l.NormScale(); {
	case x != scale:
		t.Fatalf("%sSetNormScale:\nhave %f\nwant %f", s, x, scale)
	case y != scale:
		t.Fatalf("%sNormScale:\nhave %f\nwant %f", s, y, scale)
	}

	switch x, y := l[7], l.OccStrength(); {
	case x != strength:
		t.Fatalf("%sSetOccStrength:\nhave %f\nwant %f", s, x, strength)
	case y != strength:
		t.Fatalf("%sOccStrength:\nhave %f\nwant %f", s, y, strength)
	}

	checkSlicesT(l[8:11], emissive[:], t, s+"SetEmisFactor")
	if x := l.EmisFactor(); x != emissive {
		t.Fatalf("%sEmisFactor:\nhave %v\nwant %v", s, x, emissive)
	}

	switch x, y := l[11], l.AlphaCutoff(); {
	case x != cutoff:
		t.Fatalf("%sSetAlphaCutoff:\nhave %f\nwant %f", s, x, cutoff)
	case y != cutoff:
		t.Fatalf("%sAlphaCutoff:\nhave %f\nwant %f", s, y, cutoff)
	}

	switch x, y := *(*uint32)(unsafe.Pointer(&l[12])), l.Flags(); {
	case x != flags:
		t.Fatalf("%sSetFlags:\nhave 0x%x\nwant 0x%x", s, x, flags)
	case y != flags:
		t.Fatalf("%sFlags:\nhave 0x%x\nwant 0x%x", s, y, flags)
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
