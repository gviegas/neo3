// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Data as presented to shader programs.
//
// The data layouts defined here represent exactly what
// will be fed to shaders as constant/uniform buffers.
// One should use the Set* methods of a given *Layout
// type to update constant data.
//
// Constants that are updated using vector and matrices
// (i.e., linear.V*/linear.M* types) will be defined in
// the shaders as equivalent types. These data will be
// aligned to 16 bytes for portability.
//
// TODO: Consider using arrays of integers, rather than
// floats, in the layout definitions.

package shader

import (
	"time"
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/linear"
)

func copyM4(dst []float32, m *linear.M4) {
	copy(dst, unsafe.Slice((*float32)(unsafe.Pointer(m)), 16))
}

// FrameLayout is the layout of per-frame, global data.
// It is defined as follows:
//
//	[0:16]  | view-projection matrix
//	[16:32] | view matrix
//	[32:48] | projection matrix
//	[48]    | elapsed time in seconds
//	[49]    | normalized random value
//	[50]    | viewport's x
//	[51]    | viewport's y
//	[52]    | viewport's width
//	[53]    | viewport's height
//	[54]    | viewport's near plane
//	[55]    | viewport's far plane
//	[56:64] | (unused)
//
// NOTE: This layout is likely to change.
type FrameLayout [64]float32

// SetVP sets the view-projection matrix.
func (l *FrameLayout) SetVP(m *linear.M4) { copyM4(l[:16], m) }

// SetV sets the view matrix.
func (l *FrameLayout) SetV(m *linear.M4) { copyM4(l[16:32], m) }

// SetP sets the projection matrix.
func (l *FrameLayout) SetP(m *linear.M4) { copyM4(l[32:48], m) }

// SetTime sets the elapsed time.
func (l *FrameLayout) SetTime(d time.Duration) { l[48] = float32(d.Seconds()) }

// SetRand sets the normalized random value.
func (l *FrameLayout) SetRand(rnd float32) { l[49] = rnd }

// SetBounds sets the viewport bounds.
func (l *FrameLayout) SetBounds(b *driver.Viewport) {
	l[50] = b.X
	l[51] = b.Y
	l[52] = b.Width
	l[53] = b.Height
	l[54] = b.Znear
	l[55] = b.Zfar
}

// LightLayout is the layout of light data.
// It is defined as follows:
//
//	[0]     | whether the light is unused
//	[1]     | light type
//	[2]     | intensity
//	[3]     | range
//	[4:7]   | color
//	[7]     | angular scale
//	[8:11]  | position
//	[11]    | angular offset
//	[12:15] | direction
//	[15]    | (unused)
type LightLayout [16]float32

// Types of light.
const (
	DirectLight int32 = iota
	PointLight
	SpotLight
)

// SetUnused sets whether the light is unused.
func (l *LightLayout) SetUnused(unused bool) {
	var bool32 int32
	if unused {
		bool32 = 1
	}
	l[0] = *(*float32)(unsafe.Pointer(&bool32))
}

// SetType sets the light type.
func (l *LightLayout) SetType(typ int32) { l[1] = *(*float32)(unsafe.Pointer(&typ)) }

// SetIntensity sets the intensity.
func (l *LightLayout) SetIntensity(i float32) { l[2] = i }

// SetRange sets the range.
// Used for PointLight and SpotLight.
func (l *LightLayout) SetRange(rng float32) { l[3] = rng }

// SetColor sets the color.
func (l *LightLayout) SetColor(c *linear.V3) { copy(l[4:7], c[:]) }

// SetAngScale sets the angular scale.
// Used for SpotLight.
func (l *LightLayout) SetAngScale(s float32) { l[7] = s }

// SetPosition sets the position.
// Used for PointLight and SpotLight.
func (l *LightLayout) SetPosition(p *linear.V3) { copy(l[8:11], p[:]) }

// SetAngOffset sets the angular offset.
// Used for SpotLight.
func (l *LightLayout) SetAngOffset(off float32) { l[11] = off }

// SetDirection sets the direction.
// Used for DirectLight and SpotLight.
func (l *LightLayout) SetDirection(d *linear.V3) { copy(l[12:15], d[:]) }

// DrawableLayout is the layout of drawable data.
// It is defined as follows:
//
//	[0:16]  | world matrix
//	[16:32] | normal matrix
//	[32:48] | ???
//	[48]    | ID
//	[49]	| ???
//	[50]    | ???
//	[51]    | ???
//	[52:63] | (unused)
//
// NOTE: This layout is likely to change.
type DrawableLayout [64]float32

// SetWorld sets the world matrix.
func (l *DrawableLayout) SetWorld(m *linear.M4) { copyM4(l[:16], m) }

// SetNormal sets the normal matrix.
func (l *DrawableLayout) SetNormal(m *linear.M4) { copyM4(l[16:32], m) }

// SetID sets the drawable's ID.
func (l *DrawableLayout) SetID(id uint32) { l[48] = *(*float32)(unsafe.Pointer(&id)) }

// MaterialLayout is the layout of material data.
// It is defined as follows:
//
//	[0:4]   | base color factor
//	[4]     | metalness
//	[5]     | roughness
//	[6]     | normal scale
//	[7]     | occlusion strength
//	[8:11]  | emissive factor
//	[11]    | alpha cutoff
//	[12]    | flags
//	[13:15] | (unused)
type MaterialLayout [16]float32

// Material flags.
const (
	// Identifies the default material model.
	MatPBR uint32 = 1 << iota
	// Identifies the unlit material model.
	MatUnlit
	// Alpha mode is material.AlphaOpaque.
	MatAOpaque
	// Alpha mode is material.AlphaBlend.
	MatABlend
	// Alpha mode is material.AlphaMask.
	MatAMask
	// Whether the material is double-sided.
	MatDoubleSided
)

// SetColorFactor sets the base color factor.
func (l *MaterialLayout) SetColorFactor(fac *linear.V4) { copy(l[:4], fac[:]) }

// SetMetalRough sets the metalness and roughness.
func (l *MaterialLayout) SetMetalRough(metal, rough float32) { l[4], l[5] = metal, rough }

// SetNormScale sets the normal scale.
func (l *MaterialLayout) SetNormScale(s float32) { l[6] = s }

// SetOccStrength sets the occlusion strength.
func (l *MaterialLayout) SetOccStrength(s float32) { l[7] = s }

// SetEmisFactor sets the emissive factor.
func (l *MaterialLayout) SetEmisFactor(fac *linear.V3) { copy(l[8:11], fac[:]) }

// SetAlphaCutoff sets the alpha cutoff value.
// Used for AlphaMask.
func (l *MaterialLayout) SetAlphaCutoff(c float32) { l[11] = c }

// SetFlags sets the material flags.
func (l *MaterialLayout) SetFlags(flg uint32) { l[12] = *(*float32)(unsafe.Pointer(&flg)) }

// JointLayout is the layout of joint data.
// It is defined as follows:
//
//	[0:16]  | joint matrix
//	[16:32] | normal matrix
type JointLayout [32]float32

// SetJoint sets the joint matrix.
func (l *JointLayout) SetJoint(m *linear.M4) { copyM4(l[:16], m) }

// SetNormal sets the normal matrix.
func (l *JointLayout) SetNormal(m *linear.M4) { copyM4(l[16:32], m) }