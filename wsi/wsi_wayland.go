// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

package wsi

// #cgo linux LDFLAGS: -ldl
// #include <dlfcn.h>
// #include <stdlib.h>
// #include <wsi_wayland.h>
import "C"

import (
	"errors"
	"unsafe"
)

// Handle for the shared object.
var hWayland unsafe.Pointer

// openWayland opens the shared library and gets function pointers.
// It is not safe to call any of the C wrappers unless this
// function succeeds.
func openWayland() error {
	if hWayland == nil {
		lib := C.CString("libwayland-client.so.0")
		defer C.free(unsafe.Pointer(lib))
		hWayland := C.dlopen(lib, C.RTLD_LAZY|C.RTLD_GLOBAL)
		if hWayland == nil {
			return errors.New("wsi: failed to open libwayland")
		}
		for i := range C.nameWayland {
			C.ptrWayland[i] = C.dlsym(hWayland, C.nameWayland[i])
			if C.ptrWayland[i] == nil {
				C.dlclose(hWayland)
				hWayland = nil
				return errors.New("wsi: failed to fetch Wayland symbol")
			}
		}
	}
	return nil
}

// closeWayland closes the shared library.
// It is not safe to call any of the C wrappers after
// calling this function.
func closeWayland() {
	if hWayland != nil {
		C.dlclose(hWayland)
		hWayland = nil
	}
}

// initWayland initializes the Wayland platform.
func initWayland() error {
	// TODO
	panic("not implemented")
}

// deinitWayland deinitializes the Wayland platform.
func deinitWayland() {
	// TODO
	panic("not implemented")
}

// windowWayland implements Window.
type windowWayland struct {
	// TODO
	width  int
	height int
	title  string
	mapped bool
}

// newWindowWayland creates a new window.
func newWindowWayland(width, height int, title string) (Window, error) {
	// TODO
	panic("not implemented")
}

// Map makes the window visible.
func (w *windowWayland) Map() error {
	// TODO
	panic("not implemented")
}

// Unmap hides the window.
func (w *windowWayland) Unmap() error {
	// TODO
	panic("not implemented")
}

// Resize resizes the window.
func (w *windowWayland) Resize(width, height int) error {
	// TODO
	panic("not implemented")
}

// SetTitle sets the window's title.
func (w *windowWayland) SetTitle(title string) error {
	// TODO
	panic("not implemented")
}

// Close closes the window.
func (w *windowWayland) Close() {
	// TODO
	panic("not implemented")
}

// Width returns the window's width.
func (w *windowWayland) Width() int { return w.width }

// Height returns the window's height.
func (w *windowWayland) Height() int { return w.height }

// Title returns the window's title.
func (w *windowWayland) Title() string { return w.title }

// dispatchWayland dispatches queued events.
func dispatchWayland() {
	// TODO
	panic("not implemented")
}

// setAppNameWayland updates the string used to identify the
// application.
func setAppNameWayland(s string) {
	// TODO
	panic("not implemented")
}
