// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"math"

	"gviegas/neo3/engine/internal/shader"
	"gviegas/neo3/linear"
)

const (
	sunLight = iota
	pointLight
	spotLight
)

// Light defines a light source.
// The zero value for Light is not valid; one must
// call SunLight.Light, PointLight.Light or
// SpotLight.Light to create an initialized Light.
type Light struct {
	typ    int
	layout shader.LightLayout
}

// SetDirection sets the direction of l.
// It does not normalize d.
// Only applies to sun and spot lights.
func (l *Light) SetDirection(d *linear.V3) { l.layout.SetDirection(d) }

// SetPosition sets the position of l.
// Only applies to point and spot lights.
func (l *Light) SetPosition(p *linear.V3) { l.layout.SetPosition(p) }

// SetIntensity sets the intensity of l.
func (l *Light) SetIntensity(i float32) { l.layout.SetIntensity(max(0, i)) }

// SetRange sets the falloff range of l.
// Only applies to point and spot lights.
func (l *Light) SetRange(r float32) { l.layout.SetRange(r) }

// SetColor sets the RGB color of l.
func (l *Light) SetColor(r, g, b float32) { l.layout.SetColor(&linear.V3{r, g, b}) }

// SetConeAngles sets the inner/outer cone angles of l.
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
}

// SunLight is a directional light.
// The light is emitted in the given Direction.
// It behaves as if located infinitely far way.
// Intensity is the illuminance in lux.
type SunLight struct {
	Direction linear.V3
	Intensity float32
	R, G, B   float32
}

// Light creates the light source described by t.
// t.Direction must have length 1.
// t.R/G/B must be in the range [0, 1].
func (t *SunLight) Light() (light Light) {
	light.typ = sunLight
	light.layout.SetType(shader.SunLight)
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
