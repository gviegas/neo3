// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
)

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

func TestSetMeshBuffer(t *testing.T) {
	setMeshBuffer(nil)
	if storage.buf != nil {
		t.Fatalf("setMeshBuffer: storage.buf\nhave %v\nwant nil", storage.buf)
	}
	if x := storage.spanMap.Len(); x != 0 {
		t.Fatalf("setMeshBuffer: storage.spanMap.Len\nhave %d\nwant 0", x)
	}
	if x := storage.primMap.Len(); x != 0 {
		t.Fatalf("setMeshBuffer: storage.primMap.Len\nhave %d\nwant 0", x)
	}
	if x := len(storage.prims); x != 0 {
		t.Fatalf("setMeshBuffer: len(storage.prims)\nhave %d\nwant 0", x)
	}
	// Set to non-nil.
	var prev driver.Buffer
	for _, s := range [...]int64{16384, 32768, 1048576, 16777216 + 16384} {
		buf, err := ctxt.GPU().NewBuffer(s, true, driver.UVertexData|driver.UIndexData)
		if err != nil {
			panic("could not create a driver.Buffer for testing")
		}
		if x := setMeshBuffer(buf); x != prev {
			t.Fatalf("setMeshBuffer: storage.buf\nhave %v\nwant %v", x, prev)
		} else {
			if x != nil {
				x.Destroy()
			}
			prev = buf
		}
		if storage.buf != buf {
			t.Fatalf("setMeshBuffer: storage.buf\nhave %v\nwant %v", storage.buf, buf)
		}
		n := storage.spanMap.Len()
		if x := s / spanBlock; int(x) != n {
			t.Fatalf("setMeshBuffer: storage.spanMap.Len\nhave %d\nwant %d", n, x)
		}
		if x := storage.primMap.Len(); x != 0 {
			t.Fatalf("setMeshBuffer: storage.primMap.Len\nhave %d\nwant 0", x)
		}
		if x := len(storage.prims); x != 0 {
			t.Fatalf("setMeshBuffer: len(storage.prims)\nhave %d\nwant 0", x)
		}
	}
	// Set to nil again.
	if x := setMeshBuffer(nil); x != prev {
		t.Fatalf("setMeshBuffer: storage.buf\nhave %v\nwant %v", x, prev)
	} else {
		x.Destroy()
	}
	if storage.buf != nil {
		t.Fatalf("setMeshBuffer: storage.buf\nhave %v\nwant nil", storage.buf)
	}
	if x := storage.spanMap.Len(); x != 0 {
		t.Fatalf("setMeshBuffer: storage.spanMap.Len\nhave %d\nwant 0", x)
	}
	if x := storage.primMap.Len(); x != 0 {
		t.Fatalf("setMeshBuffer: storage.primMap.Len\nhave %d\nwant 0", x)
	}
	if x := len(storage.prims); x != 0 {
		t.Fatalf("setMeshBuffer: len(storage.prims)\nhave %d\nwant 0", x)
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
	check(s, want{0, spanBlock, spanBlock})
	s.end++
	check(s, want{0, spanBlock * 2, spanBlock * 2})
	s.start++
	check(s, want{spanBlock, spanBlock * 2, spanBlock})
	var p primitive
	for i := range p.vertex {
		check(p.vertex[i].span, want{})
	}
	check(p.index.span, want{})
	p.vertex[Normal.I()].span = s
	p.vertex[Normal.I()].start++
	p.vertex[Normal.I()].end += 3
	check(p.vertex[Normal.I()].span, want{spanBlock * 2, spanBlock * 5, spanBlock * 3})
}

func TestStorage(t *testing.T) {
	var b meshBuffer

	newSrc := func(byteLen int, mark byte) io.ReadSeeker {
		data := make([]byte, byteLen)
		for i := range data {
			data[i] = mark
		}
		return bytes.NewReader(data)
	}

	check := func(s span, err error, byteLen int, mark byte) {
		if x, y := b.buf.Cap(), int64(b.spanMap.Len())*spanBlock; x < y {
			t.Fatalf("meshBuffer.store: buf.Cap() < spanMap.Len()*spanBlock: %d/%d", x, y)
		} else if x != y {
			t.Logf("[!] meshBuffer.store: buf.Cap() != spanMap.Len()*spanBlock: %d/%d", x, y)
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
		spanBlock * spanMapNBit,
		spanBlock*spanMapNBit - 1,
		spanBlock*spanMapNBit + 1,
		16 * spanBlock * spanMapNBit,
		4*spanBlock*spanMapNBit + 3,
		9*spanBlock*spanMapNBit - 6,
		spanBlock,
		spanBlock * 2,
		spanBlock + 1,
		spanBlock - 1,
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
	t.Logf("span map length: %d (%d bytes)", slen, slen*spanBlock)
	t.Logf("span map remaining: %d (%d bytes)", srem, srem*spanBlock)
	t.Logf("buffer utilization: %.3f%% (%d bytes of %d are unused)", float64(acc)/float64(bcap)*100, bcap-acc, bcap)

	b.buf.Destroy()
}

func TestSemanticConv(t *testing.T) {
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
			if !strings.HasPrefix(err.Error(), meshPrefix) {
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

func dummyData1(ntris int) MeshData {
	p := PrimitiveData{
		Topology:     driver.TTriangle,
		VertexCount:  ntris * 3,
		SemanticMask: Position | Normal | TexCoord0,
	}
	srcs := make([]io.ReadSeeker, 3)
	for i, s := range [3]Semantic{Position, Normal, TexCoord0} {
		p.Semantics[s.I()] = SemanticData{
			Format: s.format(),
			Offset: 0,
			Src:    i,
		}
		d := make([]byte, p.VertexCount*s.format().Size())
		fillDummySem(s, d)
		srcs[i] = bytes.NewReader(d)
	}
	return MeshData{[]PrimitiveData{p}, srcs}
}

func checkDummyData1(m *Mesh, ntris int, t *testing.T) {
	if x := m.Len(); x != 1 {
		t.Fatalf("Mesh.Len:\nhave %d\nwant 1", x)
	}
	p := storage.prims[m.primIdx]
	if p.topology != driver.TTriangle {
		t.Fatalf("storage.prims[%d].topology:\nhave %v\nwant %v", m.primIdx, p.topology, driver.TTriangle)
	}
	if x := ntris * 3; p.count != x {
		t.Fatalf("storage.prims[%d].count:\nhave %d\nwant %d", m.primIdx, p.count, x)
	}
	if x := Position | Normal | TexCoord0; p.mask != x {
		t.Fatalf("storage.prims[%d].mask:\nhave %x\nwant %x", m.primIdx, p.mask, x)
	}
	if p.next >= 0 {
		t.Fatalf("storage.prims[%d].next:\nhave %d\nwant < 0", m.primIdx, p.next)
	}
	b := storage.buf.Bytes()
	for _, s := range [3]Semantic{Position, Normal, TexCoord0} {
		n := p.vertex[s.I()].format.Size() * ntris * 3
		spn := p.vertex[s.I()].span
		if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
			t.Fatalf("storage.prims[%d].vertex[%s.I()].span: end - start\nhave %d\nwant %d", m.primIdx, s, x, y)
		}
		x := ^byte(s.I())
		for i := spn.byteStart(); i < spn.byteStart()+n; i++ {
			if b[i] != x {
				t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
			}
		}
	}
}

func dummyData2(ntris int) MeshData {
	p := PrimitiveData{
		Topology:     driver.TTriangle,
		VertexCount:  ntris * 3,
		IndexCount:   ntris * 6,
		SemanticMask: Position | Color0,
	}
	srcs := make([]io.ReadSeeker, 2)

	p.Semantics[Color0.I()] = SemanticData{
		Format: driver.Float32x3,
		Offset: 0,
		Src:    0,
	}
	p.Semantics[Position.I()] = SemanticData{
		Format: Position.format(),
		Offset: int64(p.VertexCount * p.Semantics[Color0.I()].Format.Size()),
		Src:    0,
	}
	dlen := p.VertexCount * (p.Semantics[Color0.I()].Format.Size() + p.Semantics[Position.I()].Format.Size())
	d := make([]byte, dlen)
	fillDummySem(Color0, d[:p.Semantics[Position.I()].Offset])
	fillDummySem(Position, d[p.Semantics[Position.I()].Offset:])
	srcs[0] = bytes.NewReader(d)

	p.Index = IndexData{
		Format: driver.Index16,
		Offset: 0,
		Src:    1,
	}
	dlen = p.IndexCount * 2
	d = make([]byte, dlen)
	fillDummyIdx(p.Index.Format, d)
	srcs[1] = bytes.NewReader(d)

	return MeshData{[]PrimitiveData{p}, srcs}
}

func checkDummyData2(m *Mesh, ntris int, t *testing.T) {
	if x := m.Len(); x != 1 {
		t.Fatalf("Mesh.Len:\nhave %d\nwant 1", x)
	}
	p := storage.prims[m.primIdx]
	if p.topology != driver.TTriangle {
		t.Fatalf("storage.prims[%d].topology:\nhave %v\nwant %v", m.primIdx, p.topology, driver.TTriangle)
	}
	if x := ntris * 6; p.count != x {
		t.Fatalf("storage.prims[%d].count:\nhave %d\nwant %d", m.primIdx, p.count, x)
	}
	if x := Position | Color0; p.mask != x {
		t.Fatalf("storage.prims[%d].mask:\nhave %x\nwant %x", m.primIdx, p.mask, x)
	}
	if p.index.format != driver.Index16 {
		t.Fatalf("storage.prims[%d].next:\nhave %v\nwant %v", m.primIdx, p.index.format, driver.Index16)
	}
	if p.next >= 0 {
		t.Fatalf("storage.prims[%d].next:\nhave %d\nwant < 0", m.primIdx, p.next)
	}
	b := storage.buf.Bytes()

	s := Position
	n := p.vertex[s.I()].format.Size() * ntris * 3
	spn := p.vertex[s.I()].span
	if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
		t.Fatalf("storage.prims[%d].vertex[%s.I()].span: end - start\nhave %d\nwant %d", m.primIdx, s, x, y)
	}
	x := ^byte(s.I())
	for i := spn.byteStart(); i < spn.byteStart()+n; i++ {
		if b[i] != x {
			t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
		}
	}

	s = Color0
	n = p.vertex[s.I()].format.Size() * ntris * 3
	spn = p.vertex[s.I()].span
	if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
		t.Fatalf("storage.prims[%d].vertex[%s.I()].span: end - start\nhave %d\nwant %d", m.primIdx, s, x, y)
	}
	x = ^byte(s.I())
	for i := spn.byteStart(); i < spn.byteStart()+n; i += 16 {
		for j := i; j < 12; j++ {
			if b[j] != x {
				t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
			}
		}
		if f := *(*float32)(unsafe.Pointer(unsafe.SliceData(b[i+12:]))); f != 1 {
			t.Fatalf("storage.buf.Bytes()[%d:%d]:\nhave %f\nwant float32(1)", i+12, i+16, f)
		}
	}

	n = 2 * ntris * 6
	spn = p.index.span
	if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
		t.Fatalf("storage.prims[%d].index.span: end - start\nhave %d\nwant %d", m.primIdx, x, y)
	}
	x = byte(p.index.format + 1)
	for i := spn.byteStart(); i < spn.byteStart()+n; i++ {
		if b[i] != x {
			t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
		}
	}
}

func dummyData3(ntris int) MeshData {
	p := PrimitiveData{
		Topology:     driver.TTriangle,
		VertexCount:  ntris * 3,
		IndexCount:   ntris*6 + 15,
		SemanticMask: Position | Normal | Tangent | TexCoord0 | TexCoord1 | Color0 | Joints0 | Weights0,
	}

	var off int64
	for _, s := range [8]Semantic{
		Normal,
		Tangent,
		Color0,
		TexCoord1,
		Weights0,
		Position,
		TexCoord0,
		Joints0,
	} {
		p.Semantics[s.I()] = SemanticData{
			Format: s.format(),
			Offset: off,
			Src:    0,
		}
		off += int64(p.VertexCount * s.format().Size())
	}
	p.Index = IndexData{
		Format: driver.Index32,
		Offset: off,
		Src:    0,
	}

	dlen := int(off) + p.IndexCount*4
	d := make([]byte, dlen)
	fillDummyIdx(p.Index.Format, d[off:])
	for i, x := range p.Semantics {
		sz := int64(x.Format.Size() * p.VertexCount)
		fillDummySem(Semantic(1<<i), d[x.Offset:x.Offset+sz])
	}

	return MeshData{[]PrimitiveData{p}, []io.ReadSeeker{bytes.NewReader(d)}}
}

func checkDummyData3(m *Mesh, ntris int, t *testing.T) {
	if x := m.Len(); x != 1 {
		t.Fatalf("Mesh.Len:\nhave %d\nwant 1", x)
	}
	p := storage.prims[m.primIdx]
	if p.topology != driver.TTriangle {
		t.Fatalf("storage.prims[%d].topology:\nhave %v\nwant %v", m.primIdx, p.topology, driver.TTriangle)
	}
	if x := ntris*6 + 15; p.count != x {
		t.Fatalf("storage.prims[%d].count:\nhave %d\nwant %d", m.primIdx, p.count, x)
	}
	if x := Position | Normal | Tangent | TexCoord0 | TexCoord1 | Color0 | Joints0 | Weights0; p.mask != x {
		t.Fatalf("storage.prims[%d].mask:\nhave %x\nwant %x", m.primIdx, p.mask, x)
	}
	if p.index.format != driver.Index32 {
		t.Fatalf("storage.prims[%d].next:\nhave %v\nwant %v", m.primIdx, p.index.format, driver.Index32)
	}
	if p.next >= 0 {
		t.Fatalf("storage.prims[%d].next:\nhave %d\nwant < 0", m.primIdx, p.next)
	}
	b := storage.buf.Bytes()

	for i := 0; i < MaxSemantic; i++ {
		s := Semantic(1 << i)
		n := p.vertex[s.I()].format.Size() * ntris * 3
		spn := p.vertex[s.I()].span
		if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
			t.Fatalf("storage.prims[%d].vertex[%s.I()].span: end - start\nhave %d\nwant %d", m.primIdx, s, x, y)
		}
		x := ^byte(s.I())
		for i := spn.byteStart(); i < spn.byteStart()+n; i++ {
			if b[i] != x {
				t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
			}
		}
	}

	n := 4*ntris*6 + 60
	spn := p.index.span
	if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
		t.Fatalf("storage.prims[%d].index.span: end - start\nhave %d\nwant %d", m.primIdx, x, y)
	}
	x := byte(p.index.format + 1)
	for i := spn.byteStart(); i < spn.byteStart()+n; i++ {
		if b[i] != x {
			t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
		}
	}
}

func dummyData4(ntris int) MeshData {
	var srcs []io.ReadSeeker

	p1 := PrimitiveData{
		Topology:     driver.TTriangle,
		VertexCount:  ntris * 3,
		SemanticMask: Position | TexCoord1,
	}
	for _, s := range [2]Semantic{Position, TexCoord1} {
		p1.Semantics[s.I()] = SemanticData{
			Format: s.format(),
			Offset: 0,
			Src:    len(srcs),
		}
		d := make([]byte, p1.VertexCount*s.format().Size())
		fillDummySem(s, d)
		srcs = append(srcs, bytes.NewReader(d))
	}

	p2 := PrimitiveData{
		Topology:     driver.TTriangle,
		VertexCount:  ntris * 3,
		SemanticMask: Position,
	}
	p2.Semantics[Position.I()] = SemanticData{
		Format: Position.format(),
		Offset: 0,
		Src:    len(srcs),
	}
	d := make([]byte, p2.VertexCount*Position.format().Size())
	fillDummySem(Position, d)
	srcs = append(srcs, bytes.NewReader(d))

	p3 := PrimitiveData{
		Topology:     driver.TTriangle,
		VertexCount:  ntris * 3,
		SemanticMask: Position | Normal | TexCoord0 | TexCoord1,
	}
	for _, s := range [4]Semantic{Position, Normal, TexCoord0, TexCoord1} {
		p3.Semantics[s.I()] = SemanticData{
			Format: s.format(),
			Offset: 0,
			Src:    len(srcs),
		}
		d := make([]byte, p3.VertexCount*s.format().Size())
		fillDummySem(s, d)
		srcs = append(srcs, bytes.NewReader(d))
	}

	return MeshData{[]PrimitiveData{p3, p1, p2}, srcs}
}

func checkDummyData4(m *Mesh, ntris int, t *testing.T) {
	if x := m.Len(); x != 3 {
		t.Fatalf("Mesh.Len:\nhave %d\nwant 3", x)
	}
	ps := [3]*primitive{&storage.prims[m.primIdx]}
	if ps[0].next < 0 {
		t.Fatalf("storage.prims[%d].next:\nhave %d\nwant >= 0", m.primIdx, ps[0].next)
	}
	ps[1] = &storage.prims[ps[0].next]
	if ps[1].next < 0 {
		t.Fatalf("storage.prims[%d].next:\nhave %d\nwant >= 0", ps[0].next, ps[1].next)
	}
	ps[2] = &storage.prims[ps[1].next]
	if ps[2].next >= 0 {
		t.Fatalf("storage.prims[%d].next:\nhave %d\nwant < 0", ps[1].next, ps[2].next)
	}
	ss := [3][]Semantic{
		{Position, Normal, TexCoord0, TexCoord1},
		{Position, TexCoord1},
		{Position},
	}
	is := [3]int{m.primIdx, ps[0].next, ps[1].next}
	b := storage.buf.Bytes()
	for i, p := range ps {
		if p.topology != driver.TTriangle {
			t.Fatalf("storage.prims[%d].topology:\nhave %v\nwant %v", is[i], p.topology, driver.TTriangle)
		}
		if x := ntris * 3; p.count != x {
			t.Fatalf("storage.prims[%d].count:\nhave %d\nwant %d", is[i], p.count, x)
		}
		var mask Semantic
		for _, s := range ss[i] {
			mask |= s
		}
		if p.mask != mask {
			t.Fatalf("storage.prims[%d].mask:\nhave %x\nwant %x", is[1], p.mask, mask)
		}
		for _, s := range ss[i] {
			n := p.vertex[s.I()].format.Size() * ntris * 3
			spn := p.vertex[s.I()].span
			if x, y := spn.end-spn.start, (n+(spanBlock-1))/spanBlock; x != y {
				t.Fatalf("storage.prims[%d].vertex[%s.I()].span: end - start\nhave %d\nwant %d", is[i], s, x, y)
			}
			x := ^byte(s.I())
			for i := spn.byteStart(); i < spn.byteStart()+n; i++ {
				if b[i] != x {
					t.Fatalf("storage.buf.Bytes()[%d]:\nhave %d\nwant %d", i, b[i], x)
				}
			}
		}
	}
}

func fillDummySem(s Semantic, d []byte) {
	x := ^byte(s.I())
	for i := range d {
		d[i] = x
	}
}

func fillDummyIdx(i driver.IndexFmt, d []byte) {
	x := byte(i + 1)
	for i := range d {
		d[i] = x
	}
}

func TestMesh(t *testing.T) {
	defer func() {
		b := setMeshBuffer(nil)
		if b != nil {
			b.Destroy()
		}
	}()
	const n = 20 << 20
	buf, err := ctxt.GPU().NewBuffer(n, true, driver.UVertexData|driver.UIndexData)
	if err == nil {
		setMeshBuffer(buf)
	} else {
		t.Fatalf("ctxt.GPU().NewBuffer: %#v", err)
	}

	cases := [...]struct {
		ntris int
		dummy func(int) MeshData
		check func(*Mesh, int, *testing.T)
	}{
		{1, dummyData1, checkDummyData1},
		{12, dummyData4, checkDummyData4},
		{300, dummyData1, checkDummyData1},
		{99, dummyData2, checkDummyData2},
		{760, dummyData3, checkDummyData3},
		{1024, dummyData1, checkDummyData1},
		{4097, dummyData2, checkDummyData2},
		{4095, dummyData2, checkDummyData2},
		{4096, dummyData4, checkDummyData4},
		{16000, dummyData1, checkDummyData1},
		{1, dummyData3, checkDummyData3},
		{256, dummyData3, checkDummyData3},
		{12500, dummyData3, checkDummyData3},
		{5, dummyData2, checkDummyData2},
		{3, dummyData1, checkDummyData1},
		{21673, dummyData2, checkDummyData2},
		{10181, dummyData4, checkDummyData4},
		{512, dummyData4, checkDummyData4},
		{100, dummyData3, checkDummyData3},
	}
	var res [len(cases)]struct {
		data MeshData
		mesh *Mesh
	}
	for i := range cases {
		res[i].data = cases[i].dummy(cases[i].ntris)
		res[i].mesh, err = NewMesh(&res[i].data)
		if err != nil {
			t.Log(cases[i])
			t.Fatalf("New: unexpected error: %#v", err)
		}
		cases[i].check(res[i].mesh, cases[i].ntris, t)
	}
	for i := range cases {
		cases[i].check(res[i].mesh, cases[i].ntris, t)
	}

	cap := storage.buf.Cap()
	if cap < n || cap == n && storage.buf != buf {
		t.Fatal("New: unexpected storage.buf state")
	}
	t.Logf("final storage.buf.Cap() is %.2f MiB", float64(cap)/(1<<20))
}

func TestMeshInputs(t *testing.T) {
	defer func() {
		b := setMeshBuffer(nil)
		if b != nil {
			b.Destroy()
		}
	}()

	check := func(want, have [][]driver.VertexIn) {
		if len(want) != len(have) {
			panic("bad check args")
		}
		for i := 0; i < len(want); i++ {
			if x, y := len(want[i]), len(have[i]); x != y {
				t.Fatalf("Mesh.inputs: length mismatch\nhave %d\nwant %d", y, x)
			}
			for j := 0; j < len(want[i]); j++ {
				if x, y := want[i][j], have[i][j]; x != y {
					t.Fatalf("Mesh.inputs: value mismatch\nhave %v\nwant %v", y, x)
				}
			}
		}
	}

	want := [6][]driver.VertexIn{
		{
			{Format: Position.format(), Stride: Position.format().Size(), Nr: Position.I()},
			{Format: Normal.format(), Stride: Normal.format().Size(), Nr: Normal.I()},
			{Format: TexCoord0.format(), Stride: TexCoord0.format().Size(), Nr: TexCoord0.I()},
		},
		{
			{Format: Position.format(), Stride: Position.format().Size(), Nr: Position.I()},
			{Format: Color0.format(), Stride: Color0.format().Size(), Nr: Color0.I()},
		},
		{
			{Format: Position.format(), Stride: Position.format().Size(), Nr: Position.I()},
			{Format: Normal.format(), Stride: Normal.format().Size(), Nr: Normal.I()},
			{Format: Tangent.format(), Stride: Tangent.format().Size(), Nr: Tangent.I()},
			{Format: TexCoord0.format(), Stride: TexCoord0.format().Size(), Nr: TexCoord0.I()},
			{Format: TexCoord1.format(), Stride: TexCoord1.format().Size(), Nr: TexCoord1.I()},
			{Format: Color0.format(), Stride: Color0.format().Size(), Nr: Color0.I()},
			{Format: Joints0.format(), Stride: Joints0.format().Size(), Nr: Joints0.I()},
			{Format: Weights0.format(), Stride: Weights0.format().Size(), Nr: Weights0.I()},
		},
		{
			{Format: Position.format(), Stride: Position.format().Size(), Nr: Position.I()},
			{Format: TexCoord1.format(), Stride: TexCoord1.format().Size(), Nr: TexCoord1.I()},
		},
		{
			{Format: Position.format(), Stride: Position.format().Size(), Nr: Position.I()},
		},
		{
			{Format: Position.format(), Stride: Position.format().Size(), Nr: Position.I()},
			{Format: Normal.format(), Stride: Normal.format().Size(), Nr: Normal.I()},
			{Format: TexCoord0.format(), Stride: TexCoord0.format().Size(), Nr: TexCoord0.I()},
			{Format: TexCoord1.format(), Stride: TexCoord1.format().Size(), Nr: TexCoord1.I()},
		},
	}

	have := [6][]driver.VertexIn{}
	var d MeshData
	var m *Mesh
	var err error

	d = dummyData1(200)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	have[0] = m.inputs(0)
	check(want[:1], have[:1])

	d = dummyData2(100)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	have[1] = m.inputs(0)
	check(want[:2], have[:2])

	d = dummyData3(1024)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	have[2] = m.inputs(0)
	check(want[:2], have[:2])

	d = dummyData4(60)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	have[3] = m.inputs(1)
	have[4] = m.inputs(2)
	have[5] = m.inputs(0)
	check(want[:], have[:])
}

func TestMeshFree(t *testing.T) {
	defer func() {
		b := setMeshBuffer(nil)
		if b != nil {
			b.Destroy()
		}
	}()

	type snapshot struct {
		spans                      []span
		nspan, nprim, rspan, rprim int
	}
	take := func(m *Mesh, s *snapshot) {
		s.spans = s.spans[:0]
		s.nspan, s.nprim = 0, m.primLen
		s.rspan, s.rprim = storage.spanMap.Rem(), storage.primMap.Rem()
		p := m.primIdx
		for {
			prim := &storage.prims[p]
			for i := 0; i < MaxSemantic; i++ {
				if prim.mask&(1<<i) != 0 {
					s.spans = append(s.spans, prim.vertex[i].span)
					s.nspan += prim.vertex[i].end - prim.vertex[i].start
				}
			}
			if prim.index.start < prim.index.end {
				s.spans = append(s.spans, prim.index.span)
				s.nspan += prim.index.end - prim.index.start
			}
			var ok bool
			if p, ok = storage.next(p); !ok {
				break
			}
		}
	}
	check := func(s *snapshot) {
		if x, y := storage.spanMap.Rem(), s.rspan+s.nspan; x != y {
			t.Fatalf("Mesh.Free: spanMap.Rem()\nhave %d\nwant %d\n(should remove %d block(s))", x, y, s.nspan)
		}
		if x, y := storage.primMap.Rem(), s.rprim+s.nprim; x != y {
			t.Fatalf("Mesh.Free: primMap.Rem()\nhave %d\nwant %d\n(should remove %d primitive(s))", x, y, s.nprim)
		}
		for _, x := range s.spans {
			for i := x.start; i < x.end; i++ {
				if storage.spanMap.IsSet(i) {
					t.Fatalf("Mesh.Free: spanMap.IsSet(%d)\nhave true\nwant false", i)
				}
			}
		}
	}

	var s snapshot
	var d MeshData
	var m *Mesh
	var err error

	d = dummyData1(500)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m, &s)
	m.Free()
	check(&s)

	d = dummyData2(5)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m, &s)
	m.Free()
	check(&s)

	d = dummyData3(1000)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m, &s)
	m.Free()
	check(&s)

	d = dummyData4(175)
	m, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m, &s)
	m.Free()
	check(&s)

	d = dummyData3(12)
	m1, err := NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData4(50)
	m2, err := NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData2(2000)
	m3, err := NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData1(1023)
	m4, err := NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m1, &s)
	m1.Free()
	check(&s)
	take(m2, &s)
	m2.Free()
	check(&s)
	take(m3, &s)
	m3.Free()
	check(&s)
	take(m4, &s)
	m4.Free()
	check(&s)

	d = dummyData3(12)
	m1, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData4(50)
	m2, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData2(2000)
	m3, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData1(1023)
	m4, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m4, &s)
	m4.Free()
	check(&s)
	take(m3, &s)
	m3.Free()
	check(&s)
	take(m2, &s)
	m2.Free()
	check(&s)
	take(m1, &s)
	m1.Free()
	check(&s)

	d = dummyData1(75)
	m1, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData2(140)
	m2, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m1, &s)
	m1.Free()
	check(&s)
	d = dummyData3(80)
	m3, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData4(513)
	m4, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m3, &s)
	m3.Free()
	check(&s)
	d = dummyData1(400)
	m1, err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	take(m4, &s)
	m4.Free()
	check(&s)
	take(m1, &s)
	m1.Free()
	check(&s)
	take(m2, &s)
	m2.Free()
	check(&s)

	const n = 100
	ms := make([]*Mesh, 0, n)
	for i := 0; i < n; i++ {
		switch i % 10 {
		case 1:
			d = dummyData1(99)
		case 2, 3, 8:
			d = dummyData2(88)
		case 4, 6, 7, 9:
			d = dummyData3(77)
		default:
			d = dummyData4(66)
		}
		if m, err = NewMesh(&d); err != nil {
			t.Fatalf("New failed: %#v", err)
		} else {
			ms = append(ms, m)
		}
	}
	take(ms[n/2], &s)
	ms[n/2].Free()
	check(&s)
	take(ms[n-1], &s)
	ms[n-1].Free()
	check(&s)
	take(ms[0], &s)
	ms[0].Free()
	check(&s)
	d = dummyData1(100)
	ms[n-1], err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData3(200)
	ms[0], err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	d = dummyData4(150)
	ms[n/2], err = NewMesh(&d)
	if err != nil {
		t.Fatalf("New failed: %#v", err)
	}
	for i := range ms {
		take(ms[i], &s)
		ms[i].Free()
		check(&s)
	}
}

const (
	nbufBench  = 64 << 20
	ntrisBench = 1000
)

// TODO: NewMesh locks the storage for writing during
// span searching, buffer growth and copying, and
// new data copying. The last step (new data copying)
// can be done with just a reading lock.
// Consider splitting the meshBuffer methods so New
// can release the writing lock as soon as all spans
// it needs have been reserved, and then copy the new
// data while holding a RLock.

func BenchmarkMeshGrow(b *testing.B) {
	if buf := setMeshBuffer(nil); buf != nil {
		buf.Destroy()
	}
	data := dummyData1(ntrisBench)
	b.Run("x", func(b *testing.B) {
		// Will grow the buffer on every iteration.
		// Expected to be very slow.
		b.RunParallel(func(bp *testing.PB) {
			for bp.Next() {
				if storage.buf != nil && storage.buf.Cap() > nbufBench {
					continue
				}
				for i := range data.Srcs {
					data.Srcs[i].Seek(0, io.SeekStart)
				}
				if _, err := NewMesh(&data); err != nil {
					b.Fatalf("NewMesh failed:\n%#v", err)
				}
			}
		})
	})
	b.Log("buf.Cap():", storage.buf.Cap())
	b.Log("spanMap.Rem()/Len():", storage.spanMap.Rem(), storage.spanMap.Len())
	b.Log("primMap.Rem()/Len():", storage.primMap.Rem(), storage.primMap.Len())
}

func BenchmarkMeshPre(b *testing.B) {
	buf, err := ctxt.GPU().NewBuffer(nbufBench, true, driver.UVertexData|driver.UIndexData)
	if err != nil {
		b.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	if buf = setMeshBuffer(buf); buf != nil {
		buf.Destroy()
	}
	data := dummyData1(ntrisBench)
	b.Run("x", func(b *testing.B) {
		// Will use pre-allocated memory.
		// Expected to be fast.
		b.RunParallel(func(bp *testing.PB) {
			for bp.Next() {
				if storage.buf != nil && storage.buf.Cap() > nbufBench {
					continue
				}
				for i := range data.Srcs {
					data.Srcs[i].Seek(0, io.SeekStart)
				}
				if _, err := NewMesh(&data); err != nil {
					b.Fatalf("NewMesh failed:\n%#v", err)
				}
			}
		})
	})
	b.Log("buf.Cap():", storage.buf.Cap())
	b.Log("spanMap.Rem()/Len():", storage.spanMap.Rem(), storage.spanMap.Len())
	b.Log("primMap.Rem()/Len():", storage.primMap.Rem(), storage.primMap.Len())
}

func BenchmarkMeshFree(b *testing.B) {
	if buf := setMeshBuffer(nil); buf != nil {
		buf.Destroy()
	}
	data := dummyData1(ntrisBench)
	b.Run("x", func(b *testing.B) {
		// Will create and then free the mesh,
		// so its spans can be reused.
		// Expected to be reasonably fast.
		b.RunParallel(func(bp *testing.PB) {
			for bp.Next() {
				if storage.buf != nil && storage.buf.Cap() > nbufBench {
					continue
				}
				for i := range data.Srcs {
					data.Srcs[i].Seek(0, io.SeekStart)
				}
				m, err := NewMesh(&data)
				if err != nil {
					b.Fatalf("NewMesh failed:\n%#v", err)
				}
				m.Free()
			}
		})
	})
	b.Log("buf.Cap():", storage.buf.Cap())
	b.Log("spanMap.Rem()/Len():", storage.spanMap.Rem(), storage.spanMap.Len())
	b.Log("primMap.Rem()/Len():", storage.primMap.Rem(), storage.primMap.Len())
}
