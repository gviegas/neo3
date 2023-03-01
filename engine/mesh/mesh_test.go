// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
)

var gpu = ctxt.GPU()

func TestSemantic(t *testing.T) {
	semantics := map[Semantic]int{
		Position:  0,
		Normal:    1,
		Tangent:   2,
		TexCoord0: 3,
		TexCoord1: 4,
		Color0:    5,
		Joints0:   6,
		Weights0:  7,
	}
	if x := len(semantics); x != MaxSemantic {
		t.Fatalf("MaxSemantic:\nhave %d\nwant %d", MaxSemantic, x)
	}
	// The I values are used in shader code.
	for k, v := range semantics {
		if i := k.I(); i != v {
			t.Fatalf("Semantic.I: %s\nhave: %d\nwant %d", k, i, v)
		}
	}
	s := fmt.Sprintf("A mesh can have up to %d semantics, whose IDs are:", MaxSemantic)
	for k, v := range semantics {
		s += fmt.Sprintf("\n\t%s: %d", k, v)
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

func TestConv(t *testing.T) {
	const n = 4096 * 16
	f32 := make([]float32, n)
	u16 := make([]uint16, n)
	u8 := make([]uint8, n)
	for i := 0; i < n; i++ {
		switch i % 5 {
		case 0:
			f32[i] = 1
			u16[i] = 65535
			u8[i] = 255
		case 1:
			f32[i] = 0.5
			u16[i] = 32768
			u8[i] = 128
		case 2:
			f32[i] = 0.25
			u16[i] = 16384
			u8[i] = 64
		case 3:
			f32[i] = 0.125
			u16[i] = 8192
			u8[i] = 32
		case 4:
			f32[i] = 0
			u16[i] = 0
			u8[i] = 0
		}
	}
	b32 := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(f32))), n*4)
	r32 := bytes.NewReader(b32)
	b16 := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(u16))), n*2)
	r16 := bytes.NewReader(b16)
	r8 := bytes.NewReader(u8)

	// No conversion needed.
	for i := 0; i < MaxSemantic; i++ {
		sem := Semantic(1 << i)
		r, err := sem.conv(sem.format(), r8, n/sem.format().Size())
		if r != r8 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
	}

	// Not convertible.
	var finval = [...]driver.VertexFmt{
		driver.Int8, driver.Int8x2, driver.Int8x3, driver.Int8x4,
		driver.Int16, driver.Int16x2, driver.Int16x3, driver.Int16x4,
		driver.Int32, driver.Int32x2, driver.Int32x3, driver.Int32x4,
		driver.Uint8,
		driver.Uint16,
		driver.Uint32, driver.Uint32x2, driver.Uint32x3, driver.Uint32x4,
		driver.Float32,
	}
	var fmts [MaxSemantic][]driver.VertexFmt
	fmts[Position.I()] = append(append([]driver.VertexFmt{},
		driver.Uint8x2, driver.Uint8x3, driver.Uint8x4,
		driver.Uint16x2, driver.Uint16x3, driver.Uint16x4,
		driver.Float32x2, driver.Float32x4,
	), finval[:]...)
	fmts[Normal.I()] = append([]driver.VertexFmt{}, fmts[Position.I()]...)
	fmts[Tangent.I()] = append(append([]driver.VertexFmt{},
		driver.Uint8x2, driver.Uint8x3, driver.Uint8x4,
		driver.Uint16x2, driver.Uint16x3, driver.Uint16x4,
		driver.Float32x2, driver.Float32x3,
	), finval[:]...)
	fmts[TexCoord0.I()] = append(append([]driver.VertexFmt{},
		driver.Uint8x3, driver.Uint8x4,
		driver.Uint16x3, driver.Uint16x4,
		driver.Float32x3, driver.Float32x4,
	), finval[:]...)
	fmts[TexCoord1.I()] = append([]driver.VertexFmt{}, fmts[TexCoord0.I()]...)
	fmts[Color0.I()] = append(append([]driver.VertexFmt{},
		driver.Uint8x2,
		driver.Uint16x2,
		driver.Float32x2,
	), finval[:]...)
	fmts[Joints0.I()] = append(append([]driver.VertexFmt{},
		driver.Uint8x2, driver.Uint8x3,
		driver.Uint16x2, driver.Uint16x3,
		driver.Float32x2, driver.Float32x3, driver.Float32x4,
	), finval[:]...)
	fmts[Weights0.I()] = append(append([]driver.VertexFmt{},
		driver.Uint8x2, driver.Uint8x3,
		driver.Uint16x2, driver.Uint16x3,
		driver.Float32x2, driver.Float32x3,
	), finval[:]...)
	for i := range fmts {
		sem := Semantic(1 << i)
		for _, f := range fmts[i] {
			r, err := sem.conv(f, r8, n/f.Size())
			if r != nil || err == nil {
				t.Fatalf("%s.conv: unexpected result: (%v, nil)", sem, r)
			}
			if !strings.HasPrefix(err.Error(), prefix) {
				t.Fatalf("%s.conv: unexpected error: %#v", sem, err)
			}
			r8.Seek(0, io.SeekStart)
		}
	}

	var dst [16]byte
	cnts := [...]int{1, 2, 3, 4, 6, 9, 12, 24, 36, n / 900, n / 512, n / 256, n / 128, n / 64, n / 32, n / 16}
	convUn16 := func(x uint16) float32 { return float32(x) / 65535 }
	convUn8 := func(x uint8) float32 { return float32(x) / 255 }

	// TexCoord0 and TexCoord1.
	for _, cnt := range cnts {
		for _, sem := range [2]Semantic{TexCoord0, TexCoord1} {
			r16.Seek(0, io.SeekStart)
			r, err := sem.conv(driver.Uint16x2, r16, cnt)
			if r == r16 || err != nil {
				t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
			}
			for i := 0; i < cnt; i++ {
				if n, _ := r.Read(dst[:8]); n != 8 {
					t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 8", sem, n)
				}
				u := *(*float32)(unsafe.Pointer(&dst))
				v := *(*float32)(unsafe.Pointer(&dst[4]))
				x := convUn16(u16[i*2])
				y := convUn16(u16[i*2+1])
				if x != u || y != v {
					t.Fatalf("%s.conv: bad conversion:\nhave %f, %f\nwant %f, %f", sem, u, v, x, y)
				}
			}

			r8.Seek(0, io.SeekStart)
			r, err = sem.conv(driver.Uint8x2, r8, cnt)
			if r == r8 || err != nil {
				t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
			}
			for i := 0; i < cnt; i++ {
				if n, _ := r.Read(dst[:8]); n != 8 {
					t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 8", sem, n)
				}
				u := *(*float32)(unsafe.Pointer(&dst))
				v := *(*float32)(unsafe.Pointer(&dst[4]))
				x := convUn8(u8[i*2])
				y := convUn8(u8[i*2+1])
				if x != u || y != v {
					t.Fatalf("%s.conv: bad conversion:\nhave %f, %f\nwant %f, %f", sem, u, v, x, y)
				}
			}
		}
	}

	// Color0.
	for _, cnt := range cnts {
		sem := Color0
		r32.Seek(0, io.SeekStart)
		r, err := sem.conv(driver.Float32x3, r32, cnt)
		if r == r32 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			r := *(*float32)(unsafe.Pointer(&dst))
			g := *(*float32)(unsafe.Pointer(&dst[4]))
			b := *(*float32)(unsafe.Pointer(&dst[8]))
			a := *(*float32)(unsafe.Pointer(&dst[12]))
			x := f32[i*3]
			y := f32[i*3+1]
			z := f32[i*3+2]
			w := float32(1)
			if x != r || y != g || z != b || w != a {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, r, g, b, a, x, y, z, w)
			}
		}

		r16.Seek(0, io.SeekStart)
		r, err = sem.conv(driver.Uint16x4, r16, cnt)
		if r == r16 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:16]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			r := *(*float32)(unsafe.Pointer(&dst))
			g := *(*float32)(unsafe.Pointer(&dst[4]))
			b := *(*float32)(unsafe.Pointer(&dst[8]))
			a := *(*float32)(unsafe.Pointer(&dst[12]))
			x := convUn16(u16[i*4])
			y := convUn16(u16[i*4+1])
			z := convUn16(u16[i*4+2])
			w := convUn16(u16[i*4+3])
			if x != r || y != g || z != b || w != a {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, r, g, b, a, x, y, z, w)
			}
		}

		r16.Seek(0, io.SeekStart)
		r, err = sem.conv(driver.Uint16x3, r16, cnt)
		if r == r16 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:16]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			r := *(*float32)(unsafe.Pointer(&dst))
			g := *(*float32)(unsafe.Pointer(&dst[4]))
			b := *(*float32)(unsafe.Pointer(&dst[8]))
			a := *(*float32)(unsafe.Pointer(&dst[12]))
			x := convUn16(u16[i*3])
			y := convUn16(u16[i*3+1])
			z := convUn16(u16[i*3+2])
			w := float32(1)
			if x != r || y != g || z != b || w != a {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, r, g, b, a, x, y, z, w)
			}
		}

		r8.Seek(0, io.SeekStart)
		r, err = sem.conv(driver.Uint8x4, r8, cnt)
		if r == r8 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:16]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			r := *(*float32)(unsafe.Pointer(&dst))
			g := *(*float32)(unsafe.Pointer(&dst[4]))
			b := *(*float32)(unsafe.Pointer(&dst[8]))
			a := *(*float32)(unsafe.Pointer(&dst[12]))
			x := convUn8(u8[i*4])
			y := convUn8(u8[i*4+1])
			z := convUn8(u8[i*4+2])
			w := convUn8(u8[i*4+3])
			if x != r || y != g || z != b || w != a {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, r, g, b, a, x, y, z, w)
			}
		}

		r8.Seek(0, io.SeekStart)
		r, err = sem.conv(driver.Uint8x3, r8, cnt)
		if r == r8 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:16]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			r := *(*float32)(unsafe.Pointer(&dst))
			g := *(*float32)(unsafe.Pointer(&dst[4]))
			b := *(*float32)(unsafe.Pointer(&dst[8]))
			a := *(*float32)(unsafe.Pointer(&dst[12]))
			x := convUn8(u8[i*3])
			y := convUn8(u8[i*3+1])
			z := convUn8(u8[i*3+2])
			w := float32(1)
			if x != r || y != g || z != b || w != a {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, r, g, b, a, x, y, z, w)
			}
		}
	}

	// Joints0.
	for _, cnt := range cnts {
		sem := Joints0
		r8.Seek(0, io.SeekStart)
		r, err := sem.conv(driver.Uint8x4, r8, cnt)
		if r == r8 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:8]); n != 8 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 8", sem, n)
			}
			a := *(*uint16)(unsafe.Pointer(&dst))
			b := *(*uint16)(unsafe.Pointer(&dst[2]))
			c := *(*uint16)(unsafe.Pointer(&dst[4]))
			d := *(*uint16)(unsafe.Pointer(&dst[6]))
			x := uint16(u8[i*4])
			y := uint16(u8[i*4+1])
			z := uint16(u8[i*4+2])
			w := uint16(u8[i*4+3])
			if x != a || y != b || z != c || w != d {
				t.Fatalf("%s.conv: bad conversion:\nhave %d, %d, %d, %d\nwant %d, %d, %d, %d", sem, a, b, c, d, x, y, z, w)
			}
		}
	}

	// Weights0.
	for _, cnt := range cnts {
		sem := Weights0
		r16.Seek(0, io.SeekStart)
		r, err := sem.conv(driver.Uint16x4, r16, cnt)
		if r == r16 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:16]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			a := *(*float32)(unsafe.Pointer(&dst))
			b := *(*float32)(unsafe.Pointer(&dst[4]))
			c := *(*float32)(unsafe.Pointer(&dst[8]))
			d := *(*float32)(unsafe.Pointer(&dst[12]))
			x := convUn16(u16[i*4])
			y := convUn16(u16[i*4+1])
			z := convUn16(u16[i*4+2])
			w := convUn16(u16[i*4+3])
			if x != a || y != b || z != c || w != d {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, a, b, c, d, x, y, z, w)
			}
		}

		r8.Seek(0, io.SeekStart)
		r, err = sem.conv(driver.Uint8x4, r8, cnt)
		if r == r8 || err != nil {
			t.Fatalf("%s.conv: unexpected result: (%v, %v)", sem, r, err)
		}
		for i := 0; i < cnt; i++ {
			if n, _ := r.Read(dst[:16]); n != 16 {
				t.Fatalf("%s.conv: unexpected read count:\nhave %d\nwant 16", sem, n)
			}
			a := *(*float32)(unsafe.Pointer(&dst))
			b := *(*float32)(unsafe.Pointer(&dst[4]))
			c := *(*float32)(unsafe.Pointer(&dst[8]))
			d := *(*float32)(unsafe.Pointer(&dst[12]))
			x := convUn8(u8[i*4])
			y := convUn8(u8[i*4+1])
			z := convUn8(u8[i*4+2])
			w := convUn8(u8[i*4+3])
			if x != a || y != b || z != c || w != d {
				t.Fatalf("%s.conv: bad conversion:\nhave %f, %f, %f, %f\nwant %f, %f, %f, %f", sem, a, b, c, d, x, y, z, w)
			}
		}
	}
}
