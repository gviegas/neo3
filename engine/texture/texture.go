// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package texture provides a wrapper around the
// driver's Image/Sampler types.
package texture

import (
	"errors"

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
	// TODO: This call returns the Limits struct by value.
	limits := ctx.GPU().Limits()
	var reason string
	switch {
	case param == nil:
		reason = "nil param"
	case param.Dim3D.Width < 1, param.Dim3D.Height < 1, param.Dim3D.Depth != 0:
		reason = "invalid size"
	case param.Dim3D.Width > limits.MaxImage2D, param.Dim3D.Height > limits.MaxImage2D:
		reason = "size too big"
	case param.Layers < 1:
		reason = "invalid layer count"
	case param.Layers > limits.MaxLayers:
		reason = "too many layers"
	case param.Levels < 1, param.Levels > ComputeLevels(param.Dim3D):
		reason = "invalid level count"
	case param.Samples < 1, param.Samples&(param.Samples-1) != 0:
		reason = "invalid sample count"
	default:
		goto validParam
	}
	err = errors.New(reason)
	return
validParam:
	usg := driver.UShaderSample
	img, err := ctx.GPU().NewImage(param.PixelFmt, param.Dim3D, param.Layers, param.Levels, param.Samples, usg)
	if err == nil {
		// TODO: Must call Image.Destroy when unreachable.
		t = &Texture{img, usg, *param}
	}
	return
}

// ComputeLevels returns the maximum number of mip levels
// for a given driver.Dim3D.
// It assumes that size is valid (i.e., neither negative
// nor the zero value).
func ComputeLevels(size driver.Dim3D) int {
	x := size.Width
	if x < size.Height {
		x = size.Height
	}
	if x < size.Depth {
		x = size.Depth
	}
	var l int
	for ; x > 0; l++ {
		x /= 2
	}
	return l
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
	var reason string
	switch {
	case param == nil:
		reason = "nil param"
	// TODO: An upper bound should be defined
	// in driver.Limits.
	case param.MaxAniso < 1:
		reason = "invalid max anisotropy"
	case param.MinLOD < 0:
		reason = "invalid min LOD"
	case param.MaxLOD < 0:
		reason = "invalid max LOD"
	case param.MinLOD > param.MaxLOD:
		reason = "min LOD greater than max LOD"
	default:
		goto validParam
	}
	err = errors.New(reason)
	return
validParam:
	splr, err := ctx.GPU().NewSampler(param)
	if err == nil {
		// TODO: Must call Sampler.Destroy when unreachable.
		s = &Sampler{splr, *param}
	}
	return
}
