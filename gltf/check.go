// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"errors"
)

func newErr(reason string) error {
	return errors.New("gltf: " + reason)
}

// Check checks that f is valid glTF.
// TODO
func (f *GLTF) Check() error {
	if s := f.Scene; s != nil && (*s < 0 || *s >= int64(len(f.Nodes))) {
		return newErr("invalid GLTF.Scene index")
	}
	return nil
}
