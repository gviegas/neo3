// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package texture provides a wrapper around the
// driver's Image/Sampler types.
package texture

import (
	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctx"
)

// Texture wraps a driver.Image.
type Texture struct {
	image driver.Image
	usage driver.Usage
	param TexParam
}

// TexParam describes parameters of a texture.
type TexParam struct {
	driver.PixelFmt
	driver.Dim3D
	Layers  int
	Levels  int
	Samples int
}

// New2D creates a 2D texture.
func New2D(param *TexParam) (t *Texture, err error) {
	// TODO: Check parameters.
	usg := driver.UShaderSample
	img, err := ctx.GPU().NewImage(param.PixelFmt, param.Dim3D, param.Layers, param.Levels, param.Samples, usg)
	if err == nil {
		// TODO: Must call Image.Destroy when unreachable.
		t = &Texture{img, usg, *param}
	}
	return
}

// Sampler wraps a driver.Sampler.
type Sampler struct {
	sampler driver.Sampler
	param   SplrParam
}

// SplrParam describes parameters of a sampler.
type SplrParam = driver.Sampling

// NewSampler creates a new sampler.
func NewSampler(param *SplrParam) (s *Sampler, err error) {
	// TODO: Check parameters.
	splr, err := ctx.GPU().NewSampler(param)
	if err == nil {
		// TODO: Must call Sampler.Destroy when unreachable.
		s = &Sampler{splr, *param}
	}
	return
}
