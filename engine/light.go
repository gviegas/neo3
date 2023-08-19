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

// SetDirection sets the direction of the light.
// It does not normalize d.
// Only applies to sun and spot lights.
func (l *Light) SetDirection(d *linear.V3) { l.layout.SetDirection(d) }

// SetPosition sets the position of the light.
// Only applies to point and spot lights.
func (l *Light) SetPosition(p *linear.V3) { l.layout.SetPosition(p) }

// TODO: Maybe add more setters to Light.

// SunLight is a directional light.
// The light is emitted in the given Direction.
// It behaves as if located infinitely far way.
// Intensity's unit is lux.
type SunLight struct {
	Direction linear.V3
	Intensity float32
	R, G, B   float32
}

// Light creates the light source described by t.
// t.Direction must have length 1.
// t.R/G/B must be in the range [0, 1].
func (t *SunLight) Light() Light {
	var l shader.LightLayout
	l.SetType(shader.SunLight)
	l.SetIntensity(t.Intensity)
	l.SetColor(&linear.V3{t.R, t.G, t.B})
	l.SetDirection(&t.Direction)
	return Light{
		typ:    sunLight,
		layout: l,
	}
}

// PointLight is an omnidirectional, positional light.
// The light is emitted in all directions from the
// given Position.
// Range determines the area affected by the light.
// Intensity's unit is candela.
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
func (t *PointLight) Light() Light {
	var l shader.LightLayout
	l.SetType(shader.PointLight)
	l.SetIntensity(t.Intensity)
	l.SetRange(t.Range)
	l.SetColor(&linear.V3{t.R, t.G, t.B})
	l.SetPosition(&t.Position)
	return Light{
		typ:    pointLight,
		layout: l,
	}
}

// SpotLight is a directional, positional light.
// The light is emitted in a cone in the given Direction
// from the given Position.
// Range, InnerAngle and OuterAngle determine the area
// affected by the light.
// InnerAngle's/OuterAngle's unit is radians.
// Intensity's unit is candela.
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
func (t *SpotLight) Light() Light {
	var (
		inner    = max(0, min(float64(t.InnerAngle), math.Pi/2-1e-6))
		outer    = max(inner+1e-6, min(float64(t.OuterAngle), math.Pi/2))
		innerCos = math.Cos(inner)
		outerCos = math.Cos(outer)
		scale    = 1 / (innerCos - outerCos)
	)
	var l shader.LightLayout
	l.SetType(shader.SpotLight)
	l.SetIntensity(t.Intensity)
	l.SetRange(t.Range)
	l.SetColor(&linear.V3{t.R, t.G, t.B})
	l.SetAngScale(float32(scale))
	l.SetPosition(&t.Position)
	l.SetAngOffset(float32(scale * -outerCos))
	l.SetDirection(&t.Direction)
	return Light{
		typ:    spotLight,
		layout: l,
	}
}
