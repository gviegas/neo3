// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/linear"
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

	checkSlicesT(l[0:16], unsafe.Slice((*float32)(unsafe.Pointer(&vp)), 16), t, s+"SetVP")
	checkSlicesT(l[16:32], unsafe.Slice((*float32)(unsafe.Pointer(&v)), 16), t, s+"SetV")
	checkSlicesT(l[32:48], unsafe.Slice((*float32)(unsafe.Pointer(&p)), 16), t, s+"SetP")
	if x := float32(tm.Seconds()); l[48] != x {
		t.Fatalf("%s.SetTime:\nhave %f\nwant%f", s, l[48], x)
	}
	if l[49] != rnd {
		t.Fatalf("%s.SetRand:\nhave %f\nwant%f", s, l[49], rnd)
	}
	if l[50] != bnd.X {
		t.Fatalf("%s.SetBounds: Viewport.X\nhave %f\nwant%f", s, l[50], bnd.X)
	}
	if l[51] != bnd.Y {
		t.Fatalf("%s.SetBounds: Viewport.Y\nhave %f\nwant%f", s, l[51], bnd.Y)
	}
	if l[52] != bnd.Width {
		t.Fatalf("%s.SetBounds: Viewport.Width\nhave %f\nwant%f", s, l[52], bnd.Width)
	}
	if l[53] != bnd.Height {
		t.Fatalf("%s.SetBounds: Viewport.Height\nhave %f\nwant%f", s, l[53], bnd.Height)
	}
	if l[54] != bnd.Znear {
		t.Fatalf("%s.SetBounds: Viewport.Znear\nhave %f\nwant%f", s, l[54], bnd.Znear)
	}
	if l[55] != bnd.Zfar {
		t.Fatalf("%s.SetBounds: Viewport.Zfar\nhave %f\nwant%f", s, l[55], bnd.Zfar)
	}
}
