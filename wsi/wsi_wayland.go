// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

package wsi

// #include <stdlib.h>
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
	ptWayland   *C.struct_wl_pointer
	kbWayland   *C.struct_wl_keyboard

	// Name of globals in the server.
	nameCptWayland  C.uint32_t
	nameWMXDG       C.uint32_t
	nameSeatWayland C.uint32_t
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
	if C.wmBaseAddListenerXDG(wmXDG) != 0 {
		err = errors.New("wsi: wmBaseAddListenerXDG failed")
		return
	}
	if C.seatAddListenerWayland(seatWayland) != 0 {
		err = errors.New("wsi: seatAddListenerWayland failed")
		return
	}
	C.displayRoundtripWayland(dpyWayland)

	if ptWayland != nil && C.pointerAddListenerWayland(ptWayland) != 0 {
		err = errors.New("wsi: pointerAddListenerWayland failed")
		return
	}
	if kbWayland != nil && C.keyboardAddListenerWayland(kbWayland) != 0 {
		err = errors.New("wsi: keyboardAddListenerWayland failed")
		return
	}
	C.displayRoundtripWayland(dpyWayland)

	newWindow = newWindowWayland
	dispatch = dispatchWayland
	setAppName = setAppNameWayland
	platform = Wayland
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
			C.compositorDestroyWayland(cptWayland)
			cptWayland = nil
		}
		if wmXDG != nil {
			C.wmBaseDestroyXDG(wmXDG)
			wmXDG = nil
		}
		if ptWayland != nil {
			C.pointerDestroyWayland(ptWayland)
			ptWayland = nil
		}
		if kbWayland != nil {
			C.keyboardDestroyWayland(kbWayland)
			kbWayland = nil
		}
		if seatWayland != nil {
			C.seatDestroyWayland(seatWayland)
			seatWayland = nil
		}
		if rtyWayland != nil {
			C.registryDestroyWayland(rtyWayland)
			rtyWayland = nil
		}
		C.displayDisconnectWayland(dpyWayland)
		dpyWayland = nil
	}
	C.closeWayland(hWayland)
	initDummy()
}

// windowWayland implements Window.
type windowWayland struct {
	wsf      *C.struct_wl_surface
	xsf      *C.struct_xdg_surface
	toplevel *C.struct_xdg_toplevel
	width    int
	height   int
	title    string
	ctitle   []C.char
	mapped   bool
}

// newWindowWayland creates a new window.
func newWindowWayland(width, height int, title string) (Window, error) {
	if wmXDG == nil {
		return nil, errors.New("wsi: xdg_wm_base not present")
	}

	wsf := C.compositorCreateSurfaceWayland(cptWayland)
	if wsf == nil {
		return nil, errors.New("wsi: compositorCreateSurfaceWayland failed")
	}
	if C.surfaceAddListenerWayland(wsf) != 0 {
		C.surfaceDestroyWayland(wsf)
		return nil, errors.New("wsi: surfaceAddListenerWayland failed")
	}

	xsf := C.wmBaseGetXDGSurfaceXDG(wmXDG, wsf)
	if xsf == nil {
		C.surfaceDestroyWayland(wsf)
		return nil, errors.New("wsi: wmBaseGetXDGSurfaceXDG failed")
	}
	if C.surfaceAddListenerXDG(xsf) != 0 {
		C.surfaceDestroyXDG(xsf)
		C.surfaceDestroyWayland(wsf)
		return nil, errors.New("wsi: surfaceAddListenerXDG failed")
	}

	toplevel := C.surfaceGetToplevelXDG(xsf)
	if toplevel == nil {
		C.surfaceDestroyXDG(xsf)
		C.surfaceDestroyWayland(wsf)
		return nil, errors.New("wsi: surfaceGetToplevelXDG failed")
	}
	if C.toplevelAddListenerXDG(toplevel) != 0 {
		C.toplevelDestroyXDG(toplevel)
		C.surfaceDestroyXDG(xsf)
		C.surfaceDestroyWayland(wsf)
		return nil, errors.New("wsi: toplevelAddListenerXDG failed")
	}
	ctitle := unsafe.Slice(C.CString(title), len(title)+1)
	C.toplevelSetTitleXDG(toplevel, &ctitle[0])

	C.surfaceCommitWayland(wsf)
	C.displayRoundtripWayland(dpyWayland)

	return &windowWayland{
		wsf:      wsf,
		xsf:      xsf,
		toplevel: toplevel,
		width:    width,
		height:   height,
		title:    title,
		ctitle:   ctitle,
		mapped:   false,
	}, nil
}

// Map makes the window visible.
func (w *windowWayland) Map() error {
	w.mapped = true
	return nil
}

// Unmap hides the window.
func (w *windowWayland) Unmap() error {
	if w.mapped {
		w.mapped = false
		C.surfaceAttachWayland(w.wsf, nil, 0, 0)
		C.surfaceCommitWayland(w.wsf)
		C.displayFlushWayland(dpyWayland)
	}
	return nil
}

// Resize resizes the window.
func (w *windowWayland) Resize(width, height int) error {
	// TODO
	println("windowWayland.Resize: not implemented")
	return nil
}

// SetTitle sets the window's title.
func (w *windowWayland) SetTitle(title string) error {
	if title == w.title {
		return nil
	}
	if n := len(title); n >= len(w.ctitle) {
		C.free(unsafe.Pointer(&w.ctitle[0]))
		w.ctitle = unsafe.Slice(C.CString(title), n+1)
	} else {
		sl := unsafe.Slice((*byte)(unsafe.Pointer(&w.ctitle[0])), n+1)
		copy(sl, title)
		sl[n] = 0
	}
	w.title = title
	return nil
}

// Close closes the window.
func (w *windowWayland) Close() {
	if w != nil {
		closeWindow(w)
		if dpyWayland != nil {
			C.toplevelDestroyXDG(w.toplevel)
			C.surfaceDestroyXDG(w.xsf)
			C.surfaceDestroyWayland(w.wsf)
			C.displayFlushWayland(dpyWayland)
		}
		C.free(unsafe.Pointer(&w.ctitle[0]))
		*w = windowWayland{}
	}
}

// Width returns the window's width.
func (w *windowWayland) Width() int { return w.width }

// Height returns the window's height.
func (w *windowWayland) Height() int { return w.height }

// Title returns the window's title.
func (w *windowWayland) Title() string { return w.title }

// dispatchWayland dispatches queued events.
func dispatchWayland() {
	C.displayFlushWayland(dpyWayland)
	C.displayDispatchPendingWayland(dpyWayland)
}

// setAppNameWayland updates the string used to identify the
// application.
func setAppNameWayland(s string) {
	if windowCount > 0 {
		appID := C.CString(s)
		for _, w := range createdWindows {
			if w != nil {
				C.toplevelSetAppIDXDG(w.(*windowWayland).toplevel, appID)
			}
		}
		C.displayFlushWayland(dpyWayland)
		C.free(unsafe.Pointer(appID))
	}
}

// windowFromWayland returns the window in createdWindows
// whose wsf field matches sf, or nil if none does.
func windowFromWayland(sf *C.struct_wl_surface) Window {
	for _, w := range createdWindows {
		if w != nil && w.(*windowWayland).wsf == sf {
			return w
		}
	}
	return nil
}

// windowFromXDG returns the window in createdWindows
// whose xsf field matches sf, or nil if none does.
func windowFromXDG(sf *C.struct_xdg_surface) Window {
	for _, w := range createdWindows {
		if w != nil && w.(*windowWayland).xsf == sf {
			return w
		}
	}
	return nil
}

// windowFromToplevel returns the window in createdWindows
// whose toplevel field matches tl, or nil if none does.
func windowFromToplevel(tl *C.struct_xdg_toplevel) Window {
	for _, w := range createdWindows {
		if w != nil && w.(*windowWayland).toplevel == tl {
			return w
		}
	}
	return nil
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
		nameCptWayland = name
	case "xdg_wm_base":
		i := &C.wmBaseInterfaceXDG
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		wmXDG = (*C.struct_xdg_wm_base)(p)
		nameWMXDG = name
	case "wl_seat":
		i := &C.seatInterfaceWayland
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		seatWayland = (*C.struct_wl_seat)(p)
		nameSeatWayland = name
	}
}

//export registryGlobalRemoveWayland
func registryGlobalRemoveWayland(name C.uint32_t) {
	println("\tregistryGlobalRemoveWayland:", name) // XXX

	closeWin := func() {
		if windowCount > 0 {
			for _, w := range createdWindows {
				if w != nil {
					w.Close()
				}
			}
		}
	}

	switch {
	case name == nameCptWayland && cptWayland != nil:
		closeWin()
		C.compositorDestroyWayland(cptWayland)
		cptWayland = nil
		nameCptWayland = 0
	case name == nameWMXDG && wmXDG != nil:
		closeWin()
		C.wmBaseDestroyXDG(wmXDG)
		wmXDG = nil
		nameWMXDG = 0
	case name == nameSeatWayland && seatWayland != nil:
		if ptWayland != nil {
			C.pointerDestroyWayland(ptWayland)
			ptWayland = nil
		}
		if kbWayland != nil {
			C.keyboardDestroyWayland(kbWayland)
			kbWayland = nil
		}
		C.seatDestroyWayland(seatWayland)
		seatWayland = nil
		nameSeatWayland = 0
	}
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
	println("\twmBasePingXDG:", serial) // XXX

	C.wmBasePongXDG(wmXDG, serial)
}

//export surfaceConfigureXDG
func surfaceConfigureXDG(xsf *C.struct_xdg_surface, serial C.uint32_t) {
	println("\tsurfaceConfigureXDG:", xsf, serial) // XXX

	// TODO: Avoid this whenever possible.
	C.surfaceAckConfigureXDG(xsf, serial)
}

//export toplevelConfigureXDG
func toplevelConfigureXDG(tl *C.struct_xdg_toplevel, width, height C.int32_t, states *C.struct_wl_array) {
	println("\ttoplevelConfigureXDG:", tl, width, height, states) // XXX

	var win *windowWayland
	if x := windowFromToplevel(tl); x == nil {
		return
	} else {
		win = x.(*windowWayland)
	}
	if int(width) == win.width && int(height) == win.height {
		return
	}
	win.width = int(width)
	win.height = int(height)
	if windowHandler != nil {
		windowHandler.WindowResize(win, win.width, win.height)
	}
}

//export toplevelCloseXDG
func toplevelCloseXDG(tl *C.struct_xdg_toplevel) {
	println("\ttoplevelCloseXDG:", tl) // XXX

	if windowHandler != nil {
		if win := windowFromToplevel(tl); win != nil {
			windowHandler.WindowClose(win)
		}
	}
}

//export toplevelConfigureBoundsXDG
func toplevelConfigureBoundsXDG(tl *C.struct_xdg_toplevel, width, height C.int32_t) {
	// TODO
	println("\ttoplevelConfigureBoundsXDG:", tl, width, height)
}

//export seatCapabilitiesWayland
func seatCapabilitiesWayland(capab C.uint32_t) {
	println("\tseatCapabilitiesWayland:", capab) // XXX

	if capab&C.WL_SEAT_CAPABILITY_POINTER != 0 {
		ptWayland = C.seatGetPointerWayland(seatWayland)
	}
	if capab&C.WL_SEAT_CAPABILITY_KEYBOARD != 0 {
		kbWayland = C.seatGetKeyboardWayland(seatWayland)
	}
}

//export seatNameWayland
func seatNameWayland(name *C.char) {
	// TODO
	println("\tseatNameWayland:", C.GoString(name))
}

//export pointerEnterWayland
func pointerEnterWayland(serial C.uint32_t, sf *C.struct_wl_surface, x, y C.wl_fixed_t) {
	// TODO
	println("\tpointerEnterWayland:", serial, sf, x, y)
}

//export pointerLeaveWayland
func pointerLeaveWayland(serial C.uint32_t, sf *C.struct_wl_surface) {
	// TODO
	println("\tpointerleaveWayland:", serial, sf)
}

//export pointerMotionWayland
func pointerMotionWayland(millis C.uint32_t, x, y C.wl_fixed_t) {
	// TODO
	println("\tpointerMotionWayland:", millis, x, y)
}

//export pointerButtonWayland
func pointerButtonWayland(serial, millis, button, state C.uint32_t) {
	// TODO
	println("\tpointerButtonWayland:", serial, millis, button, state)
}

//export pointerAxisWayland
func pointerAxisWayland(millis, axis C.uint32_t, value C.wl_fixed_t) {
	// TODO
	println("\tpointerAxisWayland:", millis, axis, value)
}

//export pointerFrameWayland
func pointerFrameWayland() {
	// TODO
	println("\tpointerFrameWyland: n/a")
}

//export pointerAxisSourceWayland
func pointerAxisSourceWayland(axisSrc C.uint32_t) {
	// TODO
	println("\tpointerAxisSourceWayland:", axisSrc)
}

//export pointerAxisStopWayland
func pointerAxisStopWayland(millis, axis C.uint32_t) {
	// TODO
	println("\tpointerAxisStopWayland:", millis, axis)
}

//export pointerAxisDiscreteWayland
func pointerAxisDiscreteWayland(axis C.uint32_t, discrete C.int32_t) {
	// TODO
	println("\tpointerAxisDiscreteWayland:", axis, discrete)
}

//export keyboardKeymapWayland
func keyboardKeymapWayland(format C.uint32_t, fd C.int32_t, size C.uint32_t) {
	// TODO
	println("\tkeyboardKeymapWayland:", format, fd, size)
}

//export keyboardEnterWayland
func keyboardEnterWayland(serial C.uint32_t, sf *C.struct_wl_surface, keys *C.struct_wl_array) {
	// TODO
	println("\tkeyboardEnterWayland:", serial, sf, keys)
}

//export keyboardLeaveWayland
func keyboardLeaveWayland(serial C.uint32_t, sf *C.struct_wl_surface) {
	// TODO
	println("\tkeyboardLeaveWayland:", serial, sf)
}

//export keyboardKeyWayland
func keyboardKeyWayland(serial, millis, key, state C.uint32_t) {
	// TODO
	println("\tkeyboardKeyWayland:", serial, millis, key, state)
}

//export keyboardModifiersWayland
func keyboardModifiersWayland(serial, depressed, latched, locked, group C.uint32_t) {
	// TODO
	println("\tkeyboardModifiersWayland:", serial, depressed, latched, locked, group)
}

//export keyboardRepeatInfoWayland
func keyboardRepeatInfoWayland(rate, delay C.int32_t) {
	// TODO
	println("\tkeyboardRepeatInfoWayland:", rate, delay)
}
