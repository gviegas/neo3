// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

// keyFrom returns the Key value that represents an
// OS-specific key code.
// It assumes that code is greater than or equal to
// zero.
// Every supported system must provide an indexable
// var named keymap that contains Key values.
//
// NOTE: keymap must be indexable by non-negative
// values up to its length minus one.
func keyFrom(code int) Key {
	if code >= len(keymap) {
		return KeyUnknown
	}
	return keymap[code]
}
