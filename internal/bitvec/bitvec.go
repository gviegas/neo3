// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package bitvec defines a bit vector type useful for
// resource management (e.g., memory allocation and
// free list implementations).
package bitvec

import (
	"iter"
	"unsafe"
)

// Uint represents the granularity of a bit vector.
type Uint interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// V is a growable bit vector with custom granularity.
type V[T Uint] struct {
	s   []T
	rem int
}

// nbit returns the number of bits in T.
// TODO: This is not constant.
func (*V[T]) nbit() int { return int(unsafe.Sizeof(T(0))) * 8 }

// Len returns the number of bits in the vector.
func (v *V[_]) Len() int { return len(v.s) * v.nbit() }

// Rem returns the number of unset bits in the vector.
func (v *V[_]) Rem() int { return v.rem }

// Grow resizes the vector to contain nplus additional Uints.
// The new extent will be appended as a contiguous range of
// unset bits, such that requesting the range
//
//	nplus * <number of bits in T>
//
// is guaranteed to succeed.
// It returns the value of v.Len prior to appending the new
// extent, so if nplus is less than 1, this value will be
// out of bounds.
// It is valid to call this method with any value of nplus.
func (v *V[T]) Grow(nplus int) (index int) {
	index = v.Len()
	if nplus > 0 {
		v.rem += nplus * v.nbit()
		v.s = append(v.s, make([]T, nplus)...)
	}
	return
}

// Shrink resizes the vector to contain nminus less Uints.
// This in effect truncates the vector, so the bits in the range
//
//	v.Len() - nminus*<number of bits in T> : v.Len()
//
// will be removed.
// It is valid to call this method with any value of nminus.
func (v *V[T]) Shrink(nminus int) {
	if nminus <= 0 {
		return
	}
	n := len(v.s) - nminus
	if n <= 0 {
		v.s = v.s[:0]
		v.rem = 0
		return
	}
	for i := n; i < n+nminus; i++ {
		switch v.s[i] {
		case 0:
			v.rem -= v.nbit()
		case ^T(0):
		default:
			for x := ^v.s[i]; x != 0; x >>= 1 {
				if x&1 == 1 {
					v.rem--
				}
			}
		}
	}
	v.s = v.s[:n]
}

// Set sets a given bit.
func (v *V[T]) Set(index int) {
	n := v.nbit()
	i := index / n
	b := T(1) << (index & (n - 1))
	if v.s[i]&b == 0 {
		v.s[i] |= b
		v.rem--
	}
}

// Unset unsets a given bit.
func (v *V[T]) Unset(index int) {
	n := v.nbit()
	i := index / n
	b := T(1) << (index & (n - 1))
	if v.s[i]&b != 0 {
		v.s[i] &^= b
		v.rem++
	}
}

// IsSet checks whether a given bit is set.
func (v *V[T]) IsSet(index int) bool {
	n := v.nbit()
	i := index / n
	b := T(1) << (index & (n - 1))
	return v.s[i]&b != 0
}

// Search attempts to locate an unset bit in the vector.
// If ok is true, then index is a value suitable for use in
// a call to v.Set.
// This method will fail only when v.Rem() == 0.
func (v *V[T]) Search() (index int, ok bool) {
	if v.Rem() == 0 {
		return
	}
	for i, x := range v.s {
		if x == ^T(0) {
			continue
		}
		var b int
		for ; x&(1<<b) != 0; b++ {
		}
		index = i*v.nbit() + b
		ok = true
		break
	}
	return
}

// SearchRange attempts to locate a contiguous range of unset bits.
// If ok is true, then all values in the range [index, index + n)
// are suitable for use in a call to v.Set.
// It calls Search if n <= 1.
func (v *V[T]) SearchRange(n int) (index int, ok bool) {
	if n <= 1 {
		return v.Search()
	}
	if v.Rem() < n {
		return
	}
	nb := v.nbit()
	var cnt, idx, bit, i int
	for {
		// Skip Uints that have no unset bits.
		if v.s[i] == ^T(0) {
			cnt, bit = 0, 0
			i++
			for ; i < len(v.s); i++ {
				if v.s[i] != ^T(0) {
					break
				}
			}
			idx = i
		}
		// Give up if there is not enough bits left.
		if cnt+nb*(len(v.s)-i) < n {
			return
		}
		// Iterate over whole Uints as much as possible.
		if v.s[i] == 0 {
			cnt += nb
			i++
			for j := 0; j < (n-cnt)/nb; j++ {
				if v.s[i+j] != 0 {
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
			if v.s[i]&(1<<j) == 0 {
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
		if i == len(v.s) {
			break
		}
	}
	return
}

// Clear unsets every bit in the vector.
func (v *V[T]) Clear() {
	n := v.Len()
	if n == v.Rem() {
		return
	}
	clear(v.s)
	v.rem = n
}

// All returns an iterator over all bits of the vector.
// The first value in the pair represents the index of the
// bit, while the second indicates whether the bit is set.
func (v *V[T]) All() iter.Seq2[int, bool] {
	return func(yield func(int, bool) bool) {
		n := v.nbit()
		for i, x := range v.s {
			for b := range n {
				if !yield(i*n+b, x&(1<<b) != 0) {
					return
				}
			}
		}
	}
}
