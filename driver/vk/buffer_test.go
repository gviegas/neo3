// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"gviegas/neo3/driver"
)

func TestBuffer(t *testing.T) {
	cases := [...]struct {
		size    int64
		visible bool
		usage   driver.Usage
	}{
		{8192, true, driver.UShaderRead | driver.UShaderWrite | driver.UShaderConst | driver.UVertexData | driver.UIndexData},
		{512, true, 0},
		{16, true, driver.UShaderRead | driver.UShaderWrite},
		{1 << 20, false, driver.UGeneric},
		{1 << 20, false, driver.UShaderConst | driver.UVertexData | driver.UIndexData},
		{1 << 20, true, driver.UVertexData | driver.UIndexData},
		{100 << 20, true, driver.UGeneric},
		{1 << 62, true, 0},
		{1, true, driver.UGeneric},
		//{0, true, 0},
	}
	zb := buffer{}
	zm := memory{}
	for _, c := range cases {
		call := fmt.Sprintf("tDrv.NewBuffer(%d, %t, %d)", c.size, c.visible, c.usage)
		// NewBuffer.
		if b, err := tDrv.NewBuffer(c.size, c.visible, c.usage); err == nil {
			if b == nil {
				t.Fatalf("%s\nhave nil, nil\nwant non-nil, nil", call)
				continue
			}
			b := b.(*buffer)
			if b.m != nil {
				if b.m.d != &tDrv {
					t.Fatalf("%s: b.m.d\nhave %p\nwant %p", call, b.m.d, &tDrv)
				}
				// The size can be greater than what was requested.
				if b.m.size < c.size {
					t.Fatalf("%s: b.m.size\nhave %d\nwant at least %d", call, b.m.size, c.size)
				}
				if b.m.vis {
					if int64(len(b.m.p)) != b.m.size {
						t.Fatalf("%s: len(b.m.p)\nhave %d\nwant %d", call, len(b.m.p), b.m.size)
					}
				} else {
					// Private memory is optional, shared memory is not.
					if c.visible {
						t.Fatalf("%s: b.m.vis\nhave false\nwant true", call)
					}
					if len(b.m.p) != 0 {
						t.Fatalf("%s: len(b.m.p)\nhave %d\nwant 0", call, len(b.m.p))
					}
				}
				// NewBuffer should bind the memory and set this to true.
				if !b.m.bound {
					t.Fatalf("%s: b.m.bound\nhave false\nwant true", call)
				}
				if b.m.mem == zm.mem {
					t.Fatalf("%s: b.m.mem\nhave %v\nwant valid handle", call, b.m.mem)
				}
				if b.m.typ < 0 || b.m.typ >= int(tDrv.mprop.memoryTypeCount) {
					t.Fatalf("%s: b.m.typ\nhave %d\nwant valid index", call, b.m.typ)
				} else {
					heap := int(tDrv.mprop.memoryTypes[b.m.typ].heapIndex)
					if b.m.heap != heap {
						t.Fatalf("%s: b.m.heap\nhave %d\nwant %d", call, b.m.heap, heap)
					}
				}
				// Bytes.
				p := b.Bytes()
				if b.m.vis {
					if int64(len(p)) != b.m.size {
						t.Fatalf("b.Bytes(): len(p)\nhave %d\nwant %d", len(p), b.m.size)
					}
					q := b.Bytes()
					if (*[0]byte)(p) != (*[0]byte)(q) {
						t.Fatalf("b.Bytes()\nhave %p\nwant %p", (*[0]byte)(q), (*[0]byte)(p))
					}
				} else if len(p) != 0 {
					t.Fatalf("b.Bytes(): len(p)\nhave %d\nwant 0", len(p))
				}
				// Cap.
				if n := b.Cap(); n != b.m.size {
					t.Fatalf("b.Cap()\nhave %d\nwant %d", n, b.m.size)
				}
			} else {
				t.Fatalf("%s: b.m\nhave nil\nwant non-nil", call)
			}
			if b.buf == zb.buf {
				t.Fatalf("%s: b.buf\nhave %v\nwant valid handle", call, b.buf)
			}
			// Destroy.
			b.Destroy()
			if *b != zb {
				t.Fatalf("b.Destroy(): b\nhave %v\nwant %v", b, zb)
			}
		} else if b != nil {
			t.Fatalf("%s\nhave %p, %v\nwant nil, %v", call, b, err, err)
		} else {
			t.Logf("(error) %s: %v", call, err)
		}
	}
}
