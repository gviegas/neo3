// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package bitm

import (
	"strconv"
	"testing"
	"unsafe"
)

func TestNbit(t *testing.T) {
	for _, x := range [...][2]int{
		{int(unsafe.Sizeof(uint(0))) * 8, (&Bitm[uint]{}).nbit()},
		{int(unsafe.Sizeof(uint8(0))) * 8, (&Bitm[uint8]{}).nbit()},
		{int(unsafe.Sizeof(uint16(0))) * 8, (&Bitm[uint16]{}).nbit()},
		{int(unsafe.Sizeof(uint32(0))) * 8, (&Bitm[uint32]{}).nbit()},
		{int(unsafe.Sizeof(uint64(0))) * 8, (&Bitm[uint64]{}).nbit()},
		{int(unsafe.Sizeof(uintptr(0))) * 8, (&Bitm[uintptr]{}).nbit()},
	} {
		if x[0] != x[1] {
			t.Fatalf("Bitm[T].nbit:\nhave %d\nwant %d", x[0], x[1])
		}
	}
}

func TestZero(t *testing.T) {
	var bitm16 Bitm[uint16]
	if bitm16.m != nil {
		t.Fatalf("bitm16.m:\nhave %d\nwant nil", bitm16.m)
	}
	if bitm16.rem != 0 {
		t.Fatalf("bitm16.rem:\nhave %d\nwant 0", bitm16.rem)
	}
	if n := bitm16.Len(); n != 0 {
		t.Fatalf("bitm16.Len:\nhave %d\nwant 0", n)
	}
	if n := bitm16.Rem(); n != 0 {
		t.Fatalf("bitm16.Rem:\nhave %d\nwant 0", n)
	}
}

func TestGrow(t *testing.T) {
	var bitm32 Bitm[uint32]
	for _, x := range [...]struct {
		nplus, wantLen int
	}{
		{1, 32},
		{2, 96},
		{3, 192},
		{0, 192},
		{16, 704},
		{17, 1248},
		{32, 2272},
		{99, 5440},
	} {
		bitm32.Grow(x.nplus)
		if n := bitm32.Len(); n != x.wantLen {
			t.Fatalf("bitm32.Grow: Len:\nhave %d\nwant %d", n, x.wantLen)
		}
		if n := bitm32.Rem(); n != x.wantLen {
			t.Fatalf("bitm32.Grow: Rem:\nhave %d\nwant %d", n, x.wantLen)
		}
		for i, x := range bitm32.m {
			if x != 0 {
				t.Fatalf("bitm32.m[%d]:\nhave %d\nwant 0", i, x)
			}
		}
	}
}

// check represents an expected Bitm.m[index] value.
type check[T Uint] struct {
	index int
	want  T
}

// checkState checks the state of m.m against a set of expected values.
func (m *Bitm[T]) checkState(v []check[T], t *testing.T) {
	for _, x := range v {
		if y := m.m[x.index]; y != x.want {
			t.Fatalf("m.m[%d]:\nhave 0x%x\nwant 0x%x", x.index, y, x.want)
		}
	}
}

// checkRem checks that m.Rem() matches the state of m.m.
func (m *Bitm[T]) checkRem(t *testing.T) {
	want := m.Len()
	n := m.nbit()
	for _, x := range m.m {
		for i := 0; i < n; i++ {
			if x&(1<<i) != 0 {
				want--
			}
		}
	}
	if r := m.Rem(); r != want {
		t.Fatalf("m.Rem:\nhave %d\nwant %d", r, want)
	}
}

func TestSetUnset(t *testing.T) {
	var bitm8 Bitm[uint8]
	bitm8.Grow(1)
	bitm8.Set(6)
	bitm8.checkState([]check[uint8]{{0, 0x40}}, t)
	bitm8.Set(1)
	bitm8.checkState([]check[uint8]{{0, 0x42}}, t)
	bitm8.checkRem(t)
	bitm8.Unset(6)
	bitm8.checkState([]check[uint8]{{0, 0x02}}, t)
	bitm8.checkRem(t)
	bitm8.Set(6)
	bitm8.checkState([]check[uint8]{{0, 0x42}}, t)
	bitm8.Grow(2)
	bitm8.checkState([]check[uint8]{{0, 0x42}, {1, 0}, {2, 0}}, t)
	bitm8.Set(10)
	bitm8.checkState([]check[uint8]{{0, 0x42}, {1, 0x04}, {2, 0}}, t)
	bitm8.Unset(1)
	bitm8.checkState([]check[uint8]{{0, 0x40}, {1, 0x04}, {2, 0}}, t)
	bitm8.Set(21)
	bitm8.checkState([]check[uint8]{{0, 0x40}, {1, 0x04}, {2, 0x20}}, t)
	bitm8.Set(21)
	bitm8.Unset(23)
	bitm8.Unset(0)
	bitm8.checkState([]check[uint8]{{0, 0x40}, {1, 0x04}, {2, 0x20}}, t)
	bitm8.checkRem(t)
	bitm8.Set(4)
	bitm8.Set(14)
	bitm8.Set(16)
	bitm8.checkState([]check[uint8]{{0, 0x50}, {1, 0x44}, {2, 0x21}}, t)
	for i := 0; i < bitm8.Len(); i++ {
		if i&3 == 0 {
			bitm8.Set(i)
		} else {
			bitm8.Unset(i)
		}
	}
	bitm8.checkState([]check[uint8]{{0, 0x11}, {1, 0x11}, {2, 0x11}}, t)
	bitm8.checkRem(t)
}

func TestIsSet(t *testing.T) {
	var bitm64 Bitm[uint64]
	bitm64.Grow(2)
	checkUnset := func(start, end int) {
		for i := start; i < end; i++ {
			if bitm64.IsSet(i) {
				t.Fatalf("bitm64.isSet: %d:\nhave true\nwant false", i)
			}
		}
	}
	checkSet := func(start, end int) {
		for i := start; i < end; i++ {
			if !bitm64.IsSet(i) {
				t.Fatalf("bitm64.isSet: %d:\nhave false\nwant true", i)
			}
		}
	}
	checkUnset(0, bitm64.Len())
	bitm64.Set(0)
	checkSet(0, 1)
	checkUnset(1, bitm64.Len())
	bitm64.Set(1)
	checkSet(0, 2)
	bitm64.Unset(0)
	checkUnset(0, 1)
	checkSet(1, 2)
	bitm64.Set(bitm64.Len() - 1)
	checkSet(bitm64.Len()-1, bitm64.Len())
	for i := 0; i < bitm64.Len(); i++ {
		bitm64.Unset(i)
	}
	checkUnset(0, bitm64.Len())
	for i := 0; i < bitm64.Len(); i++ {
		bitm64.Set(i)
	}
	checkSet(0, bitm64.Len())
}

// checkSearch calls m.Search and checks the expected result.
// If want < 0, then Search must fail.
func (m *Bitm[_]) checkSearch(want int, t *testing.T) {
	index, ok := m.Search()
	if want < 0 {
		if ok {
			t.Fatalf("m.Search: \nhave %d, true\nwant _, false", index)
		}
	} else {
		if !ok {
			t.Fatalf("m.Search: \nhave _, false\nwant %d, true", want)
		}
		if index != want {
			t.Fatalf("m.Search: index:\nhave %d\nwant %d", index, want)
		}
	}
}

func TestSearch(t *testing.T) {
	var bitm32 Bitm[uint32]
	bitm32.checkSearch(-1, t)
	bitm32.Grow(12)
	bitm32.checkSearch(0, t)
	bitm32.Set(0)
	bitm32.checkSearch(1, t)
	bitm32.Set(1)
	bitm32.checkSearch(2, t)
	bitm32.Set(3)
	bitm32.checkSearch(2, t)
	bitm32.Unset(1)
	bitm32.checkSearch(1, t)
	bitm32.Unset(0)
	bitm32.checkSearch(0, t)
	for i := 0; i < bitm32.nbit()*2; i++ {
		bitm32.Set(i)
	}
	bitm32.checkSearch(64, t)
	for i := 64; i < bitm32.Len(); i++ {
		bitm32.Set(i)
	}
	bitm32.checkSearch(-1, t)
	bitm32.Unset(120)
	bitm32.checkSearch(120, t)
}

// checkSearchRange calls m.SearchRange and checks the expected result.
// If want < 0, then SearchRange must fail.
func (m *Bitm[_]) checkSearchRange(n, want int, t *testing.T) {
	index, ok := m.SearchRange(n)
	if want < 0 {
		if ok {
			t.Fatalf("m.SearchRange: \nhave %d, true\nwant _, false", index)
		}
	} else {
		if !ok {
			t.Fatalf("m.SearchRange: \nhave _, false\nwant %d, true", want)
		}
		if index != want {
			t.Fatalf("m.SearchRange: index:\nhave %d\nwant %d", index, want)
		}
	}
}

func TestSearchRange(t *testing.T) {
	var bitm16 Bitm[uint16]
	setRange := func(start, end int) {
		for i := start; i < end; i++ {
			bitm16.Set(i)
		}
	}
	bitm16.checkSearchRange(3, -1, t)
	bitm16.Grow(4)
	bitm16.checkSearchRange(3, 0, t)
	setRange(0, 3)
	bitm16.checkSearchRange(3, 3, t)
	setRange(3, 6)
	bitm16.checkSearchRange(3, 6, t)
	setRange(6, 9)
	bitm16.checkSearchRange(1, 9, t)
	bitm16.Set(9)
	bitm16.checkSearchRange(2, 10, t)
	setRange(10, 12)
	bitm16.Unset(1)
	bitm16.checkSearchRange(2, 12, t)
	bitm16.checkSearchRange(1, 1, t)
	bitm16.Unset(2)
	bitm16.checkSearchRange(2, 1, t)
	bitm16.checkSearchRange(1, 1, t)
	bitm16.checkSearchRange(6, 12, t)
	setRange(12, 18)
	bitm16.checkSearchRange(13, 18, t)
	setRange(19, 32)
	bitm16.Set(35)
	bitm16.Set(46)
	bitm16.checkSearchRange(4, 36, t)
	bitm16.checkSearchRange(3, 32, t)
	bitm16.checkSearchRange(10, 36, t)
	bitm16.checkSearchRange(11, 47, t)
	bitm16.checkSearchRange(20, -1, t)
	bitm16.Grow(1)
	bitm16.checkSearchRange(20, 47, t)
	bitm16.checkSearchRange(31, 47, t)
	bitm16.checkSearchRange(33, 47, t)
	bitm16.checkSearchRange(34, -1, t)
	bitm16.Set(76)
	bitm16.checkSearchRange(20, 47, t)
	bitm16.checkSearchRange(31, -1, t)
	bitm16.checkSearchRange(33, -1, t)
	bitm16.checkSearchRange(34, -1, t)
	bitm16.Grow(5)
	bitm16.checkSearchRange(80, 77, t)
	bitm16.Set(79)
	bitm16.checkSearchRange(80, 80, t)
	bitm16.Set(80)
	bitm16.checkSearchRange(80, -1, t)
	bitm16.checkSearchRange(79, 81, t)
}

func TestClear(t *testing.T) {
	var bitmu Bitm[uint]
	checkClear := func() {
		if bitmu.Len() != bitmu.Rem() {
			t.Fatal("bitmu.Clear: Len == Rem\nhave false\nwant true")

		}
		for i, x := range bitmu.m {
			if x != 0 {
				t.Fatalf("bitmu.Clear: m[%d]\nhave %d\nwant 0", i, x)
			}
		}
	}
	checkClear()
	bitmu.Grow(1)
	checkClear()
	for i := 0; i < bitmu.Len(); i++ {
		bitmu.Set(i)
	}
	bitmu.Clear()
	checkClear()
	bitmu.Grow(9)
	checkClear()
	for i := 0; i < bitmu.Len(); i++ {
		bitmu.Set(i)
	}
	bitmu.Clear()
	checkClear()
	for i := bitmu.nbit(); i < bitmu.Len(); i += 3 {
		bitmu.Set(i)
	}
	bitmu.Clear()
	checkClear()
	for i := bitmu.nbit(); i < bitmu.Len()-bitmu.nbit(); i++ {
		bitmu.Set(i)
	}
	bitmu.Clear()
	checkClear()
}

// printm is for debug printing of Bitm.m.
func printm[T Uint](m *Bitm[T]) {
	n := m.nbit()
	s := "\n"
	for i, x := range m.m {
		for i := 0; i < n; i++ {
			if x&(1<<i) != 0 {
				s += "1 "
			} else {
				s += "0 "
			}
		}
		s += " " + strconv.Itoa(i*n) + ":" + strconv.Itoa(i*n+n-1) + "\n"
	}
	print(s)
}
