// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package mesh

import (
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
