// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"errors"
	"strconv"
)

func newErr(reason string) error {
	return errors.New("gltf: " + reason)
}

// Check checks that f is a valid glTF object.
// TODO
func (f *GLTF) Check() error {
	vers, err := strconv.ParseFloat(f.Asset.Version, 64)
	if err != nil {
		return newErr("invalid GLTF.Asset.Version string")
	}
	minVers, err := strconv.ParseFloat(f.Asset.MinVersion, 64)
	if err == nil && minVers >= 3 {
		return newErr("unsupported GLTF.Asset.MinVersion")
	} else if vers < 2 || vers >= 3 {
		return newErr("unsupported GLTF.Asset.Version")
	}

	if s := f.Scene; s != nil && (*s < 0 || *s >= int64(len(f.Nodes))) {
		return newErr("invalid GLTF.Scene index")
	}

	for i := range f.Accessors {
		if err := f.Accessors[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Animations {
		if err := f.Animations[i].Check(f); err != nil {
			return err
		}
	}
	return nil
}

// Check checks that a is a valid glTF.accessors element.
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

// Check checks that a is a valid glTF.animations element.
func (a *Animation) Check(gltf *GLTF) error {
	if len(a.Channels) == 0 {
		return newErr("invalid Animation.Channels length")
	}
	if len(a.Samplers) == 0 {
		return newErr("invalid Animation.Samplers length")
	}

	for i := range a.Channels {
		c := &a.Channels[i]
		if c.Sampler < 0 || c.Sampler >= int64(len(a.Samplers)) {
			return newErr("invalid Animation.Channels[].Sampler index")
		}
		if c.Target.Node != nil {
			nd := *c.Target.Node
			if nd < 0 || nd >= int64(len(gltf.Nodes)) {
				return newErr("invalid Animation.Channels[].Target.Node index")
			}
		}
		switch c.Target.Path {
		case Ptranslation, Protation, Pscale, Pweights:
		default:
			return newErr("invalid Animation.Channels[].Target.Path value")
		}
	}

	for i := range a.Samplers {
		s := &a.Samplers[i]
		if s.Input < 0 || s.Input >= int64(len(gltf.Accessors)) {
			return newErr("invalid Animation.Samplers[].Input index")
		}
		switch s.Interpolation {
		case ILINEAR, STEP, CUBICSPLINE:
		default:
			return newErr("invalid Animation.Samplers[].Interpolation value")
		}
		if s.Output < 0 || s.Output >= int64(len(gltf.Accessors)) {
			return newErr("invalid Animation.Samplers[].Output index")
		}
	}
	return nil
}
