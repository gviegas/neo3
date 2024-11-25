// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"math"
	"strconv"
	"testing"

	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/linear"
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
			t.Fatalf("dataMap.len\nhave %d\nwant %d", len, want)
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
			t.Fatalf("dataMap.remove\nhave %s\nwant %s", data, want)
		}
		checkLen(len - 1)
		return data
	}
	checkAll := func(pairs map[id]string) {
		it := m.all()
		for i, d := range it {
			if v, ok := pairs[i]; !ok {
				t.Fatalf("dataMap.all: generated unexpected pair %d,%s", i, *d)
			} else if v != *d {
				t.Fatalf("dataMap.all: ID %d has wrong data %#v", i, *d)
			}
			delete(pairs, i)
		}
		if len(pairs) != 0 {
			t.Fatal("dataMap.all: missing pair(s)")
		}
		for range it {
			break
		}
	}
	checkOnly := func(ok func(id, *string) bool, pairs map[id]string) {
		it := m.only(ok)
		for i, d := range it {
			if v, ok := pairs[i]; !ok {
				t.Fatalf("dataMap.only: generated unexpected pair %d,%s", i, *d)
			} else if v != *d {
				t.Fatalf("dataMap.only: ID %d has wrong data %#v", i, *d)
			}
			delete(pairs, i)
		}
		if len(pairs) != 0 {
			t.Fatal("dataMap.only: missing pair(s)")
		}
		for range it {
			break
		}
	}

	var id0, id1, id2, id3 id
	var data0 string

	checkLen(0)
	checkAll(nil)
	id0 = checkInsert("hi")
	checkGet(id0, "hi")
	checkAll(map[id]string{id0: "hi"})
	data0 = checkRemove(id0, "hi")
	checkAll(nil)
	id0 = checkInsert(data0)
	checkGet(id0, data0)
	checkAll(map[id]string{id0: data0})
	checkRemove(id0, data0)
	checkAll(nil)

	checkLen(0)
	id0 = checkInsert("bye")
	id1 = checkInsert("bye bye")
	checkGet(id1, "bye bye")
	checkGet(id0, "bye")
	checkAll(map[id]string{id1: "bye bye", id0: "bye"})
	data0 = checkRemove(id0, "bye")
	checkAll(map[id]string{id1: "bye bye"})
	id0 = checkInsert(data0)
	checkGet(id1, "bye bye")
	checkGet(id0, "bye")
	checkAll(map[id]string{id1: "bye bye", id0: data0})
	id2 = checkInsert(data0)
	checkAll(map[id]string{id1: "bye bye", id0: data0, id2: data0})
	checkRemove(id1, "bye bye")
	checkAll(map[id]string{id0: data0, id2: data0})
	checkRemove(id0, data0)
	checkAll(map[id]string{id2: data0})
	checkRemove(id2, data0)
	checkAll(nil)

	checkLen(0)
	checkOnly(func(id, *string) bool { return true }, nil)
	id0, id1, id2, id3 = checkInsert("a"), checkInsert("b"), checkInsert("c"), checkInsert("d")
	checkOnly(func(id, *string) bool { return false }, nil)
	checkOnly(func(id, *string) bool { return true }, map[id]string{id0: "a", id1: "b", id2: "c", id3: "d"})
	checkRemove(id2, "c")
	checkOnly(func(id, *string) bool { return false }, nil)
	checkOnly(func(id, *string) bool { return true }, map[id]string{id0: "a", id1: "b", id3: "d"})
	id2 = checkInsert("e")
	checkGet(id0, "a")
	checkGet(id1, "b")
	checkGet(id2, "e")
	checkGet(id3, "d")
	checkOnly(func(id, *string) bool { return false }, nil)
	checkOnly(func(id, *string) bool { return true }, map[id]string{id0: "a", id1: "b", id3: "d", id2: "e"})
	checkRemove(id1, "b")
	checkOnly(func(id, *string) bool { return true }, map[id]string{id0: "a", id3: "d", id2: "e"})
	checkOnly(func(id, *string) bool { return false }, nil)
	id1 = checkInsert("f")
	checkGet(id0, "a")
	checkGet(id1, "f")
	checkGet(id2, "e")
	checkGet(id3, "d")
	checkOnly(func(i id, _ *string) bool { return i == id1 }, map[id]string{id1: "f"})
	checkOnly(func(i id, _ *string) bool { return i == id0 }, map[id]string{id0: "a"})
	checkRemove(id0, "a")
	checkOnly(func(i id, _ *string) bool { return i == id1 }, map[id]string{id1: "f"})
	checkOnly(func(i id, _ *string) bool { return i == id0 }, nil)
	id0 = checkInsert("g")
	checkGet(id0, "g")
	checkGet(id1, "f")
	checkGet(id2, "e")
	checkGet(id3, "d")
	checkRemove(id3, "d")
	checkOnly(func(_ id, d *string) bool { return len(*d) == 1 }, map[id]string{id0: "g", id1: "f", id2: "e"})
	id3 = checkInsert("h")
	checkGet(id0, "g")
	checkGet(id1, "f")
	checkGet(id2, "e")
	checkGet(id3, "h")
	checkOnly(func(_ id, d *string) bool { return *d != "g" }, map[id]string{id3: "h", id1: "f", id2: "e"})
	checkRemove(id2, "e")
	checkRemove(id1, "f")
	id2 = checkInsert("i")
	checkGet(id0, "g")
	checkGet(id2, "i")
	checkGet(id3, "h")
	checkOnly(func(_ id, d *string) bool { return *d != "i" && *d != "h" }, map[id]string{id0: "g"})
	checkRemove(id0, "g")
	checkOnly(func(_ id, d *string) bool { return *d == "i" || *d == "h" }, map[id]string{id3: "h", id2: "i"})
	id1 = checkInsert("j")
	checkGet(id1, "j")
	checkGet(id2, "i")
	checkGet(id3, "h")
	id0 = checkInsert("k")
	checkRemove(id3, "h")
	checkGet(id0, "k")
	checkGet(id1, "j")
	checkGet(id2, "i")
	checkOnly(func(i id, d *string) bool { return i == id0 && len(*d) == 1 }, map[id]string{id0: "k"})
	checkRemove(id2, "i")
	checkRemove(id1, "j")
	checkRemove(id0, "k")
	checkOnly(func(i id, d *string) bool { return i == id0 && len(*d) == 1 }, nil)

	// XXX: This is somewhat frail since it relies on
	// implementation details.
	checkBits := func(wantLen, wantRem int) {
		if len := len(m.ids); len != wantLen {
			t.Fatalf("len(dataMap.ids)\nhave %d\nwant %d", len, wantLen)
		}
		if len := m.idMap.Len(); len != wantLen {
			t.Fatalf("dataMap.idMap.Len\nhave %d\nwant %d", len, wantLen)
		}
		if rem := m.idMap.Rem(); rem != wantRem {
			t.Fatalf("dataMap.idMap.Rem\nhave %d\nwant %d", rem, wantRem)
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
		if x := light.Direction(); x != dist.Direction {
			t.Fatalf("Light.Direction: values differ\nhave %v\nwant %v", x, dist.Direction)
		}
		if x := light.Intensity(); x != dist.Intensity {
			t.Fatalf("Light.Intensity: values differ\nhave %v\nwant %v", x, dist.Intensity)
		}
		if r, g, b := light.Color(); r != dist.R || g != dist.G || b != dist.B {
			t.Fatalf("Light.Color: values differ\nhave %v,%v,%v\nwant %v,%v,%v", r, g, b, dist.R, dist.G, dist.B)
		}
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
		if x := light.Position(); x != point.Position {
			t.Fatalf("Light.Position: values differ\nhave %v\nwant %v", x, point.Position)
		}
		if x := light.Range(); x != point.Range {
			t.Fatalf("Light.Range: values differ\nhave %v\nwant %v", x, point.Range)
		}
		if x := light.Intensity(); x != point.Intensity {
			t.Fatalf("Light.Intensity: values differ\nhave %v\nwant %v", x, point.Intensity)
		}
		if r, g, b := light.Color(); r != point.R || g != point.G || b != point.B {
			t.Fatalf("Light.Color: values differ\nhave %v,%v,%v\nwant %v,%v,%v", r, g, b, point.R, point.G, point.B)
		}
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
		if x := light.Direction(); x != spot.Direction {
			t.Fatalf("Light.Direction: values differ\nhave %v\nwant %v", x, spot.Direction)
		}
		if x := light.Position(); x != spot.Position {
			t.Fatalf("Light.Position: values differ\nhave %v\nwant %v", x, spot.Position)
		}
		if inner, outer := light.ConeAngles(); math.Abs(float64(inner-spot.InnerAngle)) > 1e-6 || math.Abs(float64(outer-spot.OuterAngle)) > 1e-6 {
			t.Fatalf("Light.ConeAngles: values differ\nhave %v,%v\nwant %v,%v", inner, outer, spot.InnerAngle, spot.OuterAngle)
		}
		if x := light.Range(); x != spot.Range {
			t.Fatalf("Light.Range: values differ\nhave %v\nwant %v", x, spot.Range)
		}
		if x := light.Intensity(); x != spot.Intensity {
			t.Fatalf("Light.Intensity: values differ\nhave %v\nwant %v", x, spot.Intensity)
		}
		if r, g, b := light.Color(); r != spot.R || g != spot.G || b != spot.B {
			t.Fatalf("Light.Color: values differ\nhave %v,%v,%v\nwant %v,%v,%v", r, g, b, spot.R, spot.G, spot.B)
		}
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
