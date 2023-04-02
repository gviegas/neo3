// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
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
