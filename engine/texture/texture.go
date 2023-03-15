// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package texture provides a wrapper around the
// driver's Image/Sampler types.
package texture

import (
	"errors"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
)

const prefix = "texture: "

// Texture wraps a driver.Image.
type Texture struct {
	// One view per layer (or every 6th, in case
	// of cube textures). If the image is arrayed,
	// then there will be an additional view of
	// the whole array at the end.
	views []driver.ImageView
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

const (
	tex2D = iota
	texCube
	texTarget
)

// makeViews creates a driver.Image from param/usage and
// makes the driver.ImageView slice that Texture expects.
// It assumes that the parameters are valid.
func makeViews(param *TexParam, usage driver.Usage, texType int) (v []driver.ImageView, err error) {
	img, err := ctxt.GPU().NewImage(param.PixelFmt, param.Dim3D, param.Layers, param.Levels, param.Samples, usage)
	if err != nil {
		return
	}

	var typ driver.ViewType
	// Non-arrayed cube views take
	// six layers.
	var nl int

	switch texType {
	case tex2D, texTarget:
		if param.Layers > 1 {
			var ltyp driver.ViewType
			if param.Samples > 1 {
				ltyp = driver.IView2DMSArray
				typ = driver.IView2DMS
			} else {
				ltyp = driver.IView2DArray
				typ = driver.IView2D
			}
			view, err := img.NewView(ltyp, 0, param.Layers, 0, param.Levels)
			if err != nil {
				img.Destroy()
				return nil, err
			}
			v = make([]driver.ImageView, param.Layers+1)
			v[param.Layers] = view
		} else {
			if param.Samples > 1 {
				typ = driver.IView2DMS
			} else {
				typ = driver.IView2D
			}
			v = []driver.ImageView{nil}
		}
		nl = 1
	case texCube:
		if param.Layers > 6 {
			// BUG: Certain back-ends may not support
			// views of type IViewCubeArray.
			view, err := img.NewView(driver.IViewCubeArray, 0, param.Layers, 0, param.Levels)
			if err != nil {
				img.Destroy()
				return nil, err
			}
			v = make([]driver.ImageView, param.Layers/6+1)
			v[param.Layers/6] = view
		} else {
			typ = driver.IViewCube
			v = []driver.ImageView{nil}
		}
		nl = 6
	default:
		panic("undefined texture type")
	}

	// Create non-arrayed views.
	for i := 0; i < param.Layers/nl; i++ {
		v[i], err = img.NewView(typ, i*nl, nl, 0, param.Levels)
		if err != nil {
			for j := 0; j < i; j++ {
				v[j].Destroy()
			}
			if param.Layers > nl {
				v[param.Layers/nl].Destroy()
			}
			img.Destroy()
			v = nil
			break
		}
	}
	return
}

// New2D creates a 2D texture.
func New2D(param *TexParam) (t *Texture, err error) {
	limits := ctxt.Limits()
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
	case param.Levels > 1 && param.Samples != 1:
		reason = "multi-sample mipmap"
	default:
		goto validParam
	}
	err = errors.New(prefix + reason)
	return
validParam:
	usage := driver.UShaderSample
	views, err := makeViews(param, usage, tex2D)
	if err == nil {
		// TODO: Should destroy driver resources
		// when unreachable (unless Texture.Free
		// is called first).
		t = &Texture{views, usage, *param}
	}
	return
}

// NewCube creates a new cube texture.
func NewCube(param *TexParam) (t *Texture, err error) {
	limits := ctxt.Limits()
	var reason string
	switch {
	case param == nil:
		reason = "nil param"
	case param.Dim3D.Width < 1, param.Dim3D.Height < 1, param.Dim3D.Depth != 0:
		reason = "invalid size"
	case param.Dim3D.Width != param.Dim3D.Height:
		reason = "cube's width and height differs"
	case param.Dim3D.Width > limits.MaxImageCube:
		reason = "size too big"
	case param.Layers < 1:
		reason = "invalid layer count"
	case param.Layers > limits.MaxLayers:
		reason = "too many layers"
	case param.Layers%6 != 0:
		reason = "cube's layer count not a multiple of 6"
	case param.Levels < 1, param.Levels > ComputeLevels(param.Dim3D):
		reason = "invalid level count"
	case param.Samples != 1:
		reason = "multi-sample cube"
	default:
		goto validParam
	}
	err = errors.New(prefix + reason)
	return
validParam:
	usage := driver.UShaderSample
	views, err := makeViews(param, usage, texCube)
	if err == nil {
		// TODO: Should destroy driver resources
		// when unreachable (unless Texture.Free
		// is called first).
		t = &Texture{views, usage, *param}
	}
	return
}

// NewTarget creates a new render target texture.
func NewTarget(param *TexParam) (t *Texture, err error) {
	limits := ctxt.Limits()
	var reason string
	switch {
	case param == nil:
		reason = "nil param"
	case param.Dim3D.Width < 1, param.Dim3D.Height < 1, param.Dim3D.Depth != 0:
		reason = "invalid size"
	case param.Width > limits.MaxRenderSize[0], param.Height > limits.MaxRenderSize[1]:
		reason = "size too big"
	case param.Layers < 1:
		reason = "invalid layer count"
	case param.Layers > limits.MaxRenderLayers:
		reason = "too many layers"
	case param.Levels < 1, param.Levels > ComputeLevels(param.Dim3D):
		reason = "invalid level count"
	case param.Samples < 1, param.Samples&(param.Samples-1) != 0:
		reason = "invalid sample count"
	case param.Levels > 1 && param.Samples != 1:
		reason = "multi-sample mipmap"
	default:
		goto validParam
	}
	err = errors.New(prefix + reason)
	return
validParam:
	usage := driver.UShaderSample | driver.URenderTarget
	views, err := makeViews(param, usage, texTarget)
	if err == nil {
		// TODO: Should destroy driver resources
		// when unreachable (unless Texture.Free
		// is called first).
		t = &Texture{views, usage, *param}
	}
	return
}

// CopyToView copies CPU data to the given view of t.
// Only the first mip level must be provided.
// If t is arrayed and view is the last view, then
// data must contain the first level of every layer,
// in order and tightly packed.
func (t *Texture) CopyToView(view int, data []byte) error {
	s := <-staging
	off, err := s.stage(data)
	if err != nil {
		// TODO: Track layouts.
		err = s.copyToView(t, view, driver.LUndefined, driver.LShaderRead, off)
	}
	staging <- s
	return err
}

// CopyFromView copies t's view to a given CPU buffer.
// It returns the number of bytes written to dst.
// This method does not grow the dst buffer, so data
// may be lost.
func (t *Texture) CopyFromView(view int, dst []byte) (int, error) {
	s := <-staging
	var n int
	off, err := s.reserve(len(dst))
	if err != nil {
		// TODO: Track layouts.
		if err = s.copyFromView(t, view, driver.LColorTarget, driver.LColorTarget, off); err != nil {
			// TODO: Try to defer this call.
			if err = s.commit(); err != nil {
				n = s.unstage(off, dst)
			}
		}
	}
	staging <- s
	return n, err
}

// Free invalidates t and destroys the driver.Image and
// the driver.ImageView(s).
func (t *Texture) Free() {
	if len(t.views) > 0 {
		img := t.views[0].Image()
		for _, v := range t.views {
			v.Destroy()
		}
		img.Destroy()
	}
	*t = Texture{}
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
	err = errors.New(prefix + reason)
	return
validParam:
	splr, err := ctxt.GPU().NewSampler(param)
	if err == nil {
		// TODO: Should destroy driver resource
		// when unreachable (unless Sampler.Free
		// is called first).
		s = &Sampler{splr, *param}
	}
	return
}

// Free invalidates s and destroys the driver.Sampler.
func (s *Sampler) Free() {
	if s.sampler != nil {
		s.sampler.Destroy()
	}
	*s = Sampler{}
}
