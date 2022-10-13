// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

package wsi

// #include <wsi_wayland.h>
import "C"

import (
	"errors"
	"unsafe"
)

// Handle for the shared object.
var hWayland unsafe.Pointer

// Common Wayland variables.
var (
	dpyWayland *C.struct_wl_display
	rtyWayland *C.struct_wl_registry
	cptWayland *C.struct_wl_compositor
	wmXDG      *C.struct_xdg_wm_base
)

// initWayland initializes the Wayland platform.
func initWayland() error {
	if dpyWayland != nil {
		return nil
	}

	// TODO: Close the lib and clear globals on error.

	if hWayland = C.openWayland(); hWayland == nil {
		return errors.New("wsi: openWayland failed")
	}
	if dpyWayland = C.displayConnectWayland(nil); dpyWayland == nil {
		return errors.New("wsi: displayConnectWayland failed")
	}
	if rtyWayland = C.displayGetRegistryWayland(dpyWayland); rtyWayland == nil {
		return errors.New("wsi: displayGetRegistryWayland failed")
	}
	if C.registryAddListenerWayland(rtyWayland) != 0 {
		return errors.New("wsi: registryAddListenerWayland failed")
	}

	C.displayRoundtripWayland(dpyWayland)

	// TODO
	if cptWayland == nil {
		panic("cptWayland is nil")
	} else {
		println("cptWayland:", cptWayland)
	}
	if wmXDG == nil {
		panic("wmXDG is nil")
	} else {
		println("wmXDG:", wmXDG)
	}

	return nil
}

// deinitWayland deinitializes the Wayland platform.
func deinitWayland() {
	if windowCount > 0 {
		for _, w := range createdWindows {
			if w != nil {
				w.Close()
			}
		}
	}
	if dpyWayland != nil {
		C.displayDisconnectWayland(dpyWayland)
		dpyWayland = nil
	}
	C.closeWayland(hWayland)
	initDummy()
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

//export registryGlobalWayland
func registryGlobalWayland(name C.uint32_t, iface *C.char, vers C.uint32_t) {
	s := C.GoString(iface)

	println("\tregistryGlobalWayland:", name, s, vers) // XXX

	switch s {
	case "wl_compositor":
		i := &C.compositorInterfaceWayland
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		cptWayland = (*C.struct_wl_compositor)(p)
	case "xdg_wm_base":
		i := &C.wmBaseInterfaceXDG
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		wmXDG = (*C.struct_xdg_wm_base)(p)
	}
}

//export registryGlobalRemoveWayland
func registryGlobalRemoveWayland(name C.uint32_t) {
	// TODO
	println("\tregistryGlobalRemoveWayland:", name)
}

//export surfaceEnterWayland
func surfaceEnterWayland(sfc *C.struct_wl_surface, out *C.struct_wl_output) {
	// TODO
	println("\tsurfaceEnterWayland:", sfc, out)
}

//export surfaceLeaveWayland
func surfaceLeaveWayland(sfc *C.struct_wl_surface, out *C.struct_wl_output) {
	// TODO
	println("\tsurfaceLeaveWayland:", sfc, out)
}

//export wmBasePingXDG
func wmBasePingXDG(serial C.uint32_t) {
	// TODO
	println("\twmBasePingXDG:", serial)
}
