// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/driver"
)

// tDesc contains lists of descriptors for testing.
var tDesc = [...][]driver.Descriptor{
	{
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SVertex, Nr: 1, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 2, Len: 8},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex, Nr: 3, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 4, Len: 3},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DSampler, Stages: driver.SVertex | driver.SFragment, Nr: 3, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SFragment, Nr: 0, Len: 4},
		{Type: driver.DConstant, Stages: driver.SVertex, Nr: 1, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DSampler, Stages: driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 3, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex, Nr: 0, Len: 1},
		{Type: driver.DSampler, Stages: driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 1, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
		{Type: driver.DBuffer, Stages: driver.SVertex, Nr: 3, Len: 1},
		{Type: driver.DImage, Stages: driver.SFragment, Nr: 4, Len: 1},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DSampler, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
		{Type: driver.DBuffer, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DImage, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DImage, Stages: driver.SFragment, Nr: 3, Len: 1},
		{Type: driver.DImage, Stages: driver.SVertex, Nr: 4, Len: 1},
		{Type: driver.DConstant, Stages: driver.SFragment, Nr: 5, Len: 1},
	},
	{
		{Type: driver.DSampler, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 12},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 4},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 3, Len: 1},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 4, Len: 4},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 5, Len: 2},
	},
}

// validDescTypeN validates descriptor type counts in h.
// It assumes that h was created using ds as parameter.
func validDescTypeN(h *descHeap, ds []driver.Descriptor) bool {
	var nbuf, nimg, nconst, ntex, nsplr int
	for i := range ds {
		switch ds[i].Type {
		case driver.DBuffer:
			nbuf += ds[i].Len
		case driver.DImage:
			nimg += ds[i].Len
		case driver.DConstant:
			nconst += ds[i].Len
		case driver.DTexture:
			ntex += ds[i].Len
		case driver.DSampler:
			nsplr += ds[i].Len
		default:
			panic("unexpected invalid descriptor type")
		}
	}
	if nbuf != h.nbuf || nimg != h.nimg || nconst != h.nconst || ntex != h.ntex || nsplr != h.nsplr {
		return false
	}
	return true
}

func TestDescHeap(t *testing.T) {
	zh := descHeap{}
	for _, ds := range tDesc {
		call := fmt.Sprintf("tDrv.NewDescHeap(%v)", ds)
		// NewDescHeap.
		if h, err := tDrv.NewDescHeap(ds); err == nil {
			if h == nil {
				t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
				continue
			}
			h := h.(*descHeap)
			if h.d != &tDrv {
				t.Errorf("%s: h.d\nhave %v\nwant %v", call, h.d, &tDrv)
			}
			if h.layout == zh.layout {
				t.Errorf("%s: h.layout\nhave %v\nwant valid handle", call, h.layout)
			}
			if h.pool != zh.pool {
				t.Errorf("%s: h.pool\nhave %v\nwant null handle", call, h.pool)
			}
			if h.sets != nil {
				t.Errorf("%s: h.sets\nhave %v\nwant nil", call, h.sets)
			}
			if !validDescTypeN(h, ds) {
				t.Errorf("%s: h.n[buf|img|const|tex|splr]: count mismatch", call)
			}
			// Len.
			n := h.Len()
			if n != 0 {
				t.Errorf("h.Len()\nhave %v\nwant 0", n)
			}
			// Destroy.
			h.Destroy()
			if h.d != nil {
				t.Errorf("h.Destroy(): h.d\nhave %v\nwant nil", h.d)
			}
			if h.layout != zh.layout {
				t.Errorf("h.Destroy(): h.layout\nhave %v\nwant null handle", h.layout)
			}
		} else if h != nil {
			t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, h, err, err)
		} else {
			t.Logf("(error) %s: %v", call, err)
		}
	}
}

func TestDescHeapNew(t *testing.T) {
	n := [...]int{1, 2, 0, 3, 2, 1, 4, 7, 10, 16, 32, 64, 100, 300, 0, 15}
	zh := descHeap{}
	for _, ds := range tDesc {
		ic, err := tDrv.NewDescHeap(ds)
		if err != nil {
			t.Errorf("tDrv.NewDescHeap(%v) failed, cannot test New method", ds)
			continue
		}
		h := ic.(*descHeap)
		for _, n := range n {
			if err = h.New(n); err == nil {
				if h.pool == zh.pool {
					t.Errorf("h.New(%d): h.pool\nhave %v\nwant valid handle", n, h.pool)
				}
				if len(h.sets) != n {
					t.Errorf("h.New(%d): len(h.sets)\nhave %d\nwant %d", n, len(h.sets), n)
				}
			} else {
				t.Logf("(error) h.New(%d): %v", n, err)
			}
		}
		if err := h.New(-1); err == nil {
			t.Logf("h.New(-1)\nhave nil\nwant non-nil")
		}
		h.Destroy()
		if len(h.sets) != 0 {
			t.Errorf("h.Destroy(): len(h.sets)\nhave %d\nwant 0", len(h.sets))
		}
	}
}

func TestDescTable(t *testing.T) {
	dh := make([]driver.DescHeap, len(tDesc))
	defer func() {
		for _, h := range dh {
			if h != nil {
				h.Destroy()
			}
		}
	}()
	hs := make([][]driver.DescHeap, len(dh))
	for i, ds := range tDesc {
		h, err := tDrv.NewDescHeap(ds)
		if err != nil {
			t.Errorf("tDrv.NewDescHeap(%v) failed, cannot test New method", ds)
			return
		}
		dh[i] = h
		hs[i] = []driver.DescHeap{h}
	}
	hs = append(hs,
		[]driver.DescHeap{dh[0], dh[2]},
		[]driver.DescHeap{dh[0], dh[3]},
		[]driver.DescHeap{dh[3], dh[4]},
		[]driver.DescHeap{dh[0], dh[1], dh[2]},
		[]driver.DescHeap{dh[1], dh[2], dh[3], dh[4]},
		[]driver.DescHeap{dh[5], dh[0]},
		[]driver.DescHeap{dh[5], dh[3]},
		[]driver.DescHeap{dh[6], dh[1]},
		[]driver.DescHeap{dh[6], dh[4]},
		[]driver.DescHeap{dh[6], dh[0], dh[1]},
		[]driver.DescHeap{dh[7], dh[6]},
		[]driver.DescHeap{dh[8], dh[0], dh[4]},
		[]driver.DescHeap{dh[9], dh[3], dh[4]},
		// Sets have separate namespaces, so these
		// should not clash.
		[]driver.DescHeap{dh[10], dh[1]},
		[]driver.DescHeap{dh[10], dh[1], dh[2], dh[3]},
		[]driver.DescHeap{dh[11], dh[10]},
		[]driver.DescHeap{dh[12], dh[4], dh[1], dh[0]},
	)
	zt := descTable{}
	for i := range hs {
		call := fmt.Sprintf("tDrv.NewDescTable(%v)", hs[i])
		// NewDescTable.
		if dt, err := tDrv.NewDescTable(hs[i]); err == nil {
			if t == nil {
				t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
				continue
			}
			dt := dt.(*descTable)
			if dt.d != &tDrv {
				t.Errorf("%s: dt.d\nhave %v\nwant %v", call, dt.d, &tDrv)
			}
			if dt.layout == zt.layout {
				t.Errorf("%s: dt.layout\nhave %v\nwant valid handle", call, dt.layout)
			}
			// Heap.
			for j := range hs[i] {
				if x := dt.Heap(j); x != hs[i][j] {
					t.Errorf("dt.Heap(%d)\nhave %v\nwant %v", j, x, hs[i][j])
				}
			}
			// Len.
			if n := dt.Len(); n != len(dt.h) {
				t.Errorf("dt.Len()\nhave %d\nwant %d", n, len(dt.h))
			}
			// Destroy.
			dt.Destroy()
			if dt.d != nil {
				t.Errorf("dt.Destroy(): dt.d\nhave %v\nwant nil", dt.d)
			}
			if dt.layout != zt.layout {
				t.Errorf("dt.Destroy(): dt.layout\nhave %v\nwant null handle", dt.layout)
			}
		} else if dt != nil {
			t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, dt, err, err)
		} else {
			t.Logf("(error) %s: %v", call, err)
		}
	}
}
