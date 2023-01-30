// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/driver"
	_ "github.com/gviegas/scene/driver/vk"
)

var gpu driver.GPU

func init() {
	var err error
	for _, d := range driver.Drivers() {
		if gpu, err = d.Open(); err == nil {
			return
		}
	}
	panic("could not obtain a driver.GPU for testing")
}

func TestSemantic(t *testing.T) {
	semantics := map[Semantic]struct {
		i int
		s string
	}{
		Position:  {0, "Position"},
		Normal:    {1, "Normal"},
		Tangent:   {2, "Tangent"},
		TexCoord0: {3, "TexCoord0"},
		TexCoord1: {4, "TexCoord1"},
		Color0:    {5, "Color0"},
		Joints0:   {6, "Joints0"},
		Weights0:  {7, "Weights0"},
	}
	if x := len(semantics); x != MaxSemantic {
		t.Fatalf("MaxSemantic:\nhave %d\nwant %d", MaxSemantic, x)
	}
	// The I values are used in shader code.
	for k, v := range semantics {
		if i := k.I(); i != v.i {
			t.Fatalf("Semantic.I: %s\nhave: %d\nwant %d", v.s, i, v.i)
		}
	}
	s := fmt.Sprintf("A mesh can have up to %d semantics, whose IDs are:", MaxSemantic)
	for _, v := range semantics {
		s += fmt.Sprintf("\n\t%s: %d", v.s, v.i)
	}
	t.Log(s)
}

func TestSetBuffer(t *testing.T) {
	SetBuffer(nil)
	if storage.buf != nil {
		t.Fatalf("SetBuffer: storage.buf\nhave %v\nwant nil", storage.buf)
	}
	if x := storage.spanMap.Len(); x != 0 {
		t.Fatalf("SetBuffer: storage.spanMap.Len\nhave %d\nwant 0", x)
	}
	if x := len(storage.prims); x != 0 {
		t.Fatalf("SetBuffer: len(storage.prims)\nhave %d\nwant 0", x)
	}
	// Set to non-nil.
	var prev driver.Buffer
	for _, s := range [...]int64{16384, 32768, 1048576, 16777216 + 16384} {
		buf, err := gpu.NewBuffer(s, true, driver.UVertexData|driver.UIndexData)
		if err != nil {
			panic("could not create a driver.Buffer for testing")
		}
		if x := SetBuffer(buf); x != prev {
			t.Fatalf("SetBuffer: storage.buf\nhave %v\nwant %v", x, prev)
		} else {
			if x != nil {
				x.Destroy()
			}
			prev = buf
		}
		if storage.buf != buf {
			t.Fatalf("SetBuffer: storage.buf\nhave %v\nwant %v", storage.buf, buf)
		}
		n := storage.spanMap.Len()
		if x := s / blockSize; int(x) != n {
			t.Fatalf("SetBuffer: storage.spanMap.Len\nhave %d\nwant %d", n, x)
		}
		if x := len(storage.prims); x != 0 {
			t.Fatalf("SetBuffer: len(storage.prims)\nhave %d\nwant 0", x)
		}
	}
	// Set to nil again.
	if x := SetBuffer(nil); x != prev {
		t.Fatalf("SetBuffer: storage.buf\nhave %v\nwant %v", x, prev)
	} else {
		x.Destroy()
	}
	if storage.buf != nil {
		t.Fatalf("SetBuffer: storage.buf\nhave %v\nwant nil", storage.buf)
	}
	if x := storage.spanMap.Len(); x != 0 {
		t.Fatalf("SetBuffer: storage.spanMap.Len\nhave %d\nwant 0", x)
	}
	if x := len(storage.prims); x != 0 {
		t.Fatalf("SetBuffer: len(storage.prims)\nhave %d\nwant 0", x)
	}
}
