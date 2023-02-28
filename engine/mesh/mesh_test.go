// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
)

var gpu = ctxt.GPU()

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
	if x := storage.primMap.Len(); x != 0 {
		t.Fatalf("SetBuffer: storage.primMap.Len\nhave %d\nwant 0", x)
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
		if x := storage.primMap.Len(); x != 0 {
			t.Fatalf("SetBuffer: storage.primMap.Len\nhave %d\nwant 0", x)
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
	if x := storage.primMap.Len(); x != 0 {
		t.Fatalf("SetBuffer: storage.primMap.Len\nhave %d\nwant 0", x)
	}
	if x := len(storage.prims); x != 0 {
		t.Fatalf("SetBuffer: len(storage.prims)\nhave %d\nwant 0", x)
	}
}

func TestSpan(t *testing.T) {
	type want struct{ bstart, bend, blen int }
	check := func(s span, w want) {
		if x := s.byteStart(); x != w.bstart {
			t.Fatalf("span.byteStart:\nhave %d\nwant %d", x, w.bstart)
		}
		if x := s.byteEnd(); x != w.bend {
			t.Fatalf("span.byteEnd:\nhave %d\nwant %d", x, w.bend)
		}
		if x := s.byteLen(); x != w.blen {
			t.Fatalf("span.byteLen:\nhave %d\nwant %d", x, w.blen)
		}
	}
	var s span
	check(s, want{})
	s.end++
	check(s, want{0, blockSize, blockSize})
	s.end++
	check(s, want{0, blockSize * 2, blockSize * 2})
	s.start++
	check(s, want{blockSize, blockSize * 2, blockSize})
	var p primitive
	for i := range p.vertex {
		check(p.vertex[i].span, want{})
	}
	check(p.index.span, want{})
	p.vertex[Normal.I()].span = s
	p.vertex[Normal.I()].start++
	p.vertex[Normal.I()].end += 3
	check(p.vertex[Normal.I()].span, want{blockSize * 2, blockSize * 5, blockSize * 3})
}

func TestStore(t *testing.T) {
	var b meshBuffer

	newSrc := func(byteLen int, mark byte) io.ReadSeeker {
		data := make([]byte, byteLen)
		for i := range data {
			data[i] = mark
		}
		return bytes.NewReader(data)
	}

	check := func(s span, err error, byteLen int, mark byte) {
		if x, y := b.buf.Cap(), int64(b.spanMap.Len())*blockSize; x < y {
			t.Fatalf("meshBuffer.store: buf.Cap() < spanMap.Len()*blockSize: %d/%d", x, y)
		} else if x != y {
			t.Logf("[!] meshBuffer.store: buf.Cap() != spanMap.Len()*blockSize: %d/%d", x, y)
		}
		if s == (span{}) || err != nil {
			t.Fatalf("meshBuffer.store: unexpected result: (%v, %v)", s, err)
		}
		if x := s.byteLen(); x < byteLen {
			t.Fatalf("meshBuffer.store: span.byteLen()\nhave %d\nwant >= %d", x, byteLen)
		}
		for i := s.byteStart(); i < s.byteStart()+byteLen; i++ {
			if x := b.buf.Bytes()[i]; x != mark {
				t.Fatalf("meshBuffer.store: unexpected byte value at index %d\nhave %#v\nwant %#v", i, x, mark)
			}
		}
		for i := s.byteStart() + byteLen; i < s.byteEnd(); i++ {
			if x := b.buf.Bytes()[i]; x != 0 {
				t.Fatalf("meshBuffer.store: unexpected byte value at index %d\nhave %#v\nwant 0x00", i, x)
			}
		}
	}

	var spans [20]span
	var acc int
	for i, x := range [20]int{
		1,
		2,
		16,
		2049,
		2048,
		2047,
		1 << 20,
		4<<20 - 1,
		1 << 16,
		3<<20 + 1,
		blockSize * spanMapNBit,
		blockSize*spanMapNBit - 1,
		blockSize*spanMapNBit + 1,
		16 * blockSize * spanMapNBit,
		4*blockSize*spanMapNBit + 3,
		9*blockSize*spanMapNBit - 6,
		blockSize,
		blockSize * 2,
		blockSize + 1,
		blockSize - 1,
	} {
		s, err := b.store(newSrc(x, byte(i+1)), x)
		check(s, err, x, byte(i+1))
		spans[i] = s
		acc += x
	}

	slen := b.spanMap.Len()
	srem := b.spanMap.Rem()
	bcap := int(b.buf.Cap())
	t.Logf("total requested size: %d bytes", acc)
	t.Logf("spans:\n%v", spans)
	t.Logf("span map length: %d (%d bytes)", slen, slen*blockSize)
	t.Logf("span map remaining: %d (%d bytes)", srem, srem*blockSize)
	t.Logf("buffer utilization: %.3f%% (%d bytes of %d are unused)", float64(acc)/float64(bcap)*100, bcap-acc, bcap)

	b.buf.Destroy()
}
