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
	dpyWayland  *C.struct_wl_display
	rtyWayland  *C.struct_wl_registry
	cptWayland  *C.struct_wl_compositor
	wmXDG       *C.struct_xdg_wm_base
	seatWayland *C.struct_wl_seat
)

// initWayland initializes the Wayland platform.
func initWayland() (err error) {
	if dpyWayland != nil {
		return
	}
	if hWayland = C.openWayland(); hWayland == nil {
		return errors.New("wsi: openWayland failed")
	}
	defer func() {
		if err != nil {
			deinitWayland()
		}
	}()
	if dpyWayland = C.displayConnectWayland(nil); dpyWayland == nil {
		err = errors.New("wsi: displayConnectWayland failed")
		return
	}
	if rtyWayland = C.displayGetRegistryWayland(dpyWayland); rtyWayland == nil {
		err = errors.New("wsi: displayGetRegistryWayland failed")
		return
	}
	if C.registryAddListenerWayland(rtyWayland) != 0 {
		err = errors.New("wsi: registryAddListenerWayland failed")
		return
	}
	C.displayRoundtripWayland(dpyWayland)
	if cptWayland == nil {
		err = errors.New("wsi: cptWayland is nil")
		return
	}
	if wmXDG == nil {
		err = errors.New("wsi: wmXDG is nil")
		return
	}
	if seatWayland == nil {
		err = errors.New("wsi: seatWayland is nil")
		return
	}
	return
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
		if cptWayland != nil {
			// TODO
		}
		if wmXDG != nil {
			// TODO
		}
		if seatWayland != nil {
			// TODO
		}
		if rtyWayland != nil {
			// TODO
		}
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
	case "wl_seat":
		i := &C.seatInterfaceWayland
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		seatWayland = (*C.struct_wl_seat)(p)
	}
}

//export registryGlobalRemoveWayland
func registryGlobalRemoveWayland(name C.uint32_t) {
	// TODO
	println("\tregistryGlobalRemoveWayland:", name)
}

//export surfaceEnterWayland
func surfaceEnterWayland(sf *C.struct_wl_surface, out *C.struct_wl_output) {
	// TODO
	println("\tsurfaceEnterWayland:", sf, out)
}

//export surfaceLeaveWayland
func surfaceLeaveWayland(sf *C.struct_wl_surface, out *C.struct_wl_output) {
	// TODO
	println("\tsurfaceLeaveWayland:", sf, out)
}

//export wmBasePingXDG
func wmBasePingXDG(serial C.uint32_t) {
	// TODO
	println("\twmBasePingXDG:", serial)
}

//export seatCapabilitiesWayland
func seatCapabilitiesWayland(capab C.uint32_t) {
	// TODO
	println("\tseatCapabilitiesWayland:", capab)
}

//export seatNameWayland
func seatNameWayland(name *C.char) {
	// TODO
	println("\tseatNameWayland:", C.GoString(name))
}
