// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver

import (
	"errors"

	"gviegas/neo3/wsi"
)

// ErrCannotPresent means that the driver and/or device do not
// support presentation.
var ErrCannotPresent = errors.New("driver: presentation not supported")

// ErrWindow represents an error related to a specific window.
// This error usually indicates that a window misconfiguration
// is preventing correct operation. For instance, the driver
// may require a visible window to create a swapchain.
var ErrWindow = errors.New("driver: window-related error")

// ErrSwapchain represents an error related to a specific
// swapchain.
// This error usually indicates that changes to the window or
// compositor made the swapchain unusable.
var ErrSwapchain = errors.New("driver: swapchain-related error")

// ErrNoBackbuffer means that all available backbuffers
// were acquired.
// Backbuffers are released during presentation.
var ErrNoBackbuffer = errors.New("driver: all backbuffers in use")

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
// To present, one calls Next to obtain the index of an
// image view to target, transitions the view to a valid
// layout (e.g., from LUndefined to LColorTarget),
// records commands as needed, transitions the view to
// the LPresent layout, commits these commands and then
// calls Present to present the image view.
type Swapchain interface {
	Destroyer

	// Views returns the list of image views that
	// comprises the swapchain.
	// This value remains unchanged as long as the
	// swapchain's Destroy or Recreate methods are
	// not called.
	// Swapchain image views are in the LUndefined
	// layout when created/recreated.
	Views() []ImageView

	// Next returns the index of the next writable
	// image view.
	// The image view must be transitioned to a
	// valid layout before it can be used.
	// If there was a prior presentation whose
	// execution succeeded, the view can be
	// transitioned from LPresent instead of
	// LUndefined.
	Next() (int, error)

	// Present presents the image view identified
	// by index.
	// Before calling this method, the given image
	// view must be transitioned to the LPresent
	// layout and the command buffer used to
	// record this transition must be committed.
	Present(index int) error

	// Recreate recreates the swapchain.
	// It is meant to be called in response to a
	// ErrSwapchain error.
	Recreate() error

	// Format returns the image views' PixelFmt.
	Format() PixelFmt

	// Usage returns the image views' Usage.
	// URenderTarget is guaranteed to be set.
	Usage() Usage
}
