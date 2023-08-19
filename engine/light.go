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
type Light struct {
	typ    int
	layout shader.LightLayout
}

// SunLight is a directional light.
// The light is emitted in the given Direction.
// It behaves as if located infinitely far way.
type SunLight struct {
	Direction linear.V3
	Intensity float32
	R, G, B   float32
}

// Light creates the light source described by t.
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
type PointLight struct {
	Position  linear.V3
	Range     float32
	Intensity float32
	R, G, B   float32
}

// Light creates the light source described by t.
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
