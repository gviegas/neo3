// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
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
		} else {
			csz += n * int(x.spn)
		}
	}
	csz *= blockSize
	if x := tb.ConstSize(); x != csz {
		t.Fatalf("Table.ConstSize:\nhave %d\nwant %d", x, csz)
	} else if x%blockSize != 0 {
		t.Fatal("Table.ConstSize: misaligned size")
	} else if tb.cbuf != nil && tb.cbuf.Cap()-tb.coff < int64(x) {
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

		wbuf, woff := driver.Buffer(nil), int64(0)
		for _, x := range [3]int64{0, sz / 2, sz - int64(tb.ConstSize())} {
			if hbuf, hoff := tb.SetConstBuf(buf, x); wbuf != hbuf || woff != hoff {
				t.Fatalf("Table.SetConstBuf:\nhave %v, %d\nwant %v, %d", hbuf, hoff, wbuf, woff)
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
