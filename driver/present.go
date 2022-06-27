// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver

import (
	"errors"

	"github.com/gviegas/scene/wsi"
)

// ErrCannotPresent means that the driver and/or device do not
// support presentation.
var ErrCannotPresent = errors.New("presentation not supported")

// ErrWindow represents an error related to a specific window.
// This error usually indicates that a window misconfiguration
// is preventing correct operation. For instance, the driver
// may require a visible window to create a swapchain.
var ErrWindow = errors.New("window-related error")

// ErrCompositor represents an error related to the compositor.
// This error usually indicates that the compositor behavior
// is preventing correct operation. For instance, the driver
// may require support for opaque composition.
var ErrCompositor = errors.New("compositor-related error")

// ErrSwapchain represents an error related to a specific
// swapchain.
// This error usually indicates that changes to the window or
// compositor made the swapchain unusable.
var ErrSwapchain = errors.New("swapchain-related error")

// ErrNoBackbuffer means that all available backbuffers
// were acquired.
// Backbuffers are released during presentation.
var ErrNoBackbuffer = errors.New("all backbuffers in use")

// Presenter is the interface that a GPU may implement
// to enable presentation on a display.
type Presenter interface {
	// NewSwapchain creates a new swapchain.
	// Only one swapchain can be associated with a specific
	// wsi.Window at a time.
	NewSwapchain(win wsi.Window, imageCount int) (Swapchain, error)
}

// Swapchain is the interface that defines a n-buffered
// swapchain for presentation.
// Presentation works similar as commands, such that it
// only takes effect after calling GPU.Commit.
// To present, one calls the Next and Present methods of
// the swapchain and then commits the command buffer(s)
// that it targets for execution.
// As a limitation, only one Next/Present pair can be
// recorded in a single Commit.
type Swapchain interface {
	Destroyer

	// Views returns the list of image views that
	// comprises the swapchain.
	// This value remains unchanged as long as the
	// swapchain's Destroy or Recreate methods are
	// not called.
	Views() []ImageView

	// Next returns the index of the next writable
	// image view.
	// cb must be the first command buffer that will
	// access the image's contents.
	// This method must be called before the image
	// is written, i.e., any render pass that uses
	// the image as render target must be recorded
	// after Next.
	Next(cb CmdBuffer) (int, error)

	// Present presents the image view identified
	// by index.
	// cb must be the last command buffer that will
	// write to the image.
	// This method must be called after the image is
	// written, i.e., any render pass that uses the
	// image as render target must be recorded
	// before Present.
	Present(index int, cb CmdBuffer) error

	// Recreate recreates the swapchain.
	// It is meant to be called in response to a
	// ErrSwapchain error.
	Recreate() error

	// Format returns the image views' PixelFmt.
	Format() PixelFmt
}
