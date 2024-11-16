// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"math"
	"strconv"
	"testing"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/linear"
	"gviegas/neo3/wsi"
)

func TestCtxt(t *testing.T) {
	drv := ctxt.Driver()
	if drv == nil {
		t.Fatal("ctxt.Driver: unexpected nil driver.Driver")
	}
	gpu := ctxt.GPU()
	if gpu == nil {
		t.Fatal("ctxt.GPU: unexpected nil driver.GPU")
	}
	if d := ctxt.Driver(); d != drv {
		t.Fatalf("ctxt.Driver: ctxt mismatch\nhave %v\nwant %v", d, drv)
	}
	if u := ctxt.GPU(); u != gpu {
		t.Fatalf("ctxt.GPU: ctxt mismatch\nhave %v\nwant %v", u, gpu)
	}
	if u, err := drv.Open(); u != gpu || err != nil {
		t.Fatalf("drv.Open: ctxt mismatch\nhave %v, %v\nwant %v, nil", u, err, gpu)
	}
}

func TestID(t *testing.T) {
	type id int
	var m dataMap[id, string]

	checkGet := func(id id, want string) {
		if data := m.get(id); *data != want {
			t.Fatalf("dataMap.get\nhave %s\nwant %s", *data, want)
		}
	}
	checkLen := func(want int) {
		if len := m.len(); len != want {
			t.Fatalf("dataMap.len()\nhave %d\nwant %d", len, want)
		}
	}
	checkInsert := func(data string) id {
		len := m.len()
		id := m.insert(data)
		checkGet(id, data)
		checkLen(len + 1)
		return id
	}
	checkRemove := func(id id, want string) string {
		len := m.len()
		data := m.remove(id)
		if data != want {
			t.Fatalf("dataMap.remove()\nhave %s\nwant %s", data, want)
		}
		checkLen(len - 1)
		return data
	}

	var id0, id1, id2, id3 id
	var data0 string

	checkLen(0)
	id0 = checkInsert("hi")
	checkGet(id0, "hi")
	data0 = checkRemove(id0, "hi")
	id0 = checkInsert(data0)
	checkGet(id0, data0)
	checkRemove(id0, data0)

	checkLen(0)
	id0 = checkInsert("bye")
	id1 = checkInsert("bye bye")
	checkGet(id1, "bye bye")
	checkGet(id0, "bye")
	data0 = checkRemove(id0, "bye")
	id0 = checkInsert(data0)
	checkGet(id1, "bye bye")
	checkGet(id0, "bye")
	id2 = checkInsert(data0)
	checkRemove(id1, "bye bye")
	checkRemove(id0, data0)
	checkRemove(id2, data0)

	checkLen(0)
	id0, id1, id2, id3 = checkInsert("a"), checkInsert("b"), checkInsert("c"), checkInsert("d")
	checkRemove(id2, "c")
	id2 = checkInsert("e")
	checkGet(id0, "a")
	checkGet(id1, "b")
	checkGet(id2, "e")
	checkGet(id3, "d")
	checkRemove(id1, "b")
	id1 = checkInsert("f")
	checkGet(id0, "a")
	checkGet(id1, "f")
	checkGet(id2, "e")
	checkGet(id3, "d")
	checkRemove(id0, "a")
	id0 = checkInsert("g")
	checkGet(id0, "g")
	checkGet(id1, "f")
	checkGet(id2, "e")
	checkGet(id3, "d")
	checkRemove(id3, "d")
	id3 = checkInsert("h")
	checkGet(id0, "g")
	checkGet(id1, "f")
	checkGet(id2, "e")
	checkGet(id3, "h")
	checkRemove(id2, "e")
	checkRemove(id1, "f")
	id2 = checkInsert("i")
	checkGet(id0, "g")
	checkGet(id2, "i")
	checkGet(id3, "h")
	checkRemove(id0, "g")
	id1 = checkInsert("j")
	checkGet(id1, "j")
	checkGet(id2, "i")
	checkGet(id3, "h")
	id0 = checkInsert("k")
	checkRemove(id3, "h")
	checkGet(id0, "k")
	checkGet(id1, "j")
	checkGet(id2, "i")
	checkRemove(id2, "i")
	checkRemove(id1, "j")
	checkRemove(id0, "k")

	// XXX: This is somewhat frail since it relies on
	// implementation details.
	checkBits := func(wantLen, wantRem int) {
		if len := len(m.ids); len != wantLen {
			t.Fatalf("len(dataMap.ids)\nhave: %d\nwant %d", len, wantLen)
		}
		if len := m.idMap.Len(); len != wantLen {
			t.Fatalf("dataMap.idMap.Len()\nhave: %d\nwant %d", len, wantLen)
		}
		if rem := m.idMap.Rem(); rem != wantRem {
			t.Fatalf("dataMap.idMap.Rem()\nhave: %d\nwant %d", rem, wantRem)
		}
	}

	checkLen(0)
	checkBits(32, 32)
	ids := make([]id, 32)
	for i := range ids {
		ids[i] = checkInsert(strconv.Itoa(i))
	}
	checkBits(32, 0)
	for i, id := range ids {
		checkRemove(id, strconv.Itoa(i))
	}
	checkBits(32, 32)
	for i := range ids {
		ids[i] = checkInsert(strconv.Itoa(i))
	}
	checkBits(32, 0)
	ids = append(ids, checkInsert("32"))
	checkBits(64, 31)
	for i, id := range ids {
		checkRemove(id, strconv.Itoa(i))
	}

	checkLen(0)
	checkBits(64, 64)
	ids = append(ids, make([]id, 256-len(ids))...)
	for i := range ids {
		ids[i] = checkInsert(strconv.Itoa(i))
	}
	checkBits(256, 0)
	for i, id := range ids {
		checkRemove(id, strconv.Itoa(i))
	}
	checkBits(256, 256)
	id0 = checkInsert("0")
	checkBits(256, 255)
	for i := range ids {
		ids[i] = checkInsert(strconv.Itoa(i + 1))
	}
	checkBits(512, 255)
	for i, id := range ids {
		checkRemove(id, strconv.Itoa(i+1))
	}
	checkBits(512, 511)
	checkLen(1)
	checkRemove(id0, "0")
	checkBits(512, 512)
}

func TestLight(t *testing.T) {
	t.Run("Distant", func(t *testing.T) {
		dist := DistantLight{
			Direction: linear.V3{0, 0, -1},
			Intensity: 100_000,
			R:         1,
			G:         1,
			B:         1,
		}
		light := dist.Light()
		if other := dist.Light(); light != other {
			t.Fatalf("DistantLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// Intensity should not go below 0.
		dist.Intensity = -dist.Intensity
		if other := dist.Light(); light == other {
			t.Fatalf("DistantLight.Light: created Lights don't differ\n1st: %v\n2nd: %v", light, other)
		} else {
			light = other
		}
		dist.Intensity = 0
		if other := dist.Light(); light != other {
			t.Fatalf("DistantLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
	})

	t.Run("Point", func(t *testing.T) {
		point := PointLight{
			Position:  linear.V3{},
			Range:     0.5,
			Intensity: 1,
			R:         1,
			G:         1,
			B:         1,
		}
		light := point.Light()
		if other := point.Light(); light != other {
			t.Fatalf("PointLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// Intensity should not go below 0.
		point.Intensity = -1
		if other := point.Light(); light == other {
			t.Fatalf("PointLight.Light: created Lights don't differ\n1st: %v\n2nd: %v", light, other)
		} else {
			light = other
		}
		point.Intensity = 0
		if other := point.Light(); light != other {
			t.Fatalf("PointLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
	})

	t.Run("Spot", func(t *testing.T) {
		spot := SpotLight{
			Direction:  linear.V3{0, 0, -1},
			Position:   linear.V3{},
			InnerAngle: math.Pi / 16,
			OuterAngle: math.Pi / 4,
			Range:      36,
			Intensity:  100,
			R:          1,
			G:          1,
			B:          1,
		}
		light := spot.Light()
		if other := spot.Light(); light != other {
			t.Fatalf("SpotLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// Intensity and InnerAngle should not go below 0.
		// OuterAngle should not go above Pi/2.
		spot.Intensity = -123
		spot.InnerAngle = -math.Pi
		spot.OuterAngle = math.Pi
		if other := spot.Light(); light == other {
			t.Fatalf("SpotLight.Light: created Lights don't differ\n1st: %v\n2nd: %v", light, other)
		} else {
			light = other
		}
		spot.Intensity = 0
		spot.InnerAngle = 0
		spot.OuterAngle = math.Pi / 2
		if other := spot.Light(); light != other {
			t.Fatalf("SpotLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// InnerAngle should be less than OuterAngle.
		spot.InnerAngle = math.Pi / 4
		spot.OuterAngle = math.Pi/4 + 1e-6
		light = spot.Light()
		spot.OuterAngle = spot.InnerAngle
		if other := spot.Light(); light != other {
			t.Fatalf("SpotLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
	})
}

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
