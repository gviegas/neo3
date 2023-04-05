// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
	"github.com/gviegas/scene/linear"
)

// check checks that tb is valid.
func (tb *Table) check(globalN, drawableN, materialN, jointN int, t *testing.T) {
	if tb == nil {
		t.Fatal("Table is nil (NewTable likely failed)")
	}
	var csz int
	for _, x := range [4]struct {
		s    string
		i, n int
		spn  uintptr
	}{
		{"globalHeap", globalHeap, globalN, frameSpan + lightSpan + shadowSpan},
		{"drawableHeap", drawableHeap, drawableN, drawableSpan},
		{"materialHeap", materialHeap, materialN, materialSpan},
		{"jointHeap", jointHeap, jointN, jointSpan},
	} {
		if n := tb.dt.Heap(x.i).Len(); n != x.n {
			t.Fatalf("Table.dt.Heap(%s).Len:\nhave %d\nwant %d", x.s, n, x.n)
		} else if tb.dcpy[x.i] != x.n {
			t.Fatalf("Table.dcpy[%s]:\nhave %d\nwant %d", x.s, tb.dcpy[x.i], x.n)
		} else {
			csz += n * int(x.spn)
		}
	}
	csz *= blockSize
	if x := tb.ConstSize(); x != csz {
		t.Fatalf("Table.ConstSize:\nhave %d\nwant %d", x, csz)
	} else if x%blockSize != 0 {
		t.Fatal("Table.ConstSize: misaligned size")
	} else if tb.cbuf != nil && tb.cbuf.Cap()-tb.coff[globalHeap] < int64(x) {
		t.Fatal("Table.cbuf/coff: range out of bounds")
	}
}

func TestNewTable(t *testing.T) {
	for _, x := range [...]struct{ ng, nd, nm, nj int }{
		{ng: 1},
		{ng: 2},
		{ng: 3},
		{ng: 1, nd: 1},
		{ng: 1, nd: 2},
		{ng: 1, nd: 3},
		{ng: 1, nd: 1, nm: 1},
		{ng: 1, nd: 1, nm: 2},
		{ng: 1, nd: 1, nm: 3},
		{ng: 1, nd: 1, nm: 1, nj: 1},
		{ng: 1, nd: 1, nm: 1, nj: 2},
		{ng: 1, nd: 1, nm: 1, nj: 3},
		{ng: 0, nd: 2, nm: 2, nj: 2},
		{ng: 0, nd: 0, nm: 2, nj: 2},
		{ng: 0, nd: 0, nm: 0, nj: 2},
		{ng: 3, nd: 0, nm: 2, nj: 1},
		{ng: 2, nd: 0, nm: 0, nj: 3},
		{ng: 2, nd: 1, nm: 0, nj: 3},
		{ng: 1, nd: 16, nm: 16, nj: 16},
		{ng: 2, nd: 64, nm: 64, nj: 64},
		{ng: 3, nd: 256, nm: 256, nj: 256},
		{ng: 4, nd: 255, nm: 254, nj: 253},
		{ng: 5, nd: 128, nm: 0, nj: 128},
		{ng: 6, nd: 0, nm: 127, nj: 0},
		{ng: 7, nd: 150, nm: 0, nj: 0},
		{ng: 8, nd: 31, nm: 40, nj: 0},
		{ng: 9, nd: 1000, nm: 1000, nj: 1000},
		{ng: 3 * 1, nd: 3 * 1024, nm: 3 * 1024, nj: 3 * 1024},
	} {
		tb, _ := NewTable(x.ng, x.nd, x.nm, x.nj)
		tb.check(x.ng, x.nd, x.nm, x.nj, t)
		tb.Free()
	}
}

func TestSetConstBuf(t *testing.T) {
	ng, nd, nm, nj := 1, 1, 1, 1
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)

	buf, err := ctxt.GPU().NewBuffer(int64(tb.ConstSize()+blockSize), true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}

	if buf, off := tb.SetConstBuf(nil, 0); buf != nil || off != 0 {
		t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant <nil>, 0", buf, off)
	}
	if tb.coff[globalHeap] != 0 {
		t.Fatalf("Table.coff[globalHeap]:\nhave %d\nwant 0", tb.coff[globalHeap])
	}
	if buf, off := tb.SetConstBuf(buf, blockSize); buf != nil || off != 0 {
		t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant <nil>, 0", buf, off)
	}
	if tb.coff[globalHeap] != blockSize {
		t.Fatalf("Table.coff[globalHeap]:\nhave %d\nwant %d", tb.coff[globalHeap], blockSize)
	}
	if buf_, off := tb.SetConstBuf(nil, blockSize*2); buf_ != buf || off != blockSize {
		t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant %v, %d", buf_, off, buf, blockSize)
	}
	if tb.coff[globalHeap] != 0 {
		t.Fatalf("Table.coff[globalHeap]:\nhave %d\nwant 0", tb.coff[globalHeap])
	}

	tb.Free()
	buf.Destroy()

	for _, x := range [...]struct{ ng, nd, nm, nj int }{
		{ng: 1},
		{ng: 1, nd: 1},
		{ng: 1, nd: 1, nm: 1},
		{ng: 1, nd: 1, nm: 1, nj: 1},
		{ng: 2, nd: 2, nm: 0, nj: 2},
		{ng: 3, nd: 3, nm: 3, nj: 0},
		{ng: 1, nd: 16, nm: 15, nj: 14},
		{ng: 2, nd: 62, nm: 63, nj: 64},
		{ng: 3, nd: 384, nm: 384, nj: 384},
	} {
		tb, _ := NewTable(x.ng, x.nd, x.nm, x.nj)
		tb.check(x.ng, x.nd, x.nm, x.nj, t)

		sz := int64(tb.ConstSize() * 4)
		buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
		if err != nil {
			t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
		}

		var goff, doff, moff, joff int64
		doff = int64(frameSpan+lightSpan+shadowSpan) * blockSize * int64(x.ng)
		moff = doff + int64(drawableSpan)*blockSize*int64(x.nd)
		joff = moff + int64(materialSpan)*blockSize*int64(x.nm)

		wbuf, woff := driver.Buffer(nil), int64(0)
		for _, x := range [3]int64{0, sz / 2, sz - int64(tb.ConstSize())} {
			if hbuf, hoff := tb.SetConstBuf(buf, x); wbuf != hbuf || woff != hoff {
				t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant %v, %d", hbuf, hoff, wbuf, woff)
			}
			if x := goff + x; tb.coff[globalHeap] != x {
				t.Fatalf("Table.coff[globalHeap]:\nhave %d\nwant %d", tb.coff[globalHeap], x)
			}
			if x := doff + x; tb.coff[drawableHeap] != x {
				t.Fatalf("Table.coff[drawableHeap]:\nhave %d\nwant %d", tb.coff[drawableHeap], x)
			}
			if x := moff + x; tb.coff[materialHeap] != x {
				t.Fatalf("Table.coff[materialHeap]:\nhave %d\nwant %d", tb.coff[materialHeap], x)
			}
			if x := joff + x; tb.coff[jointHeap] != x {
				t.Fatalf("Table.coff[jointHeap]:\nhave %d\nwant %d", tb.coff[jointHeap], x)
			}
			wbuf = buf
			woff = x
		}

		tb.Free()
		buf.Destroy()
	}
}

func TestConstWrite(t *testing.T) {
	dummyData := func(ng, nd, nm, nj int) (
		f []FrameLayout,
		l [][MaxLight]LightLayout,
		s [][MaxShadow]ShadowLayout,
		d []DrawableLayout,
		m []MaterialLayout,
		j [][MaxJoint]JointLayout,
	) {
		var x float32
		const y = 0.0001220703125

		f = make([]FrameLayout, ng)
		l = make([][MaxLight]LightLayout, ng)
		s = make([][MaxShadow]ShadowLayout, ng)
		d = make([]DrawableLayout, nd)
		m = make([]MaterialLayout, nm)
		j = make([][MaxJoint]JointLayout, nj)

		for i := range f {
			for j := range f[i] {
				x += y
				f[i][j] = x
			}
		}
		for i := range l {
			for j := range l[i] {
				for k := range l[i][j] {
					x += y
					l[i][j][k] = x
				}
			}
		}
		for i := range s {
			for j := range s[i] {
				for k := range s[i][j] {
					x += y
					s[i][j][k] = x
				}
			}
		}
		for i := range d {
			for j := range d[i] {
				x += y
				d[i][j] = x
			}
		}
		for i := range m {
			for j := range m[i] {
				x += y
				m[i][j] = x
			}
		}
		for i := range j {
			for j_ := range j[i] {
				for k := range j[i][j_] {
					x += y
					j[i][j_][k] = x
				}
			}
		}

		return
	}
	f, l, s, d, m, j := dummyData(3, 15, 15, 15)

	for _, x := range [...]struct{ ng, nd, nm, nj int }{
		{ng: 1},
		{nd: 1},
		{nm: 1},
		{nj: 1},
		{ng: 1, nd: 1},
		{ng: 1, nm: 1},
		{ng: 1, nj: 1},
		{nd: 1, nm: 1},
		{nd: 1, nj: 1},
		{nm: 1, nj: 1},
		{ng: 1, nd: 1, nm: 1},
		{ng: 1, nd: 1, nj: 1},
		{ng: 1, nm: 1, nj: 1},
		{nd: 1, nm: 1, nj: 1},
		{ng: 1, nd: 1, nm: 1, nj: 1},
		{ng: 2},
		{nd: 2},
		{nm: 2},
		{nj: 2},
		{ng: 2, nd: 2},
		{ng: 2, nm: 2},
		{ng: 2, nj: 2},
		{nd: 2, nm: 2},
		{nd: 2, nj: 2},
		{nm: 2, nj: 2},
		{ng: 2, nd: 2, nm: 2},
		{ng: 2, nd: 2, nj: 2},
		{ng: 2, nm: 2, nj: 2},
		{nd: 2, nm: 2, nj: 2},
		{ng: 2, nd: 2, nm: 2, nj: 2},
		{ng: 3, nd: 2},
		{ng: 3, nm: 2},
		{ng: 3, nj: 2},
		{nd: 3, nm: 2},
		{nd: 3, nj: 2},
		{nm: 3, nj: 2},
		{ng: 3, nd: 2, nm: 1},
		{ng: 3, nd: 1, nj: 2},
		{ng: 3, nm: 2, nj: 1},
		{nd: 3, nm: 1, nj: 2},
		{ng: 3, nd: 15, nm: 15, nj: 15},
	} {
		tb, _ := NewTable(x.ng, x.nd, x.nm, x.nj)
		tb.check(x.ng, x.nd, x.nm, x.nj, t)

		sz := int64(tb.ConstSize() * 4)
		buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
		if err != nil {
			t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
		}

		rng := [3]int64{0, sz / 2, sz - int64(tb.ConstSize())}

		for _, y := range rng {
			tb.SetConstBuf(buf, y)

			for i := 0; i < x.ng; i++ {
				*tb.Frame(i) = f[i]
				*tb.Light(i) = l[i]
				*tb.Shadow(i) = s[i]
			}
			for i := 0; i < x.nd; i++ {
				*tb.Drawable(i) = d[i]
			}
			for i := 0; i < x.nm; i++ {
				*tb.Material(i) = m[i]
			}
			for i := 0; i < x.nj; i++ {
				*tb.Joint(i) = j[i]
			}

			for i := 0; i < x.ng; i++ {
				checkSlicesT(tb.Frame(i)[:], f[i][:], t, fmt.Sprintf("Table.Frame(%d)", i))
				for j := 0; j < int(MaxLight); j++ {
					checkSlicesT(tb.Light(i)[j][:], l[i][j][:], t, fmt.Sprintf("Table.Light(%d)[%d]", i, j))
				}
				for j := 0; j < int(MaxShadow); j++ {
					checkSlicesT(tb.Shadow(i)[j][:], s[i][j][:], t, fmt.Sprintf("Table.Shadow(%d)[%d]", i, j))
				}
			}
			for i := 0; i < x.nd; i++ {
				checkSlicesT(tb.Drawable(i)[:], d[i][:], t, fmt.Sprintf("Table.Drawable(%d)", i))
			}
			for i := 0; i < x.nm; i++ {
				checkSlicesT(tb.Material(i)[:], m[i][:], t, fmt.Sprintf("Table.Material(%d)", i))
			}
			for i := 0; i < x.nj; i++ {
				for j_ := 0; j_ < int(MaxJoint); j_++ {
					checkSlicesT(tb.Joint(i)[j_][:], j[i][j_][:], t, fmt.Sprintf("Table.Joint(%d)[%d]", i, j_))
				}
			}
		}

		tb.Free()
		buf.Destroy()
	}
}

func TestGlobalWrite(t *testing.T) {
	ng, nd, nm, nj := 1, 0, 0, 0
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var v, p linear.M4
	v.Translate(-1, -2, -3)
	p.Ortho(-1, 1, -1, 1)
	rnd := float32(0.25)
	vport := driver.Viewport{0, 0, 1920, 1080, 0.01, 1000.0}

	tb.Frame(0).SetV(&v)
	tb.Frame(0).SetP(&p)
	tb.Frame(0).SetRand(rnd)
	tb.Frame(0).SetBounds(&vport)

	s := "Table.Frame(0)[%d:]"
	checkSlicesT(tb.Frame(0)[16:], unsafe.Slice((*float32)(unsafe.Pointer(&v)), 16), t, fmt.Sprintf(s, 16))
	checkSlicesT(tb.Frame(0)[32:], unsafe.Slice((*float32)(unsafe.Pointer(&p)), 16), t, fmt.Sprintf(s, 32))
	checkSlicesT(tb.Frame(0)[49:], []float32{rnd}, t, fmt.Sprintf(s, 49))
	checkSlicesT(tb.Frame(0)[50:], unsafe.Slice((*float32)(unsafe.Pointer(&vport)), 6), t, fmt.Sprintf(s, 50))

	rng := float32(15)
	color := linear.V3{0.2, 0.5, 0.1}
	dir := linear.V3{0.7071, 0, -0.7071}

	tb.Light(0)[0].SetRange(rng)
	tb.Light(0)[0].SetColor(&color)
	tb.Light(0)[0].SetDirection(&dir)
	tb.Light(0)[1].SetRange(-rng)

	s = "Table.Light(0)[0][%d:]"
	checkSlicesT(tb.Light(0)[0][3:], []float32{rng}, t, fmt.Sprintf(s, 3))
	checkSlicesT(tb.Light(0)[0][4:], color[:], t, fmt.Sprintf(s, 4))
	checkSlicesT(tb.Light(0)[0][12:], dir[:], t, fmt.Sprintf(s, 12))
	s = "Table.Light(0)[1][%d:]"
	checkSlicesT(tb.Light(0)[1][3:], []float32{-rng}, t, fmt.Sprintf(s, 3))

	shdw := linear.M4{{0.5}, {1: 0.5}, {2: 0.5}, {0.5, 0.5, 0.5, 1}}
	unused, unused_ := true, int32(1)

	tb.Shadow(0)[0].SetUnused(unused)
	tb.Shadow(0)[0].SetShadow(&shdw)

	s = "Table.Shadow(0)[0][%d:]"
	checkSlicesT(tb.Shadow(0)[0][:], []float32{*(*float32)(unsafe.Pointer(&unused_))}, t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Shadow(0)[0][16:], unsafe.Slice((*float32)(unsafe.Pointer(&shdw)), 16), t, fmt.Sprintf(s, 16))
}

func TestGlobalWriteN(t *testing.T) {
	ng, nd, nm, nj := 3, 10, 10, 10
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var v, p linear.M4
	v.Translate(-1, -2, -3)
	p.Ortho(-1, 1, -1, 1)
	rnd := float32(0.25)
	vport := driver.Viewport{0, 0, 1920, 1080, 0.01, 1000.0}

	tb.Frame(0).SetV(&v)
	tb.Frame(0).SetP(&p)
	tb.Frame(0).SetRand(rnd)
	tb.Frame(1).SetBounds(&vport)
	tb.Frame(2).SetV(&v)
	tb.Frame(0).SetP(&p)
	tb.Frame(2).SetRand(rnd + 0.1)
	tb.Frame(1).SetBounds(&vport)

	s := "Table.Frame(0)[%d:]"
	checkSlicesT(tb.Frame(0)[16:], unsafe.Slice((*float32)(unsafe.Pointer(&v)), 16), t, fmt.Sprintf(s, 16))
	checkSlicesT(tb.Frame(0)[32:], unsafe.Slice((*float32)(unsafe.Pointer(&p)), 16), t, fmt.Sprintf(s, 32))
	checkSlicesT(tb.Frame(0)[49:], []float32{rnd}, t, fmt.Sprintf(s, 49))
	s = "Table.Frame(1)[%d:]"
	checkSlicesT(tb.Frame(1)[50:], unsafe.Slice((*float32)(unsafe.Pointer(&vport)), 6), t, fmt.Sprintf(s, 50))
	s = "Table.Frame(2)[%d:]"
	checkSlicesT(tb.Frame(2)[16:], unsafe.Slice((*float32)(unsafe.Pointer(&v)), 16), t, fmt.Sprintf(s, 16))
	checkSlicesT(tb.Frame(2)[49:], []float32{rnd + 0.1}, t, fmt.Sprintf(s, 49))

	rng := float32(15)
	color := linear.V3{0.2, 0.5, 0.1}
	dir := linear.V3{0.7071, 0, -0.7071}

	tb.Light(0)[0].SetRange(rng)
	tb.Light(0)[0].SetColor(&color)
	tb.Light(0)[0].SetDirection(&dir)
	tb.Light(0)[1].SetRange(-rng)
	tb.Light(1)[8].SetRange(rng)
	tb.Light(1)[8].SetColor(&color)
	tb.Light(2)[1].SetDirection(&dir)
	tb.Light(2)[MaxLight-1].SetRange(-rng)

	s = "Table.Light(0)[0][%d:]"
	checkSlicesT(tb.Light(0)[0][3:], []float32{rng}, t, fmt.Sprintf(s, 3))
	checkSlicesT(tb.Light(0)[0][4:], color[:], t, fmt.Sprintf(s, 4))
	checkSlicesT(tb.Light(0)[0][12:], dir[:], t, fmt.Sprintf(s, 12))
	s = "Table.Light(0)[1][%d:]"
	checkSlicesT(tb.Light(0)[1][3:], []float32{-rng}, t, fmt.Sprintf(s, 3))
	s = "Table.Light(1)[8][%d:]"
	checkSlicesT(tb.Light(1)[8][3:], []float32{rng}, t, fmt.Sprintf(s, 3))
	checkSlicesT(tb.Light(1)[8][4:], color[:], t, fmt.Sprintf(s, 4))
	s = "Table.Light(2)[1][%d:]"
	checkSlicesT(tb.Light(2)[1][12:], dir[:], t, fmt.Sprintf(s, 12))
	s = fmt.Sprintf("Table.Light(2)[%d]%s", MaxLight-1, "[%d:]")
	checkSlicesT(tb.Light(2)[MaxLight-1][3:], []float32{-rng}, t, fmt.Sprintf(s, 3))

	shdw := linear.M4{{0.5}, {1: 0.5}, {2: 0.5}, {0.5, 0.5, 0.5, 1}}
	unused, unused_ := true, int32(1)

	tb.Shadow(2)[0].SetUnused(unused)
	tb.Shadow(0)[0].SetShadow(&shdw)

	s = "Table.Shadow(2)[0][%d:]"
	checkSlicesT(tb.Shadow(2)[0][:], []float32{*(*float32)(unsafe.Pointer(&unused_))}, t, fmt.Sprintf(s, 0))
	s = "Table.Shadow(0)[0][%d:]"
	checkSlicesT(tb.Shadow(0)[0][16:], unsafe.Slice((*float32)(unsafe.Pointer(&shdw)), 16), t, fmt.Sprintf(s, 16))
}

func TestDrawableWrite(t *testing.T) {
	ng, nd, nm, nj := 1, 1, 0, 0
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var wld linear.M4
	wld.Translate(100, 50, -25)
	id := uint32(0x1d)

	tb.Drawable(0).SetWorld(&wld)
	tb.Drawable(0).SetID(id)

	s := "Table.Drawable(0)[%d:]"
	checkSlicesT(tb.Drawable(0)[:], unsafe.Slice((*float32)(unsafe.Pointer(&wld)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Drawable(0)[48:], unsafe.Slice((*float32)(unsafe.Pointer(&id)), 1), t, fmt.Sprintf(s, 48))
}

func TestDrawableWriteN(t *testing.T) {
	ng, nd, nm, nj := 3, 10, 10, 10
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var wld, norm linear.M4
	wld.Translate(100, 50, -25)
	norm.Invert(&wld)
	norm.Transpose(&norm)
	id := uint32(0x1d)
	id1 := id << 8

	tb.Drawable(0).SetWorld(&wld)
	tb.Drawable(0).SetID(id)
	tb.Drawable(3).SetWorld(&wld)
	tb.Drawable(3).SetNormal(&norm)
	tb.Drawable(9).SetID(id1)

	s := "Table.Drawable(0)[%d:]"
	checkSlicesT(tb.Drawable(0)[:], unsafe.Slice((*float32)(unsafe.Pointer(&wld)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Drawable(0)[48:], unsafe.Slice((*float32)(unsafe.Pointer(&id)), 1), t, fmt.Sprintf(s, 48))
	s = "Table.Drawable(3)[%d:]"
	checkSlicesT(tb.Drawable(3)[:], unsafe.Slice((*float32)(unsafe.Pointer(&wld)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Drawable(3)[16:], unsafe.Slice((*float32)(unsafe.Pointer(&norm)), 16), t, fmt.Sprintf(s, 16))
	s = "Table.Drawable(9)[%d:]"
	checkSlicesT(tb.Drawable(9)[48:], unsafe.Slice((*float32)(unsafe.Pointer(&id1)), 1), t, fmt.Sprintf(s, 48))
}

func TestMaterialWrite(t *testing.T) {
	ng, nd, nm, nj := 1, 1, 1, 0
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	color := linear.V4{0.1, 0.2, 0.3, 1.0}
	metal := float32(1.0)
	rough := float32(0.5)
	cutoff := float32(0.3333)

	tb.Material(0).SetColorFactor(&color)
	tb.Material(0).SetMetalRough(metal, rough)
	tb.Material(0).SetAlphaCutoff(cutoff)

	s := "Table.Material(0)[%d:]"
	checkSlicesT(tb.Material(0)[:], color[:], t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Material(0)[4:], []float32{metal}, t, fmt.Sprintf(s, 4))
	checkSlicesT(tb.Material(0)[5:], []float32{rough}, t, fmt.Sprintf(s, 5))
	checkSlicesT(tb.Material(0)[11:], []float32{cutoff}, t, fmt.Sprintf(s, 11))
}

func TestMaterialWriteN(t *testing.T) {
	ng, nd, nm, nj := 3, 10, 10, 10
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	color := linear.V4{0.1, 0.2, 0.3, 1.0}
	metal := float32(0.925)
	rough := float32(0.575)
	cutoff := float32(0.3333)

	tb.Material(0).SetColorFactor(&color)
	tb.Material(2).SetMetalRough(metal, rough)
	tb.Material(9).SetMetalRough(1.0-metal, 1.0-rough)
	tb.Material(0).SetAlphaCutoff(cutoff)
	tb.Material(9).SetAlphaCutoff(1.0 - cutoff)

	s := "Table.Material(0)[%d:]"
	checkSlicesT(tb.Material(0)[:], color[:], t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Material(0)[11:], []float32{cutoff}, t, fmt.Sprintf(s, 11))
	s = "Table.Material(2)[%d:]"
	checkSlicesT(tb.Material(2)[4:], []float32{metal}, t, fmt.Sprintf(s, 4))
	checkSlicesT(tb.Material(2)[5:], []float32{rough}, t, fmt.Sprintf(s, 5))
	s = "Table.Material(9)[%d:]"
	checkSlicesT(tb.Material(9)[4:], []float32{1.0 - metal}, t, fmt.Sprintf(s, 4))
	checkSlicesT(tb.Material(9)[5:], []float32{1.0 - rough}, t, fmt.Sprintf(s, 5))
	checkSlicesT(tb.Material(9)[11:], []float32{1.0 - cutoff}, t, fmt.Sprintf(s, 11))
}

func TestJointWrite(t *testing.T) {
	ng, nd, nm, nj := 1, 1, 1, 1
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var jnt, norm linear.M4
	jnt.Translate(-0.5, -0.25, 0.125)
	norm.Invert(&jnt)
	norm.Transpose(&norm)

	tb.Joint(0)[0].SetJoint(&jnt)
	tb.Joint(0)[0].SetNormal(&norm)
	tb.Joint(0)[MaxJoint-1].SetJoint(&jnt)

	s := "Table.Joint(0)[0][%d:]"
	checkSlicesT(tb.Joint(0)[0][:], unsafe.Slice((*float32)(unsafe.Pointer(&jnt)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Joint(0)[0][16:], unsafe.Slice((*float32)(unsafe.Pointer(&norm)), 16), t, fmt.Sprintf(s, 16))
	s = fmt.Sprintf("Table.Joint(0)[%d]%s", MaxJoint-1, "[%d:]")
	checkSlicesT(tb.Joint(0)[MaxJoint-1][:], unsafe.Slice((*float32)(unsafe.Pointer(&jnt)), 16), t, fmt.Sprintf(s, 0))
}

func TestJointWriteN(t *testing.T) {
	ng, nd, nm, nj := 3, 10, 10, 10
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var jnt, norm linear.M4
	jnt.Translate(-0.5, -0.25, 0.125)
	norm.Invert(&jnt)
	norm.Transpose(&norm)

	tb.Joint(0)[0].SetJoint(&jnt)
	tb.Joint(0)[0].SetNormal(&norm)
	tb.Joint(0)[MaxJoint-1].SetJoint(&jnt)
	tb.Joint(5)[MaxJoint/2].SetJoint(&jnt)
	tb.Joint(9)[1].SetNormal(&norm)
	tb.Joint(9)[MaxJoint-1].SetJoint(&jnt)
	tb.Joint(9)[MaxJoint-1].SetNormal(&norm)

	s := "Table.Joint(0)[0][%d:]"
	checkSlicesT(tb.Joint(0)[0][:], unsafe.Slice((*float32)(unsafe.Pointer(&jnt)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Joint(0)[0][16:], unsafe.Slice((*float32)(unsafe.Pointer(&norm)), 16), t, fmt.Sprintf(s, 16))
	s = fmt.Sprintf("Table.Joint(0)[%d]%s", MaxJoint-1, "[%d:]")
	checkSlicesT(tb.Joint(0)[MaxJoint-1][:], unsafe.Slice((*float32)(unsafe.Pointer(&jnt)), 16), t, fmt.Sprintf(s, 0))

	s = fmt.Sprintf("Table.Joint(5)[%d]%s", MaxJoint/2, "[%d:]")
	checkSlicesT(tb.Joint(5)[MaxJoint/2][:], unsafe.Slice((*float32)(unsafe.Pointer(&jnt)), 16), t, fmt.Sprintf(s, 0))

	s = "Table.Joint(9)[1][%d:]"
	checkSlicesT(tb.Joint(9)[1][16:], unsafe.Slice((*float32)(unsafe.Pointer(&norm)), 16), t, fmt.Sprintf(s, 16))
	s = fmt.Sprintf("Table.Joint(9)[%d]%s", MaxJoint-1, "[%d:]")
	checkSlicesT(tb.Joint(9)[MaxJoint-1][:], unsafe.Slice((*float32)(unsafe.Pointer(&jnt)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Joint(9)[MaxJoint-1][16:], unsafe.Slice((*float32)(unsafe.Pointer(&norm)), 16), t, fmt.Sprintf(s, 16))
}

func TestSetCBFail(t *testing.T) {
	ng, nd, nm, nj := 2, 8, 6, 8
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	buf, err := ctxt.GPU().NewBuffer(int64(tb.ConstSize()), true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	defer func() {
		s := "Table.SetConstBuf:\nhave %#v\nwant %#v"
		want := "misaligned constant buffer offset"
		if x := recover(); x != want {
			t.Fatalf(s, x, want)
		}
		defer func() {
			want := "constant buffer range out of bounds"
			if x := recover(); x != want {
				t.Fatalf(s, x, want)
			}
		}()
		tb.SetConstBuf(buf, int64(tb.ConstSize()))
	}()
	tb.SetConstBuf(buf, 1)
}

func TestSetTSFail(t *testing.T) {
	ng, nd, nm, nj := 2, 8, 6, 8
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	img, err := ctxt.GPU().NewImage(driver.RGBA8un, driver.Dim3D{256, 256, 0}, 1, 1, 1, driver.UShaderSample)
	if err != nil {
		t.Fatalf("driver.GPU.NewImage failed:\n%#v", err)
	}
	defer img.Destroy()
	iv, err := img.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		t.Fatalf("driver.Image.NewView failed:\n%#v", err)
	}
	defer iv.Destroy()
	splr, err := ctxt.GPU().NewSampler(&driver.Sampling{MaxAniso: 1, MaxLOD: 1})
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer splr.Destroy()

	for _, c := range [...]struct {
		s     string
		f     func(*Table, int, driver.ImageView, driver.Sampler)
		wcpy  string
		wtex  string
		wsplr string
	}{
		{
			"ShadowMap",
			(*Table).SetShadowMap,
			"shadow map descriptor out of bounds",
			"nil shadow map texture",
			"nil shadow map sampler",
		},
		{
			"BaseColor",
			(*Table).SetBaseColor,
			"base color descriptor out of bounds",
			"nil base color texture",
			"nil base color sampler",
		},
		{
			"MetalRough",
			(*Table).SetMetalRough,
			"metallic-roughness descriptor out of bounds",
			"nil metallic-roughness texture",
			"nil metallic-roughness sampler",
		},
		{
			"NormalMap",
			(*Table).SetNormalMap,
			"normal map descriptor out of bounds",
			"nil normal map texture",
			"nil normal map sampler",
		},
		{
			"OcclusionMap",
			(*Table).SetOcclusionMap,
			"occlusion map descriptor out of bounds",
			"nil occlusion map texture",
			"nil occlusion map sampler",
		},
		{
			"EmissiveMap",
			(*Table).SetEmissiveMap,
			"emissive map descriptor out of bounds",
			"nil emissive map texture",
			"nil emissive map sampler",
		},
	} {
		t.Run(c.s, func(t *testing.T) {
			s := "Table.Set" + c.s + ":\nhave %#v\nwant %#v"
			t.Run("cpy", func(t *testing.T) {
				defer func() {
					if x := recover(); x != c.wcpy {
						t.Fatalf(s, x, c.wcpy)
					}
					defer func() {
						if x := recover(); x != c.wcpy {
							t.Fatalf(s, x, c.wcpy)
						}
					}()
					c.f(tb, -1, iv, splr)
				}()
				c.f(tb, 6, iv, splr)
			})
			t.Run("tex", func(t *testing.T) {
				defer func() {
					if x := recover(); x != c.wtex {
						t.Fatalf(s, x, c.wtex)
					}
				}()
				c.f(tb, 0, nil, splr)
			})
			t.Run("splr", func(t *testing.T) {
				defer func() {
					if x := recover(); x != c.wsplr {
						t.Fatalf(s, x, c.wsplr)
					}
				}()
				c.f(tb, 0, iv, nil)
			})
		})
	}
}

func TestConstFail(t *testing.T) {
	ng, nd, nm, nj := 2, 8, 6, 8
	tb, _ := NewTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	buf, err := ctxt.GPU().NewBuffer(int64(tb.ConstSize()), true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	for _, c := range [...]struct {
		s    string
		f    func(int)
		want string
	}{
		{
			"Frame",
			func(cpy int) { tb.Frame(cpy) },
			"frame descriptor out of bounds",
		},
		{
			"Light",
			func(cpy int) { tb.Light(cpy) },
			"light descriptor out of bounds",
		},
		{
			"Shadow",
			func(cpy int) { tb.Shadow(cpy) },
			"shadow descriptor out of bounds",
		},
		{
			"Drawable",
			func(cpy int) { tb.Drawable(cpy) },
			"drawable descriptor out of bounds",
		},
		{
			"Material",
			func(cpy int) { tb.Material(cpy) },
			"material descriptor out of bounds",
		},
		{
			"Joint",
			func(cpy int) { tb.Joint(cpy) },
			"joint descriptor out of bounds",
		},
	} {
		t.Run(c.s, func(t *testing.T) {
			s := "Table." + c.s + ":\nhave %#v\nwant %#v"
			defer func() {
				if x := recover(); x != c.want {
					t.Fatalf(s, x, c.want)
				}
				defer func() {
					if x := recover(); x != c.want {
						t.Fatalf(s, x, c.want)
					}
				}()
				c.f(-2)
			}()
			c.f(8)
		})
	}
}
