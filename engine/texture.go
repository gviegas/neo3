// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/internal/bitvec"
)

const texPrefix = "texture: "

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
	img, err := ctxt.GPU().NewImage(
		param.PixelFmt, param.Dim3D, param.Layers, param.Levels, param.Samples, usage)
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
	err = errors.New(texPrefix + reason)
	return
validParam:
	// TODO: Consider removing driver.UCopySrc and
	// disallowing CopyFromView calls instead.
	usage := driver.UCopySrc | driver.UCopyDst | driver.UShaderSample
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
	features := ctxt.Features()
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
	case param.Layers > 6 && !features.CubeArray:
		reason = "cube arrays not supported"
	case param.Levels < 1, param.Levels > ComputeLevels(param.Dim3D):
		reason = "invalid level count"
	case param.Samples != 1:
		reason = "multi-sample cube"
	default:
		goto validParam
	}
	err = errors.New(texPrefix + reason)
	return
validParam:
	// TODO: Consider removing driver.UCopySrc and
	// disallowing CopyFromView calls instead.
	usage := driver.UCopySrc | driver.UCopyDst | driver.UShaderSample
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
	err = errors.New(texPrefix + reason)
	return
validParam:
	// TODO: Consider removing driver.UCopyDst and
	// disallowing CopyToView calls instead.
	usage := driver.UCopySrc | driver.UCopyDst | driver.UShaderSample | driver.URenderTarget
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
//
// TODO: Provide a method that actually considers
// the whole mip chain.
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
//
// TODO: Allow copying data to any mip level.
func (t *Texture) CopyToView(view int, data []byte, commit bool) error {
	if x := t.ViewSize(view); x < len(data) {
		data = data[:x]
	}
	s := <-texStg
	off, err := s.stage(data)
	if err == nil {
		err = s.copyToView(t, view, off)
		if commit && err == nil {
			err = s.commit()
		}
	}
	texStg <- s
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
	s := <-texStg
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
	texStg <- s
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

// transition records a layout transition for view in
// the given command buffer.
// The caller must ensure that no copies targeting
// this particular view of t happen until the command
// completes execution.
// The caller is also responsible for calling
// t.setLayout after the transition executes to
// update t's state.
func (t *Texture) transition(view int, cb driver.CmdBuffer, layout driver.Layout, barrier driver.Barrier) {
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
	// Need to split the transition if the
	// layouts of any two layers differ.
	// TODO: Maybe try to merge contiguous
	// layers that share the same layout.
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
		cb.Transition([]driver.Transition{{
			Barrier:      barrier,
			LayoutBefore: before[0],
			LayoutAfter:  layout,
			Img:          t.views[view].Image(),
			Layer:        il,
			Layers:       nl,
			Level:        0,
			Levels:       t.param.Levels,
		}})
	}
}

// setLayout sets the layout of view.
// It must be called, exactly once, after the preceding
// t.transition command executes to update t's state.
// layout must either match the transition's layout, or
// be driver.LUndefined (in case of failure to execute
// the layout transition command).
// Calling this method with no preceding transition is
// not allowed.
func (t *Texture) setLayout(view int, layout driver.Layout) {
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
// The caller is responsible for ensuring that there
// are no pending copies targeting any view of t, and
// that none is issued during the call.
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
	err = errors.New(texPrefix + reason)
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

// Cmp returns the driver.CmpFunc of s and a boolean
// indicating whether it supports depth comparison.
func (s *Sampler) Cmp() (driver.CmpFunc, bool) { return s.param.Cmp, s.param.DoCmp }

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

// TODO:
// - Separate read/write staging buffers;
// - Give more control to when commit happens;
// - Support buffer staging.

var (
	// Global texture staging buffer(s).
	texStg chan *texStgBuffer
	// Variables for commitTexStg calls.
	texStgMu    sync.Mutex
	texStgCache []*texStgBuffer
	texStgWk    chan *driver.WorkItem
)

func init() {
	n := runtime.GOMAXPROCS(-1)
	texStg = make(chan *texStgBuffer, n)
	for i := 0; i < n; i++ {
		s, err := newTexStg(texStgBlock * texStgNBit)
		if err != nil {
			s = &texStgBuffer{}
		}
		texStg <- s
	}
	texStgCache = make([]*texStgBuffer, 0, n)
	texStgWk = make(chan *driver.WorkItem, 1)
	texStgWk <- &driver.WorkItem{Work: make([]driver.CmdBuffer, 0, n)}
}

// commitTexStg executes all pending Texture copies.
// It blocks until execution completes.
func commitTexStg() (err error) {
	texStgMu.Lock()
	swk := <-texStgWk

	// This deferral correctly clears global
	// state, regardless of the outcome.
	// Code below ensures that the command
	// buffers are reset if necessary.
	defer func() {
		for _, x := range texStgCache {
			x.bv.Clear()
			x.drainPending(err != nil)
			texStg <- x
		}
		texStgCache = texStgCache[:0]
		swk.Work = swk.Work[:0]
		texStgWk <- swk
		texStgMu.Unlock()
	}()

	n := cap(texStg)
	for i := 0; i < n; i++ {
		texStgCache = append(texStgCache, <-texStg)
	}

	for i, x := range texStgCache {
		wk := <-x.wk
		if !wk.Work[0].IsRecording() {
			if len(x.pend) != 0 {
				// This should never happen.
				panic("commitTexStg: pending copies while not recording")
			}
		} else if err = wk.Work[0].End(); err != nil {
			x.wk <- wk
			for _, x := range swk.Work {
				// Need to reset these since
				// they won't be committed.
				x.Reset()
			}
			for _, x := range texStgCache[i+1:] {
				// Need to reset these since
				// they won't be ended.
				wk := <-x.wk
				wk.Work[0].Reset()
				x.wk <- wk
			}
			return
		} else {
			swk.Work = append(swk.Work, wk.Work[0])
		}
		x.wk <- wk
	}

	if len(swk.Work) == 0 {
		return
	}
	if err = ctxt.GPU().Commit(swk, texStgWk); err != nil {
		return
	}
	swk = <-texStgWk
	err, swk.Err = swk.Err, nil
	return
}

// texStgBuffer is used to copy image data
// between the CPU and the GPU.
type texStgBuffer struct {
	wk   chan *driver.WorkItem
	buf  driver.Buffer
	bv   bitvec.V[uint32]
	pend []pendingCopy
}

// pendingCopy is used to track Texture/view
// pairs that have a pending copy operation.
type pendingCopy struct {
	tex  *Texture
	view int
	// The layout that will be set
	// after the copy executes.
	layout driver.Layout
}

// Use a large block size since textures usually
// need large allocations.
// 1024x1024 32-bit textures (no mip) will take
// one bit vector word with this configuration.
const (
	texStgBlock = 131072
	texStgNBit  = 32
)

// newTexStg creates a new texStgBuffer with the
// given size in bytes.
// n must be greater than 0; it will be rounded up
// to a multiple of texStgBlock * texStgNBit.
func newTexStg(n int) (*texStgBuffer, error) {
	if n <= 0 {
		panic("newTexStg: n <= 0")
	}
	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		return nil, err
	}
	wk := make(chan *driver.WorkItem, 1)
	wk <- &driver.WorkItem{Work: []driver.CmdBuffer{cb}}
	n = (n + texStgBlock*texStgNBit - 1) &^ (texStgBlock*texStgNBit - 1)
	buf, err := ctxt.GPU().NewBuffer(int64(n), true, driver.UCopySrc|driver.UCopyDst)
	if err != nil {
		cb.Destroy()
		return nil, err
	}
	var bv bitvec.V[uint32]
	bv.Grow(n / texStgBlock / texStgNBit)
	return &texStgBuffer{wk, buf, bv, nil}, nil
}

// copyToView records a copy command that copies
// data from s's buffer into view.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// texStgBlock).
// Only the first mip level must be provided.
// If t is arrayed and view is the last view, then
// the buffer must contain the first level of
// every layer, in order and tightly packed.
func (s *texStgBuffer) copyToView(t *Texture, view int, off int64) (err error) {
	if t.param.Samples != 1 {
		return errors.New(texPrefix + "cannot copy data to MS texture")
	}
	if view < 0 || view >= len(t.views) {
		return errors.New(texPrefix + "view index out of bounds")
	}

	il := view
	nl := 1
	if t.param.Layers > 1 {
		switch n := len(t.views); {
		case view == n-1:
			il = 0
			nl = t.param.Layers
		case n < t.param.Layers:
			// Cube texture.
			il = view * 6
			nl = 6
		}
	}
	n := t.param.PixelFmt.Size() * t.param.Dim3D.Width * t.param.Dim3D.Height
	if off+int64(n*nl) > s.buf.Cap() {
		return errors.New(texPrefix + "not enough buffer capacity for copying")
	}

	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if err = wk.Work[0].Begin(); err != nil {
			s.bv.Clear()
			s.wk <- wk
			return
		}
	}

	wk.Work[0].Transition([]driver.Transition{{
		Barrier: driver.Barrier{
			SyncBefore:   driver.SNone,
			SyncAfter:    driver.SCopy,
			AccessBefore: driver.ANone,
			AccessAfter:  driver.ACopyWrite,
		},
		LayoutBefore: driver.LUndefined,
		LayoutAfter:  driver.LCopyDst,
		Img:          t.views[view].Image(),
		Layer:        il,
		Layers:       nl,
		Level:        0,
		Levels:       1, // TODO
	}})

	wk.Work[0].CopyBufToImg(&driver.BufImgCopy{
		Buf:    s.buf,
		BufOff: off,
		// TODO: RowStrd must be 256-byte aligned.
		RowStrd: t.param.Dim3D.Width,
		SlcStrd: t.param.Dim3D.Height,
		Img:     t.views[view].Image(),
		ImgOff:  driver.Off3D{},
		Layer:   il,
		Level:   0,
		Size:    t.param.Dim3D,
		Layers:  nl,
		// TODO: Handle depth/stencil formats.
	})
	for i := 0; i < nl; i++ {
		// The current layout is not relevant
		// because the whole layer is going to
		// be overwritten by this command.
		// TODO: Change this when adding support
		// for sub-view copying.
		_ = t.setPending(il + i)
		s.pend = append(s.pend, pendingCopy{t, il + i, driver.LCopyDst})
	}
	if t.param.Levels > 1 {
		// TODO
		panic("texStgBuffer.copyToView: no mip gen yet")
	}

	s.wk <- wk
	return
}

// copyFromView records a copy command that copies
// data from view into s's buffer.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// texStgBlock).
func (s *texStgBuffer) copyFromView(t *Texture, view int, off int64) (err error) {
	if t.param.Samples != 1 {
		return errors.New(texPrefix + "cannot copy data from MS texture")
	}
	if view < 0 || view >= len(t.views) {
		return errors.New(texPrefix + "view index out of bounds")
	}

	il := view
	nl := 1
	if t.param.Layers > 1 {
		switch n := len(t.views); {
		case view == n-1:
			il = 0
			nl = t.param.Layers
		case n < t.param.Layers:
			// Cube texture.
			il = view * 6
			nl = 6
		}
	}
	// TODO: Consider the required space for
	// all mip levels.
	n := t.param.PixelFmt.Size() * t.param.Dim3D.Width * t.param.Dim3D.Height
	if off+int64(n*nl) > s.buf.Cap() {
		return errors.New(texPrefix + "not enough buffer capacity for copying")
	}
	// Need to split the transition if the
	// layouts of any two layers differ.
	// TODO: Maybe try to merge contiguous
	// layers that share the same layout.
	var differ bool
	before := []driver.Layout{t.setPending(il)}
	for i := 1; i < nl; i++ {
		layout := t.setPending(il + i)
		before = append(before, layout)
		differ = differ || layout != before[0]
	}

	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if err = wk.Work[0].Begin(); err != nil {
			s.bv.Clear()
			s.wk <- wk
			return
		}
	}

	if differ {
		// TODO: Consider caching this on s
		// (or t; see Texture.Transition).
		xs := make([]driver.Transition, nl)
		img := t.views[view].Image()
		for i := 0; i < nl; i++ {
			xs = append(xs, driver.Transition{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SNone,
					SyncAfter:    driver.SCopy,
					AccessBefore: driver.ANone,
					AccessAfter:  driver.ACopyRead,
				},
				LayoutBefore: before[i],
				LayoutAfter:  driver.LCopySrc,
				Img:          img,
				Layer:        il + i,
				Layers:       1,
				Level:        0,
				Levels:       1, // TODO
			})
		}
		wk.Work[0].Transition(xs)
	} else {
		wk.Work[0].Transition([]driver.Transition{{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SNone,
				SyncAfter:    driver.SCopy,
				AccessBefore: driver.ANone,
				AccessAfter:  driver.ACopyRead,
			},
			LayoutBefore: before[0],
			LayoutAfter:  driver.LCopySrc,
			Img:          t.views[view].Image(),
			Layer:        il,
			Layers:       nl,
			Level:        0,
			Levels:       1, // TODO
		}})
	}

	wk.Work[0].CopyImgToBuf(&driver.BufImgCopy{
		Buf:    s.buf,
		BufOff: off,
		// TODO: RowStrd must be 256-byte aligned.
		RowStrd: t.param.Dim3D.Width,
		SlcStrd: t.param.Dim3D.Height,
		Img:     t.views[view].Image(),
		ImgOff:  driver.Off3D{},
		Layer:   il,
		Level:   0,
		Size:    t.param.Dim3D,
		Layers:  nl,
		// TODO: Handle depth/stencil formats.
	})
	for i := 0; i < nl; i++ {
		s.pend = append(s.pend, pendingCopy{t, il + i, driver.LCopySrc})
	}
	if t.param.Levels > 1 {
		// TODO
		panic("texStgBuffer.copyFromView: no mip copy yet")
	}

	s.wk <- wk
	return
}

// stage writes CPU data to s's buffer.
// It may need to commit pending copy commands to
// grow the buffer.
// It returns an offset from the start of s.buf
// identifying where data was copied to.
func (s *texStgBuffer) stage(data []byte) (off int64, err error) {
	if off, err = s.reserve(len(data)); err == nil {
		copy(s.buf.Bytes()[off:], data)
	}
	return
}

// unstage writes s.buf's data to dst.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// texStgBlock).
// It returns the number of bytes written.
//
// NOTE: Since texStgBuffer methods may flush
// the command buffer and/or clear the bit vector,
// unstage usually should be called right after a
// copy-back command is committed and before
// staging new copy commands.
func (s *texStgBuffer) unstage(off int64, dst []byte) (n int) {
	if off >= s.buf.Cap() {
		return
	}
	if off%texStgBlock != 0 {
		panic("texStgBuffer.unstage: misaligned off")
	}
	n = copy(dst, s.buf.Bytes()[off:])
	ib := int(off) / texStgBlock
	nb := (n + texStgBlock - 1) / texStgBlock
	for i := 0; i < nb; i++ {
		s.bv.Unset(ib + i)
	}
	return
}

// reserve reserves a contiguous range of n bytes
// within s.buf.
// It may need to commit pending copy commands to
// grow the buffer.
// It returns an offset from the start of s.buf
// identifying where the range starts.
func (s *texStgBuffer) reserve(n int) (off int64, err error) {
	if n <= 0 {
		panic("texStgBuffer.reserve: n <= 0")
	}
	n = (n + texStgBlock - 1) / texStgBlock
	idx, ok := s.bv.SearchRange(n)
	if !ok {
		if err = s.commit(); err != nil {
			return
		}
		// TODO: Consider using idx 0 instead.
		idx = s.bv.Len()
		n := (n + texStgNBit - 1) / texStgNBit
		s.bv.Grow(n)
		// TODO: Make buffer cap bounds configurable.
		n = n * texStgBlock * texStgNBit
		if s.buf != nil {
			n += int(s.buf.Cap())
			s.buf.Destroy()
		}
		if s.buf, err = ctxt.GPU().NewBuffer(int64(n), true, 0); err != nil {
			// TODO: Try again ignoring previous
			// s.buf.Cap() value (if not 0).
			s.bv = bitvec.V[uint32]{}
			return
		}
	}
	for i := 0; i < n; i++ {
		s.bv.Set(idx + i)
	}
	off = int64(idx) * texStgBlock
	return
}

// commit commits the copy commands for execution.
// It blocks until execution completes.
func (s *texStgBuffer) commit() (err error) {
	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if len(s.pend) != 0 {
			// This should never happen.
			panic("texStgBuffer.commit: pending copies while not recording")
		}
		s.wk <- wk
		return
	}
	// TODO: May have to clear the
	// bit vector unconditionally.
	s.bv.Clear()
	if err = wk.Work[0].End(); err != nil {
		s.drainPending(true)
		s.wk <- wk
		return
	}
	if err = ctxt.GPU().Commit(wk, s.wk); err != nil {
		s.drainPending(true)
		s.wk <- wk
		return
	}
	wk = <-s.wk
	err, wk.Err = wk.Err, nil
	s.drainPending(err != nil)
	s.wk <- wk
	return
}

// drainPending removes every element from s.pend
// and updates the textures accordingly.
// If failed is true, then the layouts are set to
// driver.LUndefined instead.
func (s *texStgBuffer) drainPending(failed bool) {
	if failed {
		for _, x := range s.pend {
			x.tex.unsetPending(x.view, driver.LUndefined)
		}
	} else {
		for _, x := range s.pend {
			x.tex.unsetPending(x.view, x.layout)
		}
	}
	s.pend = s.pend[:0]
}

// free invalidates s and destroys the driver
// resources.
func (s *texStgBuffer) free() {
	if s.wk != nil {
		wk := <-s.wk
		wk.Work[0].Destroy()
	}
	if s.buf != nil {
		s.buf.Destroy()
	}
	s.drainPending(true)
	*s = texStgBuffer{}
}
