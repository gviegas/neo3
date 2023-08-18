// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"gviegas/neo3/engine/internal/shader"
	"gviegas/neo3/linear"
)

// SunLight is a directional light.
type SunLight struct {
	Direction linear.V3
	Intensity float32
	R, G, B   float32
}

// PointLight is an omnidirectional, positional light.
type PointLight struct {
	Position  linear.V3
	Range     float32
	Intensity float32
	R, G, B   float32
}

// SpotLight is a directional, positional light defined
// by a conical shape.
type SpotLight struct {
	Direction  linear.V3
	Position   linear.V3
	InnerAngle float32
	OuterAngle float32
	Range      float32
	Intensity  float32
	R, G, B    float32
}

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
