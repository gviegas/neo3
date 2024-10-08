// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/wsi"
)

const rendPrefix = "renderer: "

func newRendErr(reason string) error { return errors.New(rendPrefix + reason) }

// Renderer is a real-time renderer.
type Renderer struct {
	cb [NFrame]driver.CmdBuffer

	lights [NLight]Light
	nlight int

	// TODO: Shadow maps.

	drawables drawableMap

	hdr *Texture
	ds  *Texture

	// TODO: Post-processing data.
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
	// TODO: Initialize Renderer.
	return &Onscreen{sc: sc}, nil
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
	// TODO: Deinitialize Renderer.
	r.sc.Destroy()
	*r = Onscreen{}
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
	// TODO: Initialize Renderer.
	return &Offscreen{rt: rt}, nil
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
	// TODO: Deinitialize Renderer.
	r.rt.Free()
	*r = Offscreen{}
}
