// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package material implements the material model used in
// the engine.
package material

import (
	"errors"

	"github.com/gviegas/scene/engine/texture"
)

const prefix = "material: "

// Material.
type Material struct {
	// *PBR or *Unlit.
	// Other models may yet be added.
	prop any

	// TODO: Descriptors; const buffer.
}

// TexRef identifies a particular view of a 2D texture
// and its sampler, with sampling operations using a
// given UV set.
type TexRef struct {
	Texture *texture.Texture
	View    int
	Sampler *texture.Sampler
	UVSet   int
}

// UV sets matching mesh.TexCoord* semantics.
const (
	// mesh.TexCoord0.
	UVSet0 = iota
	// mesh.TexCoord1.
	UVSet1
)

// BaseColor is the material's base color.
type BaseColor struct {
	TexRef
	Factor [4]float32
}

// MetalRough is the material's matallic-roughness.
type MetalRough struct {
	TexRef
	Metalness float32
	Roughness float32
}

// Normal is the material's normal map.
type Normal struct {
	TexRef
	Scale float32
}

// Occlusion is the material's occlusion map.
type Occlusion struct {
	TexRef
	Strength float32
}

// Emissive is the material's emissive map.
type Emissive struct {
	TexRef
	Factor [3]float32
}

// Alpha modes.
const (
	AlphaOpaque = iota
	AlphaBlend
	AlphaMask
)

// PBR defines properties of the default material model.
type PBR struct {
	BaseColor   BaseColor
	MetalRough  MetalRough
	Normal      Normal
	Occlusion   Occlusion
	Emissive    Emissive
	AlphaMode   int
	AlphaCutoff float32
	DoubleSided bool
}

// Unlit defines properties of the unlit material model.
type Unlit struct {
	BaseColor   BaseColor
	AlphaMode   int
	AlphaCutoff float32
	DoubleSided bool
}

// New creates a new material using the default model.
func New(prop *PBR) (m *Material, err error) {
	if err = prop.validate(); err != nil {
		return
	}
	p := new(PBR)
	*p = *prop
	m = &Material{p}
	return
}

// NewUnlit creates a new material using the unlit model.
func NewUnlit(prop *Unlit) (m *Material, err error) {
	if err = prop.validate(); err != nil {
		return
	}
	p := new(Unlit)
	*p = *prop
	m = &Material{p}
	return
}

// Parameter validation for New* functions.

func newErr(reason string) error { return errors.New(prefix + reason) }

func (p *TexRef) validate(optional bool) error {
	// TODO: Should ensure somehow that the Texture
	// was created by a call to texture.New2D
	// (it is fine to ignore this for now because
	// all textures support driver.UShaderSample).
	if p.Texture == nil {
		if optional {
			return nil
		}
		return newErr("nil TexRef.Texture")
	}
	if !p.Texture.IsValidView(p.View) {
		return newErr("invalid TexRef.View")
	}
	if !p.Texture.PixelFmt().IsColor() {
		return newErr("TexRef.Texture has non-color format")
	}
	if p.Sampler == nil {
		return newErr("nil TexRef.Sampler")
	}
	switch p.UVSet {
	case UVSet0, UVSet1:
	default:
		return newErr("undefined UV set constant")
	}
	return nil
}

func (p *BaseColor) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	for _, x := range p.Factor {
		if x < 0 || x > 1 {
			return newErr("BaseColor.Factor outside [0.0, 1.0] interval")
		}
	}
	return nil
}

func (p *MetalRough) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	if p.Metalness < 0 || p.Metalness > 1 {
		return newErr("MetalRough.Metalness outside [0.0, 1.0] interval")
	}
	if p.Roughness < 0 || p.Roughness > 1 {
		return newErr("MetalRough.Roughness outside [0.0, 1.0] interval")
	}
	return nil
}

func (p *Normal) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	if p.Scale < 0 {
		return newErr("Normal.Scale less than 0.0")
	}
	return nil
}

func (p *Occlusion) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	if p.Strength < 0 || p.Strength > 1 {
		return newErr("Occlusion.Strength outside [0.0, 1.0] interval")
	}
	return nil
}

func (p *Emissive) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	for _, x := range p.Factor {
		if x < 0 || x > 1 {
			return newErr("Emissive.Factor outside [0.0, 1.0] interval")
		}
	}
	return nil
}

func validateAlpha(mode int, cutoff float32) error {
	switch mode {
	case AlphaOpaque, AlphaBlend:
	case AlphaMask:
		// Don't restrict cutoff values,
		// even if they don't make sense.
	default:
		return newErr("undefined alpha mode constant")
	}
	return nil
}

func (p *PBR) validate() error {
	if err := p.BaseColor.validate(); err != nil {
		return err
	}
	if err := p.MetalRough.validate(); err != nil {
		return err
	}
	if err := p.Normal.validate(); err != nil {
		return err
	}
	if err := p.Occlusion.validate(); err != nil {
		return err
	}
	if err := p.Emissive.validate(); err != nil {
		return err
	}
	if err := validateAlpha(p.AlphaMode, p.AlphaCutoff); err != nil {
		return err
	}
	return nil
}

func (p *Unlit) validate() error {
	if err := p.BaseColor.validate(); err != nil {
		return err
	}
	if err := validateAlpha(p.AlphaMode, p.AlphaCutoff); err != nil {
		return err
	}
	return nil
}
