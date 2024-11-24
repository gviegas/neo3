// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Data as presented to shader programs.
//
// The data layouts defined here represent exactly what
// will be fed to shaders as constant/uniform buffers.
// One should use the Set* methods of a given *Layout
// type to update constant data.
//
// Constants that are updated using vectors or matrices
// (i.e., linear.V*/linear.M* types) will be defined in
// the shaders as equivalent types. These data will be
// aligned to 16 bytes for portability.
//
// TODO: Consider using arrays of integers, rather than
// floats, in the layout definitions.
//
// TODO: Missing getters.

package shader

import (
	"time"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/linear"
)

func copyM4(dst []float32, m *linear.M4) {
	copy(dst, unsafe.Slice((*float32)(unsafe.Pointer(m)), 16))
}

func copyM3(dst []float32, m *linear.M3) {
	// The columns themselves must be 16-byte aligned.
	copy(dst[0:], m[0][:])
	copy(dst[4:], m[1][:])
	copy(dst[8:], m[2][:])
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

// VP returns the view-projection matrix.
func (l *FrameLayout) VP() (m linear.M4) {
	for i := range m {
		copy(m[i][:], l[4*i:4*i+4])
	}
	return
}

// SetV sets the view matrix.
func (l *FrameLayout) SetV(m *linear.M4) { copyM4(l[16:32], m) }

// V returns the view matrix.
func (l *FrameLayout) V() (m linear.M4) {
	for i := range m {
		copy(m[i][:], l[16+4*i:16+4*i+4])
	}
	return
}

// SetP sets the projection matrix.
func (l *FrameLayout) SetP(m *linear.M4) { copyM4(l[32:48], m) }

// P returns the projection matrix.
func (l *FrameLayout) P() (m linear.M4) {
	for i := range m {
		copy(m[i][:], l[32+4*i:32+4*i+4])
	}
	return
}

// SetTime sets the elapsed time.
func (l *FrameLayout) SetTime(d time.Duration) { l[48] = float32(d.Seconds()) }

// Time returns the elapsed time.
func (l *FrameLayout) Time() time.Duration {
	nsec := int64(float64(l[48]) * 1e9)
	return time.Duration(nsec)
}

// SetRand sets the normalized random value.
func (l *FrameLayout) SetRand(rnd float32) { l[49] = rnd }

// Rand returns the normalized random value.
func (l *FrameLayout) Rand() float32 { return l[49] }

// SetBounds sets the viewport bounds.
func (l *FrameLayout) SetBounds(b *driver.Viewport) {
	l[50] = b.X
	l[51] = b.Y
	l[52] = b.Width
	l[53] = b.Height
	l[54] = b.Znear
	l[55] = b.Zfar
}

// Bounds returns the viewport bounds.
func (l *FrameLayout) Bounds() driver.Viewport {
	return driver.Viewport{
		X:      l[50],
		Y:      l[51],
		Width:  l[52],
		Height: l[53],
		Znear:  l[54],
		Zfar:   l[55],
	}
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
	DistantLight int32 = iota
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

// Unused returns whether the light is unused.
func (l *LightLayout) Unused() bool {
	unused := *(*int32)(unsafe.Pointer(&l[0]))
	return unused == 1
}

// SetType sets the light type.
func (l *LightLayout) SetType(typ int32) { l[1] = *(*float32)(unsafe.Pointer(&typ)) }

// Type returns the light type.
func (l *LightLayout) Type() int32 { return *(*int32)(unsafe.Pointer(&l[1])) }

// SetIntensity sets the intensity.
func (l *LightLayout) SetIntensity(i float32) { l[2] = i }

// Intensity returns the intensity.
func (l *LightLayout) Intensity() float32 { return l[2] }

// SetRange sets the range.
// Used for PointLight and SpotLight.
func (l *LightLayout) SetRange(rng float32) { l[3] = rng }

// Range returns the range.
// Used for PointLight and SpotLight.
func (l *LightLayout) Range() float32 { return l[3] }

// SetColor sets the color.
func (l *LightLayout) SetColor(c *linear.V3) { copy(l[4:7], c[:]) }

// Color returns the color.
func (l *LightLayout) Color() linear.V3 { return linear.V3(l[4:7]) }

// SetAngScale sets the angular scale.
// Used for SpotLight.
func (l *LightLayout) SetAngScale(s float32) { l[7] = s }

// AngScale returns the angular scale.
// Used for SpotLight.
func (l *LightLayout) AngScale() float32 { return l[7] }

// SetPosition sets the position.
// Used for PointLight and SpotLight.
func (l *LightLayout) SetPosition(p *linear.V3) { copy(l[8:11], p[:]) }

// Position returns the position.
// Used for PointLight and SpotLight.
func (l *LightLayout) Position() linear.V3 { return linear.V3(l[8:11]) }

// SetAngOffset sets the angular offset.
// Used for SpotLight.
func (l *LightLayout) SetAngOffset(off float32) { l[11] = off }

// AngOffset returns the angular offset.
// Used for SpotLight.
func (l *LightLayout) AngOffset() float32 { return l[11] }

// SetDirection sets the direction.
// Used for DistantLight and SpotLight.
func (l *LightLayout) SetDirection(d *linear.V3) { copy(l[12:15], d[:]) }

// Direction returns the direction.
// Used for DistantLight and SpotLight.
func (l *LightLayout) Direction() linear.V3 { return linear.V3(l[12:15]) }

// ShadowLayout is the layout of shadow data.
// It is defined as follows:
//
//	[0]     | whether the shadow is unused
//	[1:16]  | ???
//	[16:32] | shadow matrix
//
// NOTE: This layout is likely to change.
type ShadowLayout [32]float32

// SetUnused sets whether the shadow is unused.
func (l *ShadowLayout) SetUnused(unused bool) {
	var bool32 int32
	if unused {
		bool32 = 1
	}
	l[0] = *(*float32)(unsafe.Pointer(&bool32))
}

// Unused returns whether the shadow is unused.
func (l *ShadowLayout) Unused() bool {
	unused := *(*int32)(unsafe.Pointer(&l[0]))
	return unused == 1
}

// SetShadow sets the shadow matrix.
func (l *ShadowLayout) SetShadow(m *linear.M4) { copyM4(l[16:32], m) }

// Shadow returns the shadow matrix.
func (l *ShadowLayout) Shadow() (m linear.M4) {
	for i := range m {
		copy(m[i][:], l[16+4*i:16+4*i+4])
	}
	return
}

// DrawableLayout is the layout of drawable data.
// It is defined as follows:
//
//	[0:16]  | world matrix
//	[16:28] | normal matrix (padded columns)
//	[28]    | ID
//	[29]    | ???
//	[30]    | ???
//	[31]    | ???
//	[32:63] | (unused)
//
// NOTE: This layout is likely to change.
type DrawableLayout [64]float32

// SetWorld sets the world matrix.
func (l *DrawableLayout) SetWorld(m *linear.M4) { copyM4(l[:16], m) }

// World returns the world matrix.
func (l *DrawableLayout) World() (m linear.M4) {
	for i := range m {
		copy(m[i][:], l[4*i:4*i+4])
	}
	return
}

// SetNormal sets the normal matrix.
func (l *DrawableLayout) SetNormal(m *linear.M3) { copyM3(l[16:28], m) }

// Normal returns the normal matrix.
func (l *DrawableLayout) Normal() (m linear.M3) {
	for i := range m {
		copy(m[i][:], l[16+4*i:16+4*i+3])
	}
	return
}

// SetID sets the drawable's ID.
func (l *DrawableLayout) SetID(id uint32) { l[28] = *(*float32)(unsafe.Pointer(&id)) }

// ID returns the drawable's ID.
func (l *DrawableLayout) ID() uint32 {
	id := *(*uint32)(unsafe.Pointer(&l[28]))
	return id
}

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

// ColorFactor returns the base color factor.
func (l *MaterialLayout) ColorFactor() (fac linear.V4) {
	copy(fac[:], l[:4])
	return
}

// SetMetalRough sets the metalness and roughness.
func (l *MaterialLayout) SetMetalRough(metal, rough float32) { l[4], l[5] = metal, rough }

// MetalRough returns the metalness and roughness.
func (l *MaterialLayout) MetalRough() (metal, rough float32) {
	metal = l[4]
	rough = l[5]
	return
}

// SetNormScale sets the normal scale.
func (l *MaterialLayout) SetNormScale(s float32) { l[6] = s }

// NormScale returns the normal scale.
func (l *MaterialLayout) NormScale() float32 { return l[6] }

// SetOccStrength sets the occlusion strength.
func (l *MaterialLayout) SetOccStrength(s float32) { l[7] = s }

// OccStrength returns the occlusion strength.
func (l *MaterialLayout) OccStrength() float32 { return l[7] }

// SetEmisFactor sets the emissive factor.
func (l *MaterialLayout) SetEmisFactor(fac *linear.V3) { copy(l[8:11], fac[:]) }

// EmisFactor returns the emissive factor.
func (l *MaterialLayout) EmisFactor() (fac linear.V3) {
	copy(fac[:], l[8:11])
	return
}

// SetAlphaCutoff sets the alpha cutoff value.
// Used for AlphaMask.
func (l *MaterialLayout) SetAlphaCutoff(c float32) { l[11] = c }

// AlphaCutoff returns the alpha cutoff.
// Used for AlphaMask.
func (l *MaterialLayout) AlphaCutoff() float32 { return l[11] }

// SetFlags sets the material flags.
func (l *MaterialLayout) SetFlags(flg uint32) { l[12] = *(*float32)(unsafe.Pointer(&flg)) }

// Flags returns the material flags.
func (l *MaterialLayout) Flags() uint32 {
	flg := *(*uint32)(unsafe.Pointer(&l[12]))
	return flg
}

// JointLayout is the layout of joint data.
// It is defined as follows:
//
//	[0:12]  | joint matrix (1st, 2nd and 3rd rows)
//	[12:24] | normal matrix (padded columns)
type JointLayout [24]float32

// SetJoint sets the joint matrix.
func (l *JointLayout) SetJoint(m *linear.M4) {
	var n linear.M4
	n.Transpose(m)
	copy(l[:12], unsafe.Slice((*float32)(unsafe.Pointer(&n)), 12))
}

// SetNormal sets the normal matrix.
func (l *JointLayout) SetNormal(m *linear.M3) { copyM3(l[12:24], m) }

// Constants defining the maximum number of elements
// for layouts that are aggregated into static arrays.
//
// TODO: Make these configurable.
const (
	MaxLight  = 16384 / unsafe.Sizeof(LightLayout{})
	MaxShadow = 1
	MaxJoint  = 16384 / unsafe.Sizeof(JointLayout{})
)
