// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package texture provides a wrapper around the
// driver's Image/Sampler types.
package texture

import (
	"errors"
	"sync/atomic"

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
	// The driver.Layout of each layer.
	// A given layouts element will contain an
	// invalid layout value while there is an
	// uncommitted copy or ongoing Transition
	// targeting the layer.
	layouts []atomic.Int64
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

// makeLayouts makes the initial layouts slice that
// Texture expects.
// All layouts are set to driver.LUndefined.
func makeLayouts(param *TexParam) []atomic.Int64 {
	layouts := make([]atomic.Int64, param.Layers)
	if driver.LUndefined != 0 {
		// This path should never be taken.
		for i := range layouts {
			layouts[i].Store(int64(driver.LUndefined))
		}
	}
	return layouts
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
		t = &Texture{views, usage, *param, makeLayouts(param)}
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
		t = &Texture{views, usage, *param, makeLayouts(param)}
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
		t = &Texture{views, usage, *param, makeLayouts(param)}
	}
	return
}

// IsValidView checks whether view identifies a valid
// driver.ImageView of t.
//
// For non-arrayed (single-layer) textures, or cube
// textures with exactly six layers, only view 0
// is valid. This view represents the one layer in
// a 2D/target texture, and each of the six faces in
// a cube texture.
//
// For arrayed (two layers or more) textures, or cube
// textures with more than six layers, there is one
// view per layer (or per six layers for cubes), each
// representing the given layer/cube faces, and one
// extra view encompassing the whole array.
//
// Non-arrayed textures:
//
//	2D/Target | one layer  | one view [0]
//	Cube      | six layers | one view [0]
//
// Arrayed textures:
//
//	2D/Target | N layers   | N+1 views [0, N]
//	Cube      | N>6 layers | N/6+1 views [0, N/6]
//
// In the case of non-cube textures, the arrayed view
// is identified by t.Layers(). For cube textures,
// it is identified by t.Layers() / 6.
func (t *Texture) IsValidView(view int) bool { return view >= 0 && view < len(t.views) }

// ViewLayers returns the number of layers in the
// given view.
func (t *Texture) ViewLayers(view int) int {
	if !t.IsValidView(view) {
		panic("not a valid view of Texture")
	}
	if t.param.Layers > 1 {
		if view == t.param.Layers {
			// Entire array.
			return t.param.Layers
		}
		if len(t.views) < t.param.Layers {
			// Cube faces.
			return 6
		}
	}
	return 1
}

// ViewSize returns the size in bytes of the given
// view's memory.
// It does not consider the memory consumed by
// additional mip levels.
func (t *Texture) ViewSize(view int) int {
	nl := t.ViewLayers(view)
	n := t.param.Size() * t.param.Width * t.param.Height
	return nl * n
}

// CopyToView copies CPU data to the given view of t.
// Only the first mip level must be provided.
// If t is arrayed and view is the last view, then
// data must contain the first level of every layer,
// in order and tightly packed.
// Unless commit is true, the copy may be delayed.
func (t *Texture) CopyToView(view int, data []byte, commit bool) error {
	if x := t.ViewSize(view); x < len(data) {
		data = data[:x]
	}
	s := <-staging
	off, err := s.stage(data)
	if err == nil {
		err = s.copyToView(t, view, off)
		if commit && err == nil {
			err = s.commit()
		}
	}
	staging <- s
	return err
}

// CopyFromView copies t's view to a given CPU buffer.
// It returns the number of bytes written to dst.
// This method does not grow the dst buffer, so data
// may be lost.
// It implicitly commits the staging buffer.
func (t *Texture) CopyFromView(view int, dst []byte) (int, error) {
	if x := t.ViewSize(view); x < len(dst) {
		dst = dst[:x]
	}
	s := <-staging
	var n int
	off, err := s.reserve(len(dst))
	if err == nil {
		if err = s.copyFromView(t, view, off); err == nil {
			// TODO: Try to defer this call.
			if err = s.commit(); err == nil {
				n = s.unstage(off, dst)
			}
		}
	}
	staging <- s
	return n, err
}

const invalLayout = -1

// setPending stores invalLayout in t.layouts[layer] and
// returns the replaced layout.
// It panics if the current layout is invalid.
func (t *Texture) setPending(layer int) driver.Layout {
	if layout := t.layouts[layer].Swap(invalLayout); layout != invalLayout {
		return driver.Layout(layout)
	}
	panic("layout already pending")
}

// unsetPending stores layout in t.layouts[layer].
// It panics if the current layout is valid.
func (t *Texture) unsetPending(layer int, layout driver.Layout) {
	if !t.layouts[layer].CompareAndSwap(invalLayout, int64(layout)) {
		panic("layout not pending")
	}
}

// Transition records a layout transition for view in
// the given command buffer.
// The caller must ensure that no copies targeting
// this particular view of t happen until the command
// completes execution.
// The caller is also responsible for calling
// t.SetLayout after the transition executes to
// update t's state. Not doing so may cause a panic.
func (t *Texture) Transition(view int, cb driver.CmdBuffer, layout driver.Layout, barrier driver.Barrier) {
	if !t.IsValidView(view) {
		panic("not a valid view of Texture")
	}
	if !cb.IsRecording() {
		panic("driver.CmdBuffer is not recording")
	}
	if layout == driver.LUndefined {
		panic("layout is driver.LUndefined")
	}

	il := view
	nl := 1
	if t.param.Layers > 1 {
		if view == t.param.Layers {
			// Entire array.
			il = 0
			nl = t.param.Layers
		} else if len(t.views) < t.param.Layers {
			// Cube faces.
			il = view * 6
			nl = 6
		}
	}
	// Need separate transitions if not all
	// layers are in the same layout.
	var differ bool
	before := []driver.Layout{t.setPending(il)}
	for i := 1; i < nl; i++ {
		layout := t.setPending(il + i)
		before = append(before, layout)
		differ = differ || layout != before[0]
	}

	if differ {
		// TODO: Consider caching this on t.
		xs := make([]driver.Transition, nl)
		for i := 0; i < nl; i++ {
			xs = append(xs, driver.Transition{
				Barrier:      barrier,
				LayoutBefore: before[i],
				LayoutAfter:  layout,
				Img:          t.views[view].Image(),
				Layer:        il + i,
				Layers:       1,
				Level:        0,
				Levels:       t.param.Levels,
			})
		}
		cb.Transition(xs)
	} else {
		cb.Transition([]driver.Transition{
			{
				Barrier:      barrier,
				LayoutBefore: before[0],
				LayoutAfter:  layout,
				Img:          t.views[view].Image(),
				Layer:        il,
				Layers:       nl,
				Level:        0,
				Levels:       t.param.Levels,
			},
		})
	}
}

// SetLayout sets the layout of view.
// It must be called, exactly once, after the preceding
// t.Transition command executes to update t's state.
// layout must either match the transition's layout, or
// be driver.LUndefined (in case of failure to execute
// the layout transition command).
// Calling this method with no preceding Transition is
// invalid and may cause a panic.
func (t *Texture) SetLayout(view int, layout driver.Layout) {
	if !t.IsValidView(view) {
		panic("not a valid view of Texture")
	}
	il := view
	nl := 1
	if t.param.Layers > 1 {
		if view == t.param.Layers {
			// Entire array.
			il = 0
			nl = t.param.Layers
		} else if len(t.views) < t.param.Layers {
			// Cube faces.
			il = view * 6
			nl = 6
		}
	}
	for i := 0; i < nl; i++ {
		t.unsetPending(il+i, layout)
	}
}

// PixelFmt returns the driver.PixelFmt of t.
func (t *Texture) PixelFmt() driver.PixelFmt { return t.param.PixelFmt }

// Width returns the width of t's first mip level.
func (t *Texture) Width() int { return t.param.Width }

// Height returns the height of t's first mip level.
func (t *Texture) Height() int { return t.param.Height }

// Layers returns the number of layers in t.
func (t *Texture) Layers() int { return t.param.Layers }

// Levels returns the number of levels in t.
func (t *Texture) Levels() int { return t.param.Levels }

// Samples returns the number of samples in t.
func (t *Texture) Samples() int { return t.param.Samples }

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

// Min returns the driver.Filter of s that is used
// for minification.
func (s *Sampler) Min() driver.Filter { return s.param.Min }

// Mag returns the driver.Filter of s that is used
// for magnification.
func (s *Sampler) Mag() driver.Filter { return s.param.Mag }

// Mipmap returns the driver.Filter of s that is used
// for mip level selection.
func (s *Sampler) Mipmap() driver.Filter { return s.param.Mipmap }

// AddrU returns the driver.AddrMode of s that is used
// for u coordinate addressing.
func (s *Sampler) AddrU() driver.AddrMode { return s.param.AddrU }

// AddrV returns the driver.AddrMode of s that is used
// for v coordinate addressing.
func (s *Sampler) AddrV() driver.AddrMode { return s.param.AddrV }

// AddrW returns the driver.AddrMode of s that is used
// for w coordinate addressing.
func (s *Sampler) AddrW() driver.AddrMode { return s.param.AddrW }

// MaxAniso returns the maximum anisotropy of s.
func (s *Sampler) MaxAniso() int { return s.param.MaxAniso }

// Cmp returns the driver.CmpFunc of s.
func (s *Sampler) Cmp() driver.CmpFunc { return s.param.Cmp }

// MinLOD returns the minimum level of detail of s.
func (s *Sampler) MinLOD() float32 { return s.param.MinLOD }

// MaxLOD returns the maximum level of detail of s.
func (s *Sampler) MaxLOD() float32 { return s.param.MaxLOD }

// Free invalidates s and destroys the driver.Sampler.
func (s *Sampler) Free() {
	if s.sampler != nil {
		s.sampler.Destroy()
	}
	*s = Sampler{}
}
