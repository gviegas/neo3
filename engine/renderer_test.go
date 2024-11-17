// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"testing"

	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

// checkInit checks whether r.init worked.
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
	if r.nlight != 0 {
		t.Fatal("Renderer.init: nlight should be 0")
	}
	if r.drawables.len() != 0 {
		t.Fatal("Renderer.init: drawables.len should be 0")
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

// checkFree checks whether r.free worked.
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
	if r.drawables.len() != 0 {
		t.Fatal("Renderer.free: drawables.len should be 0")
	}
	if r.hdr != nil {
		t.Fatal("Renderer.free: hdr should be nil")
	}
	if r.ds != nil {
		t.Fatal("Renderer.free: ds should be nil")
	}
}

// checkNew checks whether NewOnscreen worked.
func (r *Onscreen) checkNew(err error, win wsi.Window, t *testing.T) {
	if err != nil {
		t.Fatalf("NewOnscreen failed:\n%#v", err)
	}
	if win != r.Window() {
		t.Fatal("Onscreen.Window: windows differ")
	}
	r.checkInit(r.Window().Width(), r.Window().Height(), t)
}

// checkFree checks whether r.Free worked.
func (r *Onscreen) checkFree(t *testing.T) {
	if r.Window() != nil {
		t.Fatal("Onscreen.Window: window should be nil")
	}
	r.Renderer.checkFree(t)

}

// checkNew checks whether NewOffscreen worked.
func (r *Offscreen) checkNew(err error, width, height int, t *testing.T) {
	if err != nil {
		t.Fatalf("NewOffscreen failed:\n%#v", err)
	}
	rt := r.Target()
	if width != rt.Width() || height != rt.Height() {
		t.Fatal("Offscreen.Target: target size mismatch")
	}
	if rt.Layers() != 1 {
		t.Fatal("Offscreen.Target: target should have exactly 1 layer")
	}
	if rt.Levels() != 1 {
		t.Fatal("Offscreen.Target: target should have exactly 1 level")
	}
	r.checkInit(width, height, t)
}

// checkFree checks whether r.Free worked.
func (r *Offscreen) checkFree(t *testing.T) {
	rt := r.Target()
	if rt != nil {
		t.Fatal("Offscreen.Target: target should be nil")
	}
	r.Renderer.checkFree(t)
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
		rend.checkNew(err, win, t)
		rend.Free()
		rend.checkFree(t)
	}
	width2 := 400
	height2 := 240
	win2, err := wsi.NewWindow(width2, height2, "TestOnscreen")
	if err != nil {
		t.Fatalf("Onscreen: wsi.NewWindow failed:\n%#v", err)
	}
	for range 2 {
		rend, err := NewOnscreen(win)
		rend2, err2 := NewOnscreen(win2)
		rend.checkNew(err, win, t)
		rend2.checkNew(err2, win2, t)
		rend.Free()
		rend2.Free()
		rend.checkFree(t)
		rend2.checkFree(t)
	}
}

func TestOffscreen(t *testing.T) {
	width := 800
	height := 600
	for range 2 {
		rend, err := NewOffscreen(width, height)
		rend.checkNew(err, width, height, t)
		rend.Free()
		rend.checkFree(t)
	}
	width2 := 256
	height2 := 256
	for range 2 {
		rend, err := NewOffscreen(width, height)
		rend2, err2 := NewOffscreen(width2, height2)
		rend.checkNew(err, width, height, t)
		rend2.checkNew(err2, width2, height2, t)
		rend.Free()
		rend2.Free()
		rend.checkFree(t)
		rend2.checkFree(t)
	}
}

func TestOnscreenOffscreen(t *testing.T) {
	width := [2]int{960, 600}
	height := [2]int{540, 360}
	for i := range 2 {
		ofw, ofh := width[i%2], height[i%2]
		onw, onh := width[(i+1)%2], height[(i+1)%2]
		win, err := wsi.NewWindow(onw, onh, "TestOnscreenOffscreen")
		if err != nil {
			t.Fatalf("OnscreenOffscreen: wsi.NewWindow failed:\n%#v", err)
		}
		ofs, ofe := NewOffscreen(ofw, ofh)
		ons, one := NewOnscreen(win)
		ofs.checkNew(ofe, ofw, ofh, t)
		ons.checkNew(one, win, t)
		ofs.Free()
		ons.Free()
		ofs.checkFree(t)
		ons.checkFree(t)
	}
}
