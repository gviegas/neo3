// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"testing"

	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

func TestRenderer(t *testing.T) {
	checkInit := func(rend *Renderer, width, height int) {
		if len(rend.cb) != cap(rend.ch) {
			t.Fatal("Renderer.init: len(cb) differs from cap(ch)")
		}
		for range cap(rend.ch) {
			wk := <-rend.ch
			if len(wk.Work) != 1 {
				t.Fatal("Renderer.init: len((<-ch).Work) should have exactly 1 element")
			}
			cb := wk.Work[0]
			idx := wk.Custom.(int)
			if cb.IsRecording() {
				t.Fatalf("Renderer.init: cb[%d] should not have begun", idx)
			}
		}
		for i, cb := range rend.cb {
			rend.ch <- &driver.WorkItem{
				Work:   []driver.CmdBuffer{cb},
				Custom: i,
			}
		}
		for i, rt := range [2]*Texture{rend.hdr, rend.ds} {
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
	checkFree := func(rend *Renderer) {
		for i, cb := range rend.cb {
			if cb != nil {
				t.Fatalf("Renderer.free: cb[%d] should be nil", i)
			}
		}
		if rend.ch != nil {
			t.Fatal("Renderer.free: ch should be nil")
		}
		if rend.nlight != 0 {
			t.Fatal("Renderer.free: nlight should be 0")
		}
		if rend.hdr != nil {
			t.Fatal("Renderer.free: hdr should be nil")
		}
		if rend.ds != nil {
			t.Fatal("Renderer.free: ds should be nil")
		}
	}

	for range 2 {
		var rend Renderer
		if err := rend.init(800, 600); err != nil {
			t.Fatalf("Renderer.init failed:\n%#v", err)
		}
		checkInit(&rend, 800, 600)
		rend.free()
		checkFree(&rend)
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
		if err != nil {
			t.Fatalf("Onscreen.New failed:\n%#v", err)
		}
		if win != rend.Window() {
			t.Fatal("Onscreen.Window: windows differ")
		}
		rend.Free()
		if rend.Window() != nil {
			t.Fatal("Onscreen.Window: window should be nil")
		}
	}
}

func TestOffscreen(t *testing.T) {
	width := 480
	height := 270
	for range 2 {
		rend, err := NewOffscreen(width, height)
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
		rt = rend.Target()
		if rt != nil {
			t.Fatal("Offscreen.Target: target should be nil")
		}
	}
}
