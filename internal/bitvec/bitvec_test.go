// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package bitvec

import (
	"iter"
	"strconv"
	"testing"
	"unsafe"
)

func TestNbit(t *testing.T) {
	for _, x := range [...][2]int{
		{int(unsafe.Sizeof(uint(0))) * 8, (&V[uint]{}).nbit()},
		{int(unsafe.Sizeof(uint8(0))) * 8, (&V[uint8]{}).nbit()},
		{int(unsafe.Sizeof(uint16(0))) * 8, (&V[uint16]{}).nbit()},
		{int(unsafe.Sizeof(uint32(0))) * 8, (&V[uint32]{}).nbit()},
		{int(unsafe.Sizeof(uint64(0))) * 8, (&V[uint64]{}).nbit()},
		{int(unsafe.Sizeof(uintptr(0))) * 8, (&V[uintptr]{}).nbit()},
	} {
		if x[0] != x[1] {
			t.Fatalf("V[T].nbit:\nhave %d\nwant %d", x[0], x[1])
		}
	}
}

func TestZero(t *testing.T) {
	var v16 V[uint16]
	if v16.s != nil {
		t.Fatalf("v16.s:\nhave %d\nwant nil", v16.s)
	}
	if v16.rem != 0 {
		t.Fatalf("v16.rem:\nhave %d\nwant 0", v16.rem)
	}
	if n := v16.Len(); n != 0 {
		t.Fatalf("v16.Len:\nhave %d\nwant 0", n)
	}
	if n := v16.Rem(); n != 0 {
		t.Fatalf("v16.Rem:\nhave %d\nwant 0", n)
	}
}

func TestGrow(t *testing.T) {
	var v32 V[uint32]
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
		{-1, 5440},
		{32768, 1054016},
	} {
		if n, i := v32.Len(), v32.Grow(x.nplus); n != i {
			t.Fatalf("v32.Grow:\nhave %d\nwant %d", i, n)
		}
		if n := v32.Len(); n != x.wantLen {
			t.Fatalf("v32.Grow: Len:\nhave %d\nwant %d", n, x.wantLen)
		}
		if n := v32.Rem(); n != x.wantLen {
			t.Fatalf("v32.Grow: Rem:\nhave %d\nwant %d", n, x.wantLen)
		}
		for i, x := range v32.s {
			if x != 0 {
				t.Fatalf("v32.s[%d]:\nhave %d\nwant 0", i, x)
			}
		}
	}
}

func TestShrink(t *testing.T) {
	var v8 V[uint8]
	checkLenRem := func(len, rem int) {
		if n := v8.Len(); n != len {
			t.Fatalf("v8.Shrink: Len:\nhave %d\nwant %d", n, len)
		}
		if n := v8.Rem(); n != rem {
			t.Fatalf("v8.Shrink: Rem:\nhave %d\nwant %d", n, rem)
		}
	}
	for _, x := range [...]int{0, 1, -1, 2, 100} {
		v8.Shrink(x)
		checkLenRem(0, 0)
	}
	v8.Grow(1)
	v8.Shrink(1)
	checkLenRem(0, 0)
	v8.Grow(1)
	v8.Set(0)
	v8.Shrink(1)
	checkLenRem(0, 0)
	v8.Grow(2)
	v8.Shrink(1)
	checkLenRem(8, 8)
	v8.Grow(2)
	v8.Shrink(1)
	checkLenRem(16, 16)
	v8.Shrink(2)
	checkLenRem(0, 0)
	v8.Grow(2)
	v8.Set(0)
	v8.Shrink(1)
	checkLenRem(8, 7)
	v8.Shrink(1)
	checkLenRem(0, 0)
	v8.Grow(2)
	v8.Set(1)
	v8.Set(7)
	v8.Set(4)
	v8.Set(0)
	v8.Shrink(0)
	checkLenRem(16, 12)
	v8.Shrink(1)
	checkLenRem(8, 4)
	v8.Grow(2)
	v8.Set(10)
	v8.Set(15)
	v8.Shrink(-1)
	checkLenRem(24, 18)
	v8.Shrink(1)
	checkLenRem(16, 10)
	v8.Grow(1)
	v8.Set(23)
	v8.Shrink(^0)
	checkLenRem(24, 17)
	v8.Shrink(2)
	checkLenRem(8, 4)
	v8.Unset(0)
	v8.Grow(10)
	v8.Set(79)
	v8.Shrink(9)
	checkLenRem(16, 13)
	v8.Grow(8)
	for i := range v8.Len() {
		v8.Set(i)
	}
	v8.Shrink(0)
	checkLenRem(80, 0)
	for i := 72; i >= 0; i -= 8 {
		v8.Shrink(1)
		checkLenRem(i, 0)
	}
	v8.Grow(10)
	for i := 80; i > -3; i -= 8 {
		checkLenRem(i, i)
		v8.Shrink(1)
	}
	v8.Grow(100)
	for i := 0; i < v8.Len(); i += 2 {
		v8.Set(i)
	}
	for i := 0; i <= 100; i += 2 {
		checkLenRem(800-i*8, 400-i*4)
		v8.Shrink(2)
	}
	checkLenRem(0, 0)
}

// check represents an expected V.s[index] value.
type check[T Uint] struct {
	index int
	want  T
}

// checkState checks the state of v.s against a set of expected values.
func (v *V[T]) checkState(values []check[T], t *testing.T) {
	for _, x := range values {
		if y := v.s[x.index]; y != x.want {
			t.Fatalf("v.s[%d]:\nhave 0x%x\nwant 0x%x", x.index, y, x.want)
		}
	}
}

// checkRem checks that v.Rem() matches the state of v.s.
func (v *V[T]) checkRem(t *testing.T) {
	want := v.Len()
	n := v.nbit()
	for _, x := range v.s {
		for i := range n {
			if x&(1<<i) != 0 {
				want--
			}
		}
	}
	if r := v.Rem(); r != want {
		t.Fatalf("v.Rem:\nhave %d\nwant %d", r, want)
	}
}

func TestSetUnset(t *testing.T) {
	var v8 V[uint8]
	v8.Grow(1)
	v8.Set(6)
	v8.checkState([]check[uint8]{{0, 0x40}}, t)
	v8.Set(1)
	v8.checkState([]check[uint8]{{0, 0x42}}, t)
	v8.checkRem(t)
	v8.Unset(6)
	v8.checkState([]check[uint8]{{0, 0x02}}, t)
	v8.checkRem(t)
	v8.Set(6)
	v8.checkState([]check[uint8]{{0, 0x42}}, t)
	v8.Grow(2)
	v8.checkState([]check[uint8]{{0, 0x42}, {1, 0}, {2, 0}}, t)
	v8.Set(10)
	v8.checkState([]check[uint8]{{0, 0x42}, {1, 0x04}, {2, 0}}, t)
	v8.Unset(1)
	v8.checkState([]check[uint8]{{0, 0x40}, {1, 0x04}, {2, 0}}, t)
	v8.Set(21)
	v8.checkState([]check[uint8]{{0, 0x40}, {1, 0x04}, {2, 0x20}}, t)
	v8.Set(21)
	v8.Unset(23)
	v8.Unset(0)
	v8.checkState([]check[uint8]{{0, 0x40}, {1, 0x04}, {2, 0x20}}, t)
	v8.checkRem(t)
	v8.Set(4)
	v8.Set(14)
	v8.Set(16)
	v8.checkState([]check[uint8]{{0, 0x50}, {1, 0x44}, {2, 0x21}}, t)
	for i := range v8.Len() {
		if i&3 == 0 {
			v8.Set(i)
		} else {
			v8.Unset(i)
		}
	}
	v8.checkState([]check[uint8]{{0, 0x11}, {1, 0x11}, {2, 0x11}}, t)
	v8.checkRem(t)
}

func TestIsSet(t *testing.T) {
	var v64 V[uint64]
	v64.Grow(2)
	checkUnset := func(start, end int) {
		for i := start; i < end; i++ {
			if v64.IsSet(i) {
				t.Fatalf("v64.isSet: %d:\nhave true\nwant false", i)
			}
		}
	}
	checkSet := func(start, end int) {
		for i := start; i < end; i++ {
			if !v64.IsSet(i) {
				t.Fatalf("v64.isSet: %d:\nhave false\nwant true", i)
			}
		}
	}
	checkUnset(0, v64.Len())
	v64.Set(0)
	checkSet(0, 1)
	checkUnset(1, v64.Len())
	v64.Set(1)
	checkSet(0, 2)
	v64.Unset(0)
	checkUnset(0, 1)
	checkSet(1, 2)
	v64.Set(v64.Len() - 1)
	checkSet(v64.Len()-1, v64.Len())
	for i := range v64.Len() {
		v64.Unset(i)
	}
	checkUnset(0, v64.Len())
	for i := range v64.Len() {
		v64.Set(i)
	}
	checkSet(0, v64.Len())
}

// checkSearch calls v.Search and checks the expected result.
// If want < 0, then Search must fail.
func (v *V[_]) checkSearch(want int, t *testing.T) {
	index, ok := v.Search()
	if want < 0 {
		if ok {
			t.Fatalf("v.Search: \nhave %d, true\nwant _, false", index)
		}
	} else {
		if !ok {
			t.Fatalf("v.Search: \nhave _, false\nwant %d, true", want)
		}
		if index != want {
			t.Fatalf("v.Search: index:\nhave %d\nwant %d", index, want)
		}
	}
}

func TestSearch(t *testing.T) {
	var v32 V[uint32]
	v32.checkSearch(-1, t)
	v32.Grow(12)
	v32.checkSearch(0, t)
	v32.Set(0)
	v32.checkSearch(1, t)
	v32.Set(1)
	v32.checkSearch(2, t)
	v32.Set(3)
	v32.checkSearch(2, t)
	v32.Unset(1)
	v32.checkSearch(1, t)
	v32.Unset(0)
	v32.checkSearch(0, t)
	for i := range v32.nbit() * 2 {
		v32.Set(i)
	}
	v32.checkSearch(64, t)
	for i := 64; i < v32.Len(); i++ {
		v32.Set(i)
	}
	v32.checkSearch(-1, t)
	v32.Unset(120)
	v32.checkSearch(120, t)
}

// checkSearchRange calls v.SearchRange and checks the expected result.
// If want < 0, then SearchRange must fail.
func (v *V[_]) checkSearchRange(n, want int, t *testing.T) {
	index, ok := v.SearchRange(n)
	if want < 0 {
		if ok {
			t.Fatalf("v.SearchRange: \nhave %d, true\nwant _, false", index)
		}
	} else {
		if !ok {
			t.Fatalf("v.SearchRange: \nhave _, false\nwant %d, true", want)
		}
		if index != want {
			t.Fatalf("v.SearchRange: index:\nhave %d\nwant %d", index, want)
		}
	}
}

func TestSearchRange(t *testing.T) {
	var v16 V[uint16]
	setRange := func(start, end int) {
		for i := start; i < end; i++ {
			v16.Set(i)
		}
	}
	v16.checkSearchRange(3, -1, t)
	v16.Grow(4)
	v16.checkSearchRange(3, 0, t)
	setRange(0, 3)
	v16.checkSearchRange(3, 3, t)
	setRange(3, 6)
	v16.checkSearchRange(3, 6, t)
	setRange(6, 9)
	v16.checkSearchRange(1, 9, t)
	v16.Set(9)
	v16.checkSearchRange(2, 10, t)
	setRange(10, 12)
	v16.Unset(1)
	v16.checkSearchRange(2, 12, t)
	v16.checkSearchRange(1, 1, t)
	v16.Unset(2)
	v16.checkSearchRange(2, 1, t)
	v16.checkSearchRange(1, 1, t)
	v16.checkSearchRange(6, 12, t)
	setRange(12, 18)
	v16.checkSearchRange(13, 18, t)
	setRange(19, 32)
	v16.Set(35)
	v16.Set(46)
	v16.checkSearchRange(4, 36, t)
	v16.checkSearchRange(3, 32, t)
	v16.checkSearchRange(10, 36, t)
	v16.checkSearchRange(11, 47, t)
	v16.checkSearchRange(20, -1, t)
	v16.Grow(1)
	v16.checkSearchRange(20, 47, t)
	v16.checkSearchRange(31, 47, t)
	v16.checkSearchRange(33, 47, t)
	v16.checkSearchRange(34, -1, t)
	v16.Set(76)
	v16.checkSearchRange(20, 47, t)
	v16.checkSearchRange(31, -1, t)
	v16.checkSearchRange(33, -1, t)
	v16.checkSearchRange(34, -1, t)
	v16.Grow(5)
	v16.checkSearchRange(80, 77, t)
	v16.Set(79)
	v16.checkSearchRange(80, 80, t)
	v16.Set(80)
	v16.checkSearchRange(80, -1, t)
	v16.checkSearchRange(79, 81, t)
}

func TestClear(t *testing.T) {
	var vu V[uint]
	checkClear := func() {
		if vu.Len() != vu.Rem() {
			t.Fatal("vu.Clear: Len == Rem\nhave false\nwant true")

		}
		for i, x := range vu.s {
			if x != 0 {
				t.Fatalf("vu.Clear: s[%d]\nhave %d\nwant 0", i, x)
			}
		}
	}
	checkClear()
	vu.Grow(1)
	checkClear()
	for i := range vu.Len() {
		vu.Set(i)
	}
	vu.Clear()
	checkClear()
	vu.Grow(9)
	checkClear()
	for i := range vu.Len() {
		vu.Set(i)
	}
	vu.Clear()
	checkClear()
	for i := vu.nbit(); i < vu.Len(); i += 3 {
		vu.Set(i)
	}
	vu.Clear()
	checkClear()
	for i := vu.nbit(); i < vu.Len()-vu.nbit(); i++ {
		vu.Set(i)
	}
	vu.Clear()
	checkClear()
}

// checkAll checks that v.All() produces the expected result.
func (v *V[T]) checkAll(t *testing.T) {
	checkWith := func(it func() iter.Seq2[int, bool]) {
		for _, last := range [...]int{
			v.Len() - 1,
			v.Len()/2 - 1,
			v.Len()/3 - 1,
		} {
			prev := -1
			for i, x := range it() {
				if prev == last {
					break
				}
				if i-1 != prev {
					t.Fatalf("v.All:\nhave %d, _\nwant %d, _", i, prev+1)
				}
				if x != v.IsSet(i) {
					t.Fatalf("v.All:\nhave %d, %t\nwant %d, %t", i, x, i, !x)
				}
				prev = i
			}
		}
	}
	checkWith(func() iter.Seq2[int, bool] { return v.All() })
	it := v.All()
	checkWith(func() iter.Seq2[int, bool] { return it })
}

func TestAll(t *testing.T) {
	var v16 V[uint16]
	v16.checkAll(t)
	v16.Grow(1)
	v16.checkAll(t)
	v16.Set(0)
	v16.checkAll(t)
	v16.Unset(0)
	v16.checkAll(t)
	v16.Set(1)
	v16.Set(15)
	v16.checkAll(t)
	v16.Grow(10)
	v16.checkAll(t)
	v16.Set(100)
	v16.Set(101)
	v16.Set(102)
	v16.checkAll(t)
	v16.Set(0)
	v16.checkAll(t)
	v16.Set(v16.Len() - 1)
	v16.checkAll(t)
	for i := range v16.Len() {
		v16.Set(i)
	}
	v16.checkAll(t)
	v16.Shrink(5)
	v16.checkAll(t)
	v16.Clear()
	v16.checkAll(t)
}

// checkOnly checks that v.Only() produces the expected result.
func (v *V[T]) checkOnly(t *testing.T) {
	checkWith := func(it func() iter.Seq[int], set bool) {
		n := v.Rem()
		if set {
			n = v.Len() - n
		}
		for _, max := range [...]int{
			n,
			n / 2,
			n / 3,
		} {
			var n int
			prev := -1
			for i := range it() {
				if n == max {
					break
				}
				n++
				if i <= prev {
					t.Fatalf("v.Only:\nhave %d\nwant > %d", i, prev)
				}
				if isSet := v.IsSet(i); set && !isSet || !set && isSet {
					printVec(v)
					println(set)
					t.Fatalf("v.Only: IsSet(%d)\nhave %t\nwant %t", i, isSet, !isSet)
				}
				prev = i
			}
			if n != max {
				t.Fatalf("v.Only: wrong number of iterations\nhave %d\nwant %d", n, max)
			}
		}
	}
	for _, x := range [2]bool{true, false} {
		checkWith(func() iter.Seq[int] { return v.Only(x) }, x)
		it := v.Only(x)
		checkWith(func() iter.Seq[int] { return it }, x)
	}
}

func TestOnly(t *testing.T) {
	var v32 V[uint32]
	v32.checkOnly(t)
	v32.Grow(1)
	v32.checkOnly(t)
	v32.Set(0)
	v32.checkOnly(t)
	v32.Set(1)
	v32.checkOnly(t)
	v32.Unset(0)
	v32.checkOnly(t)
	v32.Grow(9)
	v32.checkOnly(t)
	v32.Set(32)
	v32.Set(33)
	v32.Set(66)
	v32.Set(v32.Len() - 1)
	v32.Set(0)
	v32.checkOnly(t)
	for i := range v32.Len() {
		v32.Set(i)
	}
	v32.checkOnly(t)
	v32.Shrink(5)
	v32.checkOnly(t)
	v32.Clear()
	v32.checkOnly(t)
}

// printVec is for debug printing of V.s.
func printVec[T Uint](v *V[T]) {
	n := v.nbit()
	s := "\n"
	for i, x := range v.s {
		for i := range n {
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
