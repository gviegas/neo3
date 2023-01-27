// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package bitm defines a bitmap type useful for resource management
// (e.g., memory allocation and free list implementations).
package bitm

import (
	"unsafe"
)

// Uint represents the granularity of a bitmap.
type Uint interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Bitm is a growable bitmap with custom granularity.
type Bitm[T Uint] struct {
	m   []T
	rem int
}

// nbit returns the number of bits in T.
// TODO: This is not constant.
func (m *Bitm[T]) nbit() int { return int(unsafe.Sizeof(T(0))) * 8 }

// Len returns the number of bits in the map.
func (m *Bitm[_]) Len() int { return len(m.m) * m.nbit() }

// Rem returns the number of unset bits in the map.
func (m *Bitm[_]) Rem() int { return m.rem }

// Grow resizes the map to contain nplus additional Uints.
// The newly added extent will be available as a contiguous range
// of unset bits, such that requesting a range of
//
//	nplus * <number of bits in T>
//
// is guaranteed to succeed.
func (m *Bitm[T]) Grow(nplus int) {
	//m.m = append(m.m, make([]T, nplus)...)
	m.rem += nplus * m.nbit()
	var zeroes [16]T
	for nplus > len(zeroes) {
		m.m = append(m.m, zeroes[:]...)
		nplus -= len(zeroes)
	}
	m.m = append(m.m, zeroes[:nplus]...)
}

// Set sets a given bit.
func (m *Bitm[_]) Set(index int) {
	n := m.nbit()
	i := index / n
	m.m[i] |= 1 << (index & (n - 1))
}

// Unset unsets a given bit.
func (m *Bitm[_]) Unset(index int) {
	n := m.nbit()
	i := index / n
	m.m[i] &^= 1 << (index & (n - 1))
}

// IsSet checks whether a given bit is set.
func (m *Bitm[_]) IsSet(index int) bool {
	n := m.nbit()
	i := index / n
	return m.m[i]&(1<<(index&(n-1))) != 0
}
