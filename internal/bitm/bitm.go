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
func (m *Bitm[T]) Set(index int) {
	n := m.nbit()
	i := index / n
	b := T(1) << (index & (n - 1))
	if m.m[i]&b == 0 {
		m.m[i] |= b
		m.rem--
	}
}

// Unset unsets a given bit.
func (m *Bitm[T]) Unset(index int) {
	n := m.nbit()
	i := index / n
	b := T(1) << (index & (n - 1))
	if m.m[i]&b != 0 {
		m.m[i] &^= b
		m.rem++
	}
}

// IsSet checks whether a given bit is set.
func (m *Bitm[T]) IsSet(index int) bool {
	n := m.nbit()
	i := index / n
	b := T(1) << (index & (n - 1))
	return m.m[i]&b != 0
}

// Search attempts to locate an unset bit in the map.
// If ok is true, then index is a value suitable for use in
// a call to m.Set.
// This method will fail only when m.Rem() == 0.
func (m *Bitm[T]) Search() (index int, ok bool) {
	if m.Rem() == 0 {
		return
	}
	for i, x := range m.m {
		if x == ^T(0) {
			continue
		}
		var b int
		for ; x&(1<<b) != 0; b++ {
		}
		index = i*m.nbit() + b
		ok = true
		break
	}
	return
}

// SearchRange attempts to locate a contiguous range of unset bits.
// If ok is true, then all values in the range [index, index + n)
// are suitable for use in a call to m.Set.
// It calls Search if n <= 1.
func (m *Bitm[T]) SearchRange(n int) (index int, ok bool) {
	if n <= 1 {
		return m.Search()
	}
	if m.Rem() < n {
		return
	}
	nb := m.nbit()
	var cnt, idx, bit, i int
	for {
		// Skip Uints that have no unset bits.
		if m.m[i] == ^T(0) {
			cnt, bit = 0, 0
			i++
			for ; i < len(m.m); i++ {
				if m.m[i] != ^T(0) {
					break
				}
			}
			idx = i
		}
		// Give up if there is not enough bits left.
		if cnt+nb*(len(m.m)-i) < n {
			return
		}
		// Iterate over whole Uints as much as possible.
		if m.m[i] == 0 {
			cnt += nb
			i++
			for j := 0; j < (n-cnt)/nb; j++ {
				if m.m[i+j] != 0 {
					cnt += j * nb
					i += j
					break
				}
			}
			if cnt >= n {
				index = idx*nb + bit
				ok = true
				break
			}
		}
		// Iterate over the bits of the ith Uint.
		// There are three possibilities:
		//
		// 1. It completes a range (i.e., bits 0:n-cnt
		//    are unset) or
		// 2. There is a range of n unset bits contained
		//    within this Uint or
		// 3. It has a (possibly empty) subrange x:nb of
		//    unset bits that may yet form a full range
		//    with subsequent Uint(s).
		//
		for j := 0; j < nb; j++ {
			if m.m[i]&(1<<j) == 0 {
				cnt++
				if cnt >= n {
					index = idx*nb + bit
					ok = true
					return
				}
				continue
			}
			cnt = 0
			if j < nb-1 {
				idx = i
				bit = j + 1
			} else {
				idx = i + 1
				bit = 0
			}
		}
		i++
		if i == len(m.m) {
			break
		}
	}
	return
}

// Clear unsets every bit in the map.
func (m *Bitm[T]) Clear() {
	if m.Len() == m.Rem() {
		return
	}
	const n = 16
	var zeroes [n]T
	s := m.m[:]
	for {
		if copy(s, zeroes[:]) < n {
			m.rem = m.Len()
			return
		}
		s = s[n:]
	}
}
