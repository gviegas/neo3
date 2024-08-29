// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
	"fmt"
	"testing"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/linear"
)

// check checks that tb is valid.
func (tb *DrawTable) check(globalN, drawableN, materialN, jointN int, t *testing.T) {
	if tb == nil {
		t.Fatal("DrawTable is nil (NewDrawTable likely failed)")
	}
	var csz int
	for _, x := range [4]struct {
		s    string
		i, n int
		spn  uintptr
	}{
		{"GlobalHeap", GlobalHeap, globalN, frameSpan + lightSpan + shadowSpan},
		{"DrawableHeap", DrawableHeap, drawableN, drawableSpan},
		{"MaterialHeap", MaterialHeap, materialN, materialSpan},
		{"JointHeap", JointHeap, jointN, jointSpan},
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
	} else if tb.cbuf != nil && tb.cbuf.Cap()-tb.coff[GlobalHeap] < int64(x) {
		t.Fatal("Table.cbuf/coff: range out of bounds")
	}
}

func TestNewDrawTable(t *testing.T) {
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
		tb, _ := NewDrawTable(x.ng, x.nd, x.nm, x.nj)
		tb.check(x.ng, x.nd, x.nm, x.nj, t)
		tb.Free()
	}
}

func TestSetConstBuf(t *testing.T) {
	ng, nd, nm, nj := 1, 1, 1, 1
	tb, _ := NewDrawTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)

	buf, err := ctxt.GPU().NewBuffer(int64(tb.ConstSize()+blockSize), true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}

	if buf, off := tb.SetConstBuf(nil, 0); buf != nil || off != 0 {
		t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant <nil>, 0", buf, off)
	}
	if tb.coff[GlobalHeap] != 0 {
		t.Fatalf("Table.coff[GlobalHeap]:\nhave %d\nwant 0", tb.coff[GlobalHeap])
	}
	if buf, off := tb.SetConstBuf(buf, blockSize); buf != nil || off != 0 {
		t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant <nil>, 0", buf, off)
	}
	if tb.coff[GlobalHeap] != blockSize {
		t.Fatalf("Table.coff[GlobalHeap]:\nhave %d\nwant %d", tb.coff[GlobalHeap], blockSize)
	}
	if buf_, off := tb.SetConstBuf(nil, blockSize*2); buf_ != buf || off != blockSize {
		t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant %v, %d", buf_, off, buf, blockSize)
	}
	if tb.coff[GlobalHeap] != 0 {
		t.Fatalf("Table.coff[GlobalHeap]:\nhave %d\nwant 0", tb.coff[GlobalHeap])
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
		tb, _ := NewDrawTable(x.ng, x.nd, x.nm, x.nj)
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
			if x := goff + x; tb.coff[GlobalHeap] != x {
				t.Fatalf("Table.coff[GlobalHeap]:\nhave %d\nwant %d", tb.coff[GlobalHeap], x)
			}
			if x := doff + x; tb.coff[DrawableHeap] != x {
				t.Fatalf("Table.coff[DrawableHeap]:\nhave %d\nwant %d", tb.coff[DrawableHeap], x)
			}
			if x := moff + x; tb.coff[MaterialHeap] != x {
				t.Fatalf("Table.coff[MaterialHeap]:\nhave %d\nwant %d", tb.coff[MaterialHeap], x)
			}
			if x := joff + x; tb.coff[JointHeap] != x {
				t.Fatalf("Table.coff[JointHeap]:\nhave %d\nwant %d", tb.coff[JointHeap], x)
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
		tb, _ := NewDrawTable(x.ng, x.nd, x.nm, x.nj)
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
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	p.Ortho(-1, 1, -1, 1, -1, 1)
	rnd := float32(0.25)
	vport := driver.Viewport{
		X:      0,
		Y:      0,
		Width:  600,
		Height: 360,
		Znear:  0,
		Zfar:   1,
	}

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
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	p.Ortho(-1, 1, -1, 1, -1, 1)
	rnd := float32(0.25)
	vport := driver.Viewport{
		X:      0,
		Y:      0,
		Width:  600,
		Height: 360,
		Znear:  0,
		Zfar:   1,
	}

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
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	checkSlicesT(tb.Drawable(0)[28:], unsafe.Slice((*float32)(unsafe.Pointer(&id)), 1), t, fmt.Sprintf(s, 28))
}

func TestDrawableWriteN(t *testing.T) {
	ng, nd, nm, nj := 3, 10, 10, 10
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	var norm linear.M3
	wld.Scale(0.5, 2, 3)
	norm.FromM4(&wld)
	norm.Invert(&norm)
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
	checkSlicesT(tb.Drawable(0)[28:], unsafe.Slice((*float32)(unsafe.Pointer(&id)), 1), t, fmt.Sprintf(s, 28))
	s = "Table.Drawable(3)[%d:]"
	checkSlicesT(tb.Drawable(3)[:], unsafe.Slice((*float32)(unsafe.Pointer(&wld)), 16), t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Drawable(3)[16:], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, fmt.Sprintf(s, 16))
	s = "Table.Drawable(9)[%d:]"
	checkSlicesT(tb.Drawable(9)[28:], unsafe.Slice((*float32)(unsafe.Pointer(&id1)), 1), t, fmt.Sprintf(s, 28))
}

func TestMaterialWrite(t *testing.T) {
	ng, nd, nm, nj := 1, 1, 1, 0
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	tb, _ := NewDrawTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var jnt linear.M4
	var norm linear.M3
	jnt.Scale(-0.5, -0.25, 0.125)
	norm.FromM4(&jnt)
	norm.Invert(&norm)
	norm.Transpose(&norm)

	tb.Joint(0)[0].SetJoint(&jnt)
	tb.Joint(0)[0].SetNormal(&norm)
	tb.Joint(0)[MaxJoint-1].SetJoint(&jnt)

	s := "Table.Joint(0)[0][%d:]"
	checkSlicesT(tb.Joint(0)[0][:], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Joint(0)[0][12:], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, fmt.Sprintf(s, 12))
	s = fmt.Sprintf("Table.Joint(0)[%d]%s", MaxJoint-1, "[%d:]")
	checkSlicesT(tb.Joint(0)[MaxJoint-1][:], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, fmt.Sprintf(s, 0))
}

func TestJointWriteN(t *testing.T) {
	ng, nd, nm, nj := 3, 10, 10, 10
	tb, _ := NewDrawTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	sz := int64(tb.ConstSize())
	buf, err := ctxt.GPU().NewBuffer(sz, true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	var jnt linear.M4
	var norm linear.M3
	jnt.Scale(-0.5, -0.25, 0.125)
	norm.FromM4(&jnt)
	norm.Invert(&norm)
	norm.Transpose(&norm)

	tb.Joint(0)[0].SetJoint(&jnt)
	tb.Joint(0)[0].SetNormal(&norm)
	tb.Joint(0)[MaxJoint-1].SetJoint(&jnt)
	tb.Joint(5)[MaxJoint/2].SetJoint(&jnt)
	tb.Joint(9)[1].SetNormal(&norm)
	tb.Joint(9)[MaxJoint-1].SetJoint(&jnt)
	tb.Joint(9)[MaxJoint-1].SetNormal(&norm)

	s := "Table.Joint(0)[0][%d:]"
	checkSlicesT(tb.Joint(0)[0][:], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Joint(0)[0][12:], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, fmt.Sprintf(s, 12))
	s = fmt.Sprintf("Table.Joint(0)[%d]%s", MaxJoint-1, "[%d:]")
	checkSlicesT(tb.Joint(0)[MaxJoint-1][:], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, fmt.Sprintf(s, 0))

	s = fmt.Sprintf("Table.Joint(5)[%d]%s", MaxJoint/2, "[%d:]")
	checkSlicesT(tb.Joint(5)[MaxJoint/2][:], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, fmt.Sprintf(s, 0))

	s = "Table.Joint(9)[1][%d:]"
	checkSlicesT(tb.Joint(9)[1][12:], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, fmt.Sprintf(s, 12))
	s = fmt.Sprintf("Table.Joint(9)[%d]%s", MaxJoint-1, "[%d:]")
	checkSlicesT(tb.Joint(9)[MaxJoint-1][:], []float32{
		jnt[0][0], jnt[1][0], jnt[2][0], jnt[3][0],
		jnt[0][1], jnt[1][1], jnt[2][1], jnt[3][1],
		jnt[0][2], jnt[1][2], jnt[2][2], jnt[3][2],
	}, t, fmt.Sprintf(s, 0))
	checkSlicesT(tb.Joint(9)[MaxJoint-1][12:], []float32{
		norm[0][0], norm[0][1], norm[0][2], 0,
		norm[1][0], norm[1][1], norm[1][2], 0,
		norm[2][0], norm[2][1], norm[2][2], 0,
	}, t, fmt.Sprintf(s, 12))
}

func TestSetGraph(t *testing.T) {
	ng, nd, nm, nj := 1, 10, 6, 3
	tb, _ := NewDrawTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)

	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		t.Fatalf("driver.GPU.NewCmdBuffer failed:\n%#v", err)
	}
	defer cb.Destroy()
	if err = cb.Begin(); err != nil {
		t.Fatalf("driver.CmdBuffer.Begin failed:\n%#v", err)
	}

	buf, err := ctxt.GPU().NewBuffer(int64(tb.ConstSize()), true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	img, err := ctxt.GPU().NewImage(driver.RGBA8Unorm, driver.Dim3D{Width: 256, Height: 256},
		1, 1, 1, driver.UGeneric)
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
		t.Fatalf("driver.GPU.NewSampler failed:\n%#v", err)
	}
	defer splr.Destroy()

	// NOTE: Keep this up to date.
	for i := range ng {
		tb.SetShadowMap(i, iv, splr)
		tb.SetIrradiance(i, iv, splr)
		tb.SetLD(i, iv, splr)
		tb.SetDFG(i, iv, splr)
	}
	for i := range nm {
		tb.SetBaseColor(i, iv, splr)
		tb.SetMetalRough(i, iv, splr)
		tb.SetNormalMap(i, iv, splr)
		tb.SetOcclusionMap(i, iv, splr)
		tb.SetEmissiveMap(i, iv, splr)
	}

	type testCase struct {
		start int
		cpy   []int
	}
	cases := make([]testCase, 0, 9+ng+nd+nm+nj)
	// NOTE: Assuming global/drawable/material/joint order.
	cases = append(cases, []testCase{
		{GlobalHeap, []int{0, 0, 0, 0}},
		{GlobalHeap, []int{ng - 1, nd - 1, nm - 1, nj - 1}},
		{GlobalHeap, []int{ng / 2, nd / 2, nm / 2, nj / 2}},
		{GlobalHeap, []int{0, nd / 3, nm / 3}},
		{GlobalHeap, []int{ng - 1, 0}},
		{DrawableHeap, []int{0, nm / 2, nj - 1}},
		{DrawableHeap, []int{nd - 1, 0}},
		{MaterialHeap, []int{0, 0}},
		{MaterialHeap, []int{nm - 1, nj - 1}},
	}...)
	inds := make([]int, 0, max(ng, nd, nm, nj))
	for i := range cap(inds) {
		inds = append(inds, i)
	}
	for i, x := range [4]int{
		GlobalHeap:   ng,
		DrawableHeap: nd,
		MaterialHeap: nm,
		JointHeap:    nj,
	} {
		for j := range x {
			cases = append(cases, testCase{i, inds[j : j+1]})
		}
	}

	// Every case is valid, so SetGraph must not panic.
	for _, x := range cases {
		tb.SetGraph(cb, x.start, x.cpy)
	}
}

func TestSetCBFail(t *testing.T) {
	ng, nd, nm, nj := 2, 8, 6, 8
	tb, _ := NewDrawTable(ng, nd, nm, nj)
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
	tb, _ := NewDrawTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	img, err := ctxt.GPU().NewImage(driver.RGBA8Unorm, driver.Dim3D{Width: 256, Height: 256},
		1, 1, 1, driver.UShaderSample)
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
		t.Fatalf("driver.GPU.NewSampler failed:\n%#v", err)
	}
	defer splr.Destroy()

	wcpy := "descriptor heap copy out of bounds"
	wtex := "nil texture"
	wsplr := "nil sampler"
	for _, c := range [...]struct {
		s string
		f func(*DrawTable, int, driver.ImageView, driver.Sampler)
	}{
		{"ShadowMap", (*DrawTable).SetShadowMap},
		{"Irradiance", (*DrawTable).SetIrradiance},
		{"LD", (*DrawTable).SetLD},
		{"DFG", (*DrawTable).SetDFG},
		{"BaseColor", (*DrawTable).SetBaseColor},
		{"MetalRough", (*DrawTable).SetMetalRough},
		{"NormalMap", (*DrawTable).SetNormalMap},
		{"OcclusionMap", (*DrawTable).SetOcclusionMap},
		{"EmissiveMap", (*DrawTable).SetEmissiveMap},
	} {
		t.Run(c.s, func(t *testing.T) {
			s := "Table.Set" + c.s + ":\nhave %#v\nwant %#v"
			t.Run("cpy", func(t *testing.T) {
				defer func() {
					if x := recover(); x != wcpy {
						t.Fatalf(s, x, wcpy)
					}
					defer func() {
						if x := recover(); x != wcpy {
							t.Fatalf(s, x, wcpy)
						}
					}()
					c.f(tb, -1, iv, splr)
				}()
				c.f(tb, 6, iv, splr)
			})
			t.Run("tex", func(t *testing.T) {
				defer func() {
					if x := recover(); x != wtex {
						t.Fatalf(s, x, wtex)
					}
				}()
				c.f(tb, 0, nil, splr)
			})
			t.Run("splr", func(t *testing.T) {
				defer func() {
					if x := recover(); x != wsplr {
						t.Fatalf(s, x, wsplr)
					}
				}()
				c.f(tb, 0, iv, nil)
			})
		})
	}
}

func TestConstFail(t *testing.T) {
	ng, nd, nm, nj := 2, 8, 6, 8
	tb, _ := NewDrawTable(ng, nd, nm, nj)
	tb.check(ng, nd, nm, nj, t)
	defer tb.Free()

	buf, err := ctxt.GPU().NewBuffer(int64(tb.ConstSize()), true, driver.UShaderConst)
	if err != nil {
		t.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	defer buf.Destroy()

	tb.SetConstBuf(buf, 0)

	want := "descriptor heap copy out of bounds"
	for _, c := range [...]struct {
		s string
		f func(int)
	}{
		{"Frame", func(cpy int) { tb.Frame(cpy) }},
		{"Light", func(cpy int) { tb.Light(cpy) }},
		{"Shadow", func(cpy int) { tb.Shadow(cpy) }},
		{"Drawable", func(cpy int) { tb.Drawable(cpy) }},
		{"Material", func(cpy int) { tb.Material(cpy) }},
		{"Joint", func(cpy int) { tb.Joint(cpy) }},
	} {
		t.Run(c.s, func(t *testing.T) {
			s := "Table." + c.s + ":\nhave %#v\nwant %#v"
			defer func() {
				if x := recover(); x != want {
					t.Fatalf(s, x, want)
				}
				defer func() {
					if x := recover(); x != want {
						t.Fatalf(s, x, want)
					}
				}()
				c.f(-2)
			}()
			c.f(8)
		})
	}
}
