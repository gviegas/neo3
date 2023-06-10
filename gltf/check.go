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
	for _, a := range f.Accessors {
		if err := a.Check(f); err != nil {
			return err
		}
	}
	return nil
}

// Check checks that a is valid glTF.accessors' element.
func (a *Accessor) Check(gltf *GLTF) error {
	if a.BufferView != nil {
		idx := *a.BufferView
		if idx < 0 || idx > int64(len(gltf.BufferViews)) {
			return newErr("invalid Accessor.BufferView index")
		}
	}
	if a.ByteOffset < 0 { // TODO: Check upper bound.
		return newErr("invalid Accessor.BufferOffset value")
	}
	switch a.ComponentType {
	case BYTE, UNSIGNED_BYTE, SHORT, UNSIGNED_SHORT, UNSIGNED_INT, FLOAT:
	default:
		return newErr("invalid Accessor.ComponentType value")
	}
	if a.Count < 1 {
		return newErr("invalid Accessor.Count value")
	}
	switch a.Type {
	case SCALAR, VEC2, VEC3, VEC4, MAT2, MAT3, MAT4:
	default:
		return newErr("invalid Accessor.Type value")
	}
	// TODO: Check Accessor.Max/Min.

	if s := a.Sparse; s != nil {
		if s.Count < 1 || s.Count > a.Count {
			return newErr("invalid Accessor.Sparse.Count value")
		}

		if s.Indices.BufferView < 0 || s.Indices.BufferView > int64(len(gltf.BufferViews)) {
			return newErr("invalid Accessor.Sparse.Indices.BufferView index")
		}
		if s.Indices.ByteOffset < 0 { // TODO: Check upper bound.
			return newErr("invalid Accessor.Sparse.Indices.ByteOffset value")
		}
		switch s.Indices.ComponentType {
		case UNSIGNED_BYTE, UNSIGNED_SHORT, UNSIGNED_INT:
		default:
			return newErr("invalid Accessor.Sparse.Indices.ComponentType value")
		}

		if s.Values.BufferView < 0 || s.Values.BufferView > int64(len(gltf.BufferViews)) {
			return newErr("invalid Accessor.Sparse.Values.BufferView index")
		}
		if s.Values.ByteOffset < 0 { // TODO: Check upper bound.
			return newErr("invalid Accessor.Sparse.Values.ByteOffset value")
		}
	}
	return nil
}
