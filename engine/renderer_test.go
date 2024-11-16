// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"testing"

	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

// checkInit checks that r's fields are valid.
func (r *Renderer) checkInit(width, height int, t *testing.T) {
	if len(r.cb) != cap(r.ch) {
		t.Fatal("Renderer.init: len(cb) differs from cap(ch)")
	}
	for range cap(r.ch) {
		wk := <-r.ch
		if len(wk.Work) != 1 {
			t.Fatal("Renderer.init: len((<-ch).Work) should have exactly 1 element")
		}
		cb := wk.Work[0]
		idx := wk.Custom.(int)
		if cb.IsRecording() {
			t.Fatalf("Renderer.init: cb[%d] should not have begun", idx)
		}
	}
	for i, cb := range r.cb {
		r.ch <- &driver.WorkItem{
			Work:   []driver.CmdBuffer{cb},
			Custom: i,
		}
	}
	for i, rt := range [2]*Texture{r.hdr, r.ds} {
		var s string
		if i == 0 {
			s = "hdr"
		} else {
			s = "ds"
		}
		if width != rt.Width() || height != rt.Height() {
			t.Fatalf("Renderer.init: %s size mismatch", s)
		}
		if rt.Layers() != 1 {
			t.Fatalf("Renderer.init: %s should have exactly 1 layer", s)
		}
		if rt.Levels() != 1 {
			t.Fatalf("Renderer.init: %s should have exactly 1 level", s)
		}
	}
}

// checkFree checks that r's fields are invalid.
func (r *Renderer) checkFree(t *testing.T) {
	for i, cb := range r.cb {
		if cb != nil {
			t.Fatalf("Renderer.free: cb[%d] should be nil", i)
		}
	}
	if r.ch != nil {
		t.Fatal("Renderer.free: ch should be nil")
	}
	if r.nlight != 0 {
		t.Fatal("Renderer.free: nlight should be 0")
	}
	if r.hdr != nil {
		t.Fatal("Renderer.free: hdr should be nil")
	}
	if r.ds != nil {
		t.Fatal("Renderer.free: ds should be nil")
	}
}

func TestOnscreen(t *testing.T) {
	width := 480
	height := 270
	win, err := wsi.NewWindow(width, height, "TestOnscreen")
	if err != nil {
		t.Fatalf("Onscreen: wsi.NewWindow failed:\n%#v", err)
	}
	for range 2 {
		rend, err := NewOnscreen(win)
		rend.checkInit(width, height, t)
		if err != nil {
			t.Fatalf("Onscreen.New failed:\n%#v", err)
		}
		if win != rend.Window() {
			t.Fatal("Onscreen.Window: windows differ")
		}
		rend.Free()
		rend.checkFree(t)
		if rend.Window() != nil {
			t.Fatal("Onscreen.Window: window should be nil")
		}
	}
}

func TestOffscreen(t *testing.T) {
	width := 800
	height := 600
	for range 2 {
		rend, err := NewOffscreen(width, height)
		rend.checkInit(width, height, t)
		if err != nil {
			t.Fatalf("Offscreen.New failed:\n%#v", err)
		}
		rt := rend.Target()
		if width != rt.Width() || height != rt.Height() {
			t.Fatal("Offscreen.Target: target size mismatch")
		}
		if rt.Layers() != 1 {
			t.Fatal("Offscreen.Target: target should have exactly 1 layer")
		}
		if rt.Levels() != 1 {
			t.Fatal("Offscreen.Target: target should have exactly 1 level")
		}
		rend.Free()
		rend.checkFree(t)
		rt = rend.Target()
		if rt != nil {
			t.Fatal("Offscreen.Target: target should be nil")
		}
	}
}
