// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package bitm

import (
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

func TestSetUnset(t *testing.T) {
	var bitm8 Bitm[uint8]
	bitm8.Grow(1)
	bitm8.Set(6)
	bitm8.checkState([]check[uint8]{{0, 0x40}}, t)
	bitm8.Set(1)
	bitm8.checkState([]check[uint8]{{0, 0x42}}, t)
	bitm8.Unset(6)
	bitm8.checkState([]check[uint8]{{0, 0x02}}, t)
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