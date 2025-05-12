// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"

	"gviegas/neo3/engine/internal/shader"
	"gviegas/neo3/linear"
)

const matPrefix = "material: "

func newMatErr(reason string) error { return errors.New(matPrefix + reason) }

// Material defines the material properties to be applied
// to geometry during rendering.
type Material struct {
	baseColor  TexRef
	metalRough TexRef
	normal     TexRef
	occlusion  TexRef
	emissive   TexRef
	layout     shader.MaterialLayout

	// TODO: Descriptors; const buffer.
}

// TexRef identifies a particular view of a 2D texture
// and its sampler, with sampling operations using a
// given UV set.
type TexRef struct {
	Texture *Texture
	View    int
	Sampler *Sampler
	UVSet   int
}

// UV sets matching TexCoord* semantics.
const (
	// TexCoord0.
	UVSet0 = iota
	// TexCoord1.
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

// NormalMap is the material's normal map.
type NormalMap struct {
	TexRef
	Scale float32
}

// OcclusionMap is the material's occlusion map.
type OcclusionMap struct {
	TexRef
	Strength float32
}

// EmissiveMap is the material's emissive map.
type EmissiveMap struct {
	TexRef
	Factor [3]float32
}

// Alpha modes.
const (
	// No transparency.
	// Alpha channel is unconditionally set to 1.0.
	AlphaOpaque = iota
	// Composition with background.
	AlphaBlend
	// Either fully opaque or fully transparent,
	// as determined by a cutoff value.
	AlphaMask
)

// PBR defines properties of the default material model.
type PBR struct {
	BaseColor   BaseColor
	MetalRough  MetalRough
	Normal      NormalMap
	Occlusion   OcclusionMap
	Emissive    EmissiveMap
	AlphaMode   int
	AlphaCutoff float32
	DoubleSided bool
}

// shaderLayout creates the shader.MaterialLayout of p.
// It assumes that p is valid.
func (p *PBR) shaderLayout() (l shader.MaterialLayout) {
	l.SetColorFactor((*linear.V4)(&p.BaseColor.Factor))
	l.SetMetalRough(p.MetalRough.Metalness, p.MetalRough.Roughness)
	l.SetNormScale(p.Normal.Scale)
	l.SetOccStrength(p.Occlusion.Strength)
	l.SetEmisFactor((*linear.V3)(&p.Emissive.Factor))
	l.SetAlphaCutoff(p.AlphaCutoff)
	flags := shader.MatPBR
	switch p.AlphaMode {
	case AlphaOpaque:
		flags |= shader.MatAOpaque
	case AlphaBlend:
		flags |= shader.MatABlend
	case AlphaMask:
		flags |= shader.MatAMask
	}
	if p.DoubleSided {
		flags |= shader.MatDoubleSided
	}
	l.SetFlags(flags)
	return
}

// Unlit defines properties of the unlit material model.
type Unlit struct {
	BaseColor   BaseColor
	AlphaMode   int
	AlphaCutoff float32
	DoubleSided bool
}

// shaderLayout creates the shader.MaterialLayout of u.
// It assumes that u is valid.
func (u *Unlit) shaderLayout() (l shader.MaterialLayout) {
	l.SetColorFactor((*linear.V4)(&u.BaseColor.Factor))
	l.SetAlphaCutoff(u.AlphaCutoff)
	flags := shader.MatUnlit
	switch u.AlphaMode {
	case AlphaOpaque:
		flags |= shader.MatAOpaque
	case AlphaBlend:
		flags |= shader.MatABlend
	case AlphaMask:
		flags |= shader.MatAMask
	}
	if u.DoubleSided {
		flags |= shader.MatDoubleSided
	}
	l.SetFlags(flags)
	return
}

// NewPBR creates a new material using the default model.
func NewPBR(prop *PBR) (*Material, error) {
	if err := prop.validate(); err != nil {
		return nil, err
	}
	return &Material{
		baseColor:  prop.BaseColor.TexRef,
		metalRough: prop.MetalRough.TexRef,
		normal:     prop.Normal.TexRef,
		occlusion:  prop.Occlusion.TexRef,
		emissive:   prop.Emissive.TexRef,
		layout:     prop.shaderLayout(),
	}, nil
}

// NewUnlit creates a new material using the unlit model.
func NewUnlit(prop *Unlit) (*Material, error) {
	if err := prop.validate(); err != nil {
		return nil, err
	}
	return &Material{
		baseColor: prop.BaseColor.TexRef,
		layout:    prop.shaderLayout(),
	}, nil
}

// Parameter validation for New* functions.

func (p *TexRef) validate(optional bool) error {
	// TODO: Should ensure somehow that the Texture
	// was created by a call to New2D
	// (it is fine to ignore this for now because
	// all textures support driver.UShaderSample).
	if p.Texture == nil {
		if optional {
			return nil
		}
		return newMatErr("nil TexRef.Texture")
	}
	if !p.Texture.IsValidView(p.View) {
		return newMatErr("invalid TexRef.View")
	}
	if !p.Texture.PixelFmt().IsColor() {
		return newMatErr("TexRef.Texture has non-color format")
	}
	if p.Sampler == nil {
		return newMatErr("nil TexRef.Sampler")
	}
	switch p.UVSet {
	case UVSet0, UVSet1:
	default:
		return newMatErr("undefined UV set constant")
	}
	return nil
}

func (p *BaseColor) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	for _, x := range p.Factor {
		if x < 0 || x > 1 {
			return newMatErr("BaseColor.Factor outside [0.0, 1.0] interval")
		}
	}
	return nil
}

func (p *MetalRough) validate() error {
	if p.Texture != nil {
		if err := p.TexRef.validate(false); err != nil {
			return err
		}
		if p.Texture.PixelFmt().Channels() < 2 {
			return newMatErr("MetalRough.Texture has insufficient channels")
		}
	}
	if p.Metalness < 0 || p.Metalness > 1 {
		return newMatErr("MetalRough.Metalness outside [0.0, 1.0] interval")
	}
	if p.Roughness < 0 || p.Roughness > 1 {
		return newMatErr("MetalRough.Roughness outside [0.0, 1.0] interval")
	}
	return nil
}

func (p *NormalMap) validate() error {
	if p.Texture != nil {
		if err := p.TexRef.validate(false); err != nil {
			return err
		}
		if p.Texture.PixelFmt().Channels() < 3 {
			return newMatErr("NormalMap.Texture has insufficient channels")
		}
	}
	if p.Scale < 0 {
		return newMatErr("NormalMap.Scale less than 0.0")
	}
	return nil
}

func (p *OcclusionMap) validate() error {
	if err := p.TexRef.validate(true); err != nil {
		return err
	}
	if p.Strength < 0 || p.Strength > 1 {
		return newMatErr("OcclusionMap.Strength outside [0.0, 1.0] interval")
	}
	return nil
}

func (p *EmissiveMap) validate() error {
	if p.Texture != nil {
		if err := p.TexRef.validate(false); err != nil {
			return err
		}
		if p.Texture.PixelFmt().Channels() < 3 {
			return newMatErr("EmissiveMap.Texture has insufficient channels")
		}
	}
	for _, x := range p.Factor {
		if x < 0 || x > 1 {
			return newMatErr("EmissiveMap.Factor outside [0.0, 1.0] interval")
		}
	}
	return nil
}

func validateAlphaMode(mode int, cutoff float32) error {
	switch mode {
	case AlphaOpaque, AlphaBlend:
	case AlphaMask:
		// Don't restrict cutoff values,
		// even if they don't make sense.
		_ = cutoff
	default:
		return newMatErr("undefined alpha mode constant")
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
	if err := validateAlphaMode(p.AlphaMode, p.AlphaCutoff); err != nil {
		return err
	}
	return nil
}

func (p *Unlit) validate() error {
	if err := p.BaseColor.validate(); err != nil {
		return err
	}
	if err := validateAlphaMode(p.AlphaMode, p.AlphaCutoff); err != nil {
		return err
	}
	return nil
}
