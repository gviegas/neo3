// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

// Renderer is a real-time renderer.
type Renderer struct {
	cb [MaxFrame]driver.CmdBuffer

	lights [MaxLight]Light
	nlight int

	// TODO: Shadow maps.

	hdr *Texture
	ds  *Texture

	// TODO: Post-processing data.
}

// Onscreen is a Renderer that targets a wsi.Window.
type Onscreen struct {
	Renderer
	win wsi.Window
}

// Offscreen is a Renderer that targets a Texture.
type Offscreen struct {
	Renderer
	rt *Texture
}
