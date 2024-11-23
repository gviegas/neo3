// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"
	"iter"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/wsi"
)

const rendPrefix = "renderer: "

func newRendErr(reason string) error { return errors.New(rendPrefix + reason) }

// Renderer is a real-time renderer.
// Onscreen and Offscreen embed a Renderer
// (call either NewOnscreen or NewOffscreen to
// create a valid  Renderer).
type Renderer struct {
	cb [NFrame]driver.CmdBuffer
	ch chan *driver.WorkItem

	lights [NLight]Light
	nlight int

	// TODO: Shadow maps.

	drawables drawableMap

	hdr *Texture
	ds  *Texture

	// TODO: Post-processing data.
}

// init initializes r.
// It assumes that r has not been initialized yet
// (call r.free first if that is not the case).
func (r *Renderer) init(width, height int) (err error) {
	defer func() {
		if err != nil {
			r.free()
		}
	}()
	for i := range r.cb {
		r.cb[i], err = ctxt.GPU().NewCmdBuffer()
		if err != nil {
			return
		}
	}
	r.ch = make(chan *driver.WorkItem, NFrame)
	for i := range cap(r.ch) {
		r.ch <- &driver.WorkItem{
			Work:   []driver.CmdBuffer{r.cb[i]},
			Custom: i,
		}
	}
	for i := range r.lights {
		r.lights[i].layout.SetUnused(true)
	}
	// TODO: Initialize r.drawables.
	// TODO: Customizable sample count.
	// TODO: Choose a better DS format if available.
	r.hdr, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA16Float,
		Dim3D: driver.Dim3D{
			Width:  width,
			Height: height,
		},
		Layers:  1,
		Levels:  1,
		Samples: 4,
	})
	if err != nil {
		return
	}
	r.ds, err = NewTarget(&TexParam{
		PixelFmt: driver.D16Unorm,
		Dim3D: driver.Dim3D{
			Width:  width,
			Height: height,
		},
		Layers:  1,
		Levels:  1,
		Samples: 4,
	})
	//if err != nil {
	//	return
	//}
	return
}

// SetLight updates the light at the given index
// to contain a copy of *light.
// If light is nil, the slot is set as unused.
func (r *Renderer) SetLight(index int, light *Light) {
	unused := r.lights[index].layout.Unused()
	if light != nil {
		r.lights[index] = *light
		r.lights[index].layout.SetUnused(false)
		if unused {
			r.nlight++
		}
	} else {
		r.lights[index].layout.SetUnused(true)
		if !unused {
			r.nlight--
		}
	}
}

// Light returns a pointer to the light that was
// last set at the given index.
// If the slot is unused, it returns nil instead.
// It is not allowed to assign a new value to the
// return pointer; use SetLight instead.
func (r *Renderer) Light(index int) *Light {
	unused := r.lights[index].layout.Unused()
	if !unused {
		return &r.lights[index]
	}
	return nil
}

// Lights returns an iterator over the light slots
// that are currently in use, in the usual order.
func (r *Renderer) Lights() iter.Seq2[int, *Light] {
	return func(yield func(int, *Light) bool) {
		n := r.nlight
		for i := 0; n > 0; i++ {
			if r.lights[i].layout.Unused() {
				continue
			}
			n--
			if !yield(i, &r.lights[i]) {
				return
			}
		}
	}
}

// free invalidates r and destroys/releases the
// driver resources it holds.
func (r *Renderer) free() {
	if r == nil {
		return
	}
	for range cap(r.ch) {
		<-r.ch
	}
	for _, cb := range r.cb {
		cb.Destroy()
	}
	// TODO: Deinitialize r.drawables.
	r.hdr.Free()
	r.ds.Free()
	*r = Renderer{}
}

// Onscreen is a Renderer that targets a wsi.Window.
type Onscreen struct {
	Renderer
	win wsi.Window
	sc  driver.Swapchain
}

// NewOnscreen creates a new onscreen renderer.
func NewOnscreen(win wsi.Window) (*Onscreen, error) {
	if win == nil {
		return nil, newRendErr("nil wsi.Window in call to NewOnscreen")
	}
	pres, ok := ctxt.GPU().(driver.Presenter)
	if !ok {
		return nil, newRendErr("NewOnscreen requires driver.Presenter")
	}
	sc, err := pres.NewSwapchain(win, NFrame+1)
	if err != nil {
		return nil, err
	}
	var r Onscreen
	err = r.init(win.Width(), win.Height())
	if err != nil {
		sc.Destroy()
		return nil, err
	}
	r.win = win
	r.sc = sc
	return &r, nil
}

// Window returns the wsi.Window associated with r.
func (r *Onscreen) Window() wsi.Window { return r.win }

// Free invalidates r and destroys/releases the
// driver resources it holds.
// It does not call Close on the wsi.Window.
func (r *Onscreen) Free() {
	if r == nil {
		return
	}
	r.free()
	r.sc.Destroy()
	r.win = nil
	r.sc = nil
}

// Offscreen is a Renderer that targets a Texture.
type Offscreen struct {
	Renderer
	rt *Texture
}

// NewOffscreen creates a new offscreen renderer.
func NewOffscreen(width, height int) (*Offscreen, error) {
	rt, err := NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: width, Height: height},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		return nil, err
	}
	var r Offscreen
	err = r.init(width, height)
	if err != nil {
		rt.Free()
		return nil, err
	}
	r.rt = rt
	return &r, nil
}

// Target returns the Texture into which r renders.
func (r *Offscreen) Target() *Texture { return r.rt }

// Free invalidates r and destroys/releases the
// driver resources it holds.
// It does call Free on its target Texture.
func (r *Offscreen) Free() {
	if r == nil {
		return
	}
	r.free()
	r.rt.Free()
	r.rt = nil
}
