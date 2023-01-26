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
			t.Fatalf("Bitm[T].nbit:\nhave %v\nwant %v", x[0], x[1])
		}
	}
}

func TestZero(t *testing.T) {
	var bitm16 Bitm[uint16]
	if bitm16.m != nil {
		t.Fatalf("bitm16.m:\nhave %v\nwant nil", bitm16.m)
	}
	if bitm16.rem != 0 {
		t.Fatalf("bitm16.rem:\nhave %v\nwant 0", bitm16.rem)
	}
	if n := bitm16.Len(); n != 0 {
		t.Fatalf("bitm16.Len:\nhave %v\nwant 0", n)
	}
	if n := bitm16.Cap(); n != 0 {
		t.Fatalf("bitm16.Cap:\nhave %v\nwant 0", n)
	}
}
