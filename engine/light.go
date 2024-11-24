// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"math"

	"gviegas/neo3/engine/internal/shader"
	"gviegas/neo3/linear"
)

const (
	distantLight = iota
	pointLight
	spotLight
)

// Light defines a light source.
// The zero value for Light is not valid; one must
// call DistantLight.Light, PointLight.Light or
// SpotLight.Light to create an initialized Light.
type Light struct {
	typ    int
	layout shader.LightLayout
	// Used to reconstruct the inner/outer
	// cone angles.
	// Ignored if typ is not spotLight.
	cosOuter float32
}

// SetDirection sets the direction of l.
// It does not normalize d.
// Only applies to distant and spot lights.
func (l *Light) SetDirection(d *linear.V3) { l.layout.SetDirection(d) }

// Direction returns the direction of l.
// Only applies to distant and spot lights.
func (l *Light) Direction() linear.V3 { return l.layout.Direction() }

// SetPosition sets the position of l.
// Only applies to point and spot lights.
func (l *Light) SetPosition(p *linear.V3) { l.layout.SetPosition(p) }

// Position returns the position of l.
// Only applies to point and spot lights.
func (l *Light) Position() linear.V3 { return l.layout.Position() }

// SetIntensity sets the intensity of l.
func (l *Light) SetIntensity(i float32) { l.layout.SetIntensity(max(0, i)) }

// Intensity returns the intensity of l.
func (l *Light) Intensity() float32 { return l.layout.Intensity() }

// SetRange sets the falloff range of l.
// Only applies to point and spot lights.
func (l *Light) SetRange(r float32) { l.layout.SetRange(r) }

// Range returns the falloff range of l.
// Only applies to point and spot lights.
func (l *Light) Range() float32 { return l.layout.Range() }

// SetColor sets the RGB color of l.
func (l *Light) SetColor(r, g, b float32) { l.layout.SetColor(&linear.V3{r, g, b}) }

// Color returns the RGB color of l.
func (l *Light) Color() (r, g, b float32) {
	rgb := l.layout.Color()
	r, g, b = rgb[0], rgb[1], rgb[2]
	return
}

// SetConeAngles sets the inner/outer cone angles of l.
// Cone angles that exceed math.Pi/2, or that are less
// than zero, will be clamped. The inner angle will be
// adjusted such that it is less than the outer angle.
// Only applies to spot lights.
func (l *Light) SetConeAngles(inner, outer float32) {
	var (
		i      = max(0, min(float64(inner), math.Pi/2-1e-6))
		o      = max(i+1e-6, min(float64(outer), math.Pi/2))
		cosi   = math.Cos(i)
		coso   = math.Cos(o)
		scale  = 1 / (cosi - coso)
		offset = scale * -coso
	)
	l.layout.SetAngScale(float32(scale))
	l.layout.SetAngOffset(float32(offset))
	l.cosOuter = float32(coso)
}

// ConeAngles returns the inner/outer cone angles of l.
// Note that it returns the clamped angles (see the doc
// for Light.SetConeAngles).
// Only applies to spot lights.
func (l *Light) ConeAngles() (inner, outer float32) {
	scale := l.layout.AngScale()
	coso := l.cosOuter
	cosi := (1 / scale) + coso
	inner = float32(math.Acos(float64(cosi)))
	outer = float32(math.Acos(float64(coso)))
	return
}

// DistantLight is a directional light.
// The light is emitted in the given Direction.
// It behaves as if located infinitely far way.
// Intensity is the illuminance in lux.
type DistantLight struct {
	Direction linear.V3
	Intensity float32
	R, G, B   float32
}

// Light creates the light source described by t.
// t.Direction must have length 1.
// t.R/G/B must be in the range [0, 1].
func (t *DistantLight) Light() (light Light) {
	light.typ = distantLight
	light.layout.SetType(shader.DistantLight)
	light.SetIntensity(t.Intensity)
	light.SetColor(t.R, t.G, t.B)
	light.SetDirection(&t.Direction)
	return
}

// PointLight is an omnidirectional, positional light.
// The light is emitted in all directions from the
// given Position.
// Range determines the area affected by the light.
// Intensity is the luminous intensity in candela.
type PointLight struct {
	Position  linear.V3
	Range     float32
	Intensity float32
	R, G, B   float32
}

// Light creates the light source described by t.
// t.R/G/B must be in the range [0, 1].
// t.Range may be set to 0 or less to indicate an
// infinite range.
func (t *PointLight) Light() (light Light) {
	light.typ = pointLight
	light.layout.SetType(shader.PointLight)
	light.SetIntensity(t.Intensity)
	light.SetRange(t.Range)
	light.SetColor(t.R, t.G, t.B)
	light.SetPosition(&t.Position)
	return
}

// SpotLight is a directional, positional light.
// The light is emitted in a cone in the given Direction
// from the given Position.
// InnerAngle and OuterAngle (in radians), alongside
// Range, determine the area affected by the light.
// Intensity is the luminous intensity in candela.
type SpotLight struct {
	Direction  linear.V3
	Position   linear.V3
	InnerAngle float32
	OuterAngle float32
	Range      float32
	Intensity  float32
	R, G, B    float32
}

// Light creates the light source described by t.
// t.Direction must have length 1.
// t.R/G/B must be in the range [0, 1].
// t.Range may be set to 0 or less to indicate an
// infinite range.
// The cone angles will be adjusted as per
// Light.SetConeAngles.
func (t *SpotLight) Light() (light Light) {
	light.typ = spotLight
	light.layout.SetType(shader.SpotLight)
	light.SetIntensity(t.Intensity)
	light.SetRange(t.Range)
	light.SetColor(t.R, t.G, t.B)
	light.SetConeAngles(t.InnerAngle, t.OuterAngle)
	light.SetPosition(&t.Position)
	light.SetDirection(&t.Direction)
	return
}
