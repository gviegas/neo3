// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

// keyFrom returns the Key value that represents an
// OS-specific key code.
// Every supported system must provide an indexable
// var named keymap that contains Key values.
//
// Note: If keymap is implemented as a map type,
// its length must be greater than the maximum
// key code value. Also, do not implement keymap
// as a map type.
func keyFrom(code int) Key {
	if code >= len(keymap) {
		return KeyUnknown
	}
	return keymap[code]
}
