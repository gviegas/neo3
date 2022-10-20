// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build linux && !android

package wsi

// #include <stdlib.h>
// #include <wsi_wayland.h>
import "C"

import (
	"errors"
	"fmt"
	"os"
	"unsafe"
)

// Handle for the shared object.
var hWayland unsafe.Pointer

// Common Wayland variables.
var (
	dpyWayland  *C.struct_wl_display
	rtyWayland  *C.struct_wl_registry
	cptWayland  *C.struct_wl_compositor
	shmWayland  *C.struct_wl_shm
	wmXDG       *C.struct_xdg_wm_base
	seatWayland *C.struct_wl_seat
	ptWayland   *C.struct_wl_pointer
	kbWayland   *C.struct_wl_keyboard

	// Name of globals in the server.
	nameCptWayland  C.uint32_t
	nameShmWayland  C.uint32_t
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
	if shmWayland == nil {
		err = errors.New("wsi: shmWayland is nil")
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
	if C.shmAddListenerWayland(shmWayland) != 0 {
		err = errors.New("wsi: shmAddListenerWayland failed")
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
		if shmWayland != nil {
			C.shmDestroyWayland(shmWayland)
			shmWayland = nil
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

	C.surfaceCommitWayland(wsf)
	C.displayRoundtripWayland(dpyWayland)

	return &windowWayland{
		wsf:      wsf,
		xsf:      xsf,
		toplevel: toplevel,
		width:    width,
		height:   height,
		title:    title,
		ctitle:   unsafe.Slice(C.CString(title), len(title)+1),
		mapped:   false,
	}, nil
}

// Map makes the window visible.
func (w *windowWayland) Map() error {
	if !w.mapped {
		w.mapped = true
		C.toplevelSetTitleXDG(w.toplevel, &w.ctitle[0])
		appID := C.CString(appName)
		C.toplevelSetAppIDXDG(w.toplevel, appID)
		C.surfaceCommitWayland(w.wsf)
		C.displayFlushWayland(dpyWayland)
		C.free(unsafe.Pointer(appID))
	}
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
	return errors.New("wsi: windowWayland.Resize: not implemented")
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
	switch C.GoString(iface) {
	case "wl_compositor":
		i := &C.compositorInterfaceWayland
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		cptWayland = (*C.struct_wl_compositor)(p)
		nameCptWayland = name
	case "wl_shm":
		i := &C.shmInterfaceWayland
		p := C.registryBindWayland(rtyWayland, name, i, vers)
		shmWayland = (*C.struct_wl_shm)(p)
		nameShmWayland = name
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
	case name == nameShmWayland && shmWayland != nil:
		closeWin()
		C.shmDestroyWayland(shmWayland)
		shmWayland = nil
		nameShmWayland = 0
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

//export shmFormatWayland
func shmFormatWayland(format C.uint32_t) {}

//export surfaceEnterWayland
func surfaceEnterWayland(sf *C.struct_wl_surface, out *C.struct_wl_output) {}

//export surfaceLeaveWayland
func surfaceLeaveWayland(sf *C.struct_wl_surface, out *C.struct_wl_output) {}

//export wmBasePingXDG
func wmBasePingXDG(serial C.uint32_t) {
	C.wmBasePongXDG(wmXDG, serial)
}

//export surfaceConfigureXDG
func surfaceConfigureXDG(xsf *C.struct_xdg_surface, serial C.uint32_t) {
	// TODO: Avoid this whenever possible.
	C.surfaceAckConfigureXDG(xsf, serial)
}

//export toplevelConfigureXDG
func toplevelConfigureXDG(tl *C.struct_xdg_toplevel, width, height C.int32_t, states *C.struct_wl_array) {
	// TODO: Check states.
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
	if windowHandler != nil {
		if win := windowFromToplevel(tl); win != nil {
			windowHandler.WindowClose(win)
		}
	}
}

//export toplevelConfigureBoundsXDG
func toplevelConfigureBoundsXDG(tl *C.struct_xdg_toplevel, width, height C.int32_t) {}

//export seatCapabilitiesWayland
func seatCapabilitiesWayland(capab C.uint32_t) {
	if capab&C.WL_SEAT_CAPABILITY_POINTER != 0 {
		ptWayland = C.seatGetPointerWayland(seatWayland)
	}
	if capab&C.WL_SEAT_CAPABILITY_KEYBOARD != 0 {
		kbWayland = C.seatGetKeyboardWayland(seatWayland)
	}
}

//export seatNameWayland
func seatNameWayland(name *C.char) {}

//export pointerEnterWayland
func pointerEnterWayland(serial C.uint32_t, sf *C.struct_wl_surface, x, y C.wl_fixed_t) {
	// TODO: Set cursor.
	C.pointerSetCursorWayland(ptWayland, serial, nil, 0, 0)
	if pointerHandler != nil {
		if win := windowFromWayland(sf); win != nil {
			pointerHandler.PointerEnter(win, int(x/256), int(y/256))
		}
	}
}

//export pointerLeaveWayland
func pointerLeaveWayland(serial C.uint32_t, sf *C.struct_wl_surface) {
	if pointerHandler != nil {
		if win := windowFromWayland(sf); win != nil {
			pointerHandler.PointerLeave(win)
		}
	}
}

//export pointerMotionWayland
func pointerMotionWayland(millis C.uint32_t, x, y C.wl_fixed_t) {
	if pointerHandler != nil {
		pointerHandler.PointerMotion(int(x/256), int(y/256))
	}
}

//export pointerButtonWayland
func pointerButtonWayland(serial, millis, button, state C.uint32_t) {
	if pointerHandler != nil {
		btn := BtnUnknown
		switch button {
		case 0x110:
			btn = BtnLeft
		case 0x111:
			btn = BtnRight
		case 0x112:
			btn = BtnMiddle
		case 0x113:
			btn = BtnSide
		case 0x115:
			btn = BtnForward
		case 0x116:
			btn = BtnBackward
		}
		pressed := state == C.WL_POINTER_BUTTON_STATE_PRESSED
		pointerHandler.PointerButton(btn, pressed)
	}
}

//export pointerAxisWayland
func pointerAxisWayland(millis, axis C.uint32_t, value C.wl_fixed_t) {
	// TODO
}

//export pointerFrameWayland
func pointerFrameWayland() {
	// TODO
}

//export pointerAxisSourceWayland
func pointerAxisSourceWayland(axisSrc C.uint32_t) {
	// TODO
}

//export pointerAxisStopWayland
func pointerAxisStopWayland(millis, axis C.uint32_t) {
	// TODO
}

//export pointerAxisDiscreteWayland
func pointerAxisDiscreteWayland(axis C.uint32_t, discrete C.int32_t) {
	// TODO
}

//export keyboardKeymapWayland
func keyboardKeymapWayland(format C.uint32_t, fd C.int32_t, size C.uint32_t) {
	if format != C.WL_KEYBOARD_KEYMAP_FORMAT_XKB_V1 {
		fmt.Fprintf(os.Stderr, "wsi: unknown Wayland keymap format (%d) - cannot use seat's keyboard\n", format)
		if kbWayland != nil {
			C.keyboardDestroyWayland(kbWayland)
			kbWayland = nil
		}
	}
}

//export keyboardEnterWayland
func keyboardEnterWayland(serial C.uint32_t, sf *C.struct_wl_surface, keys *C.struct_wl_array) {
	// TODO: Check keys.
	if keyboardHandler != nil {
		if win := windowFromWayland(sf); win != nil {
			keyboardHandler.KeyboardEnter(win)
		}
	}
}

//export keyboardLeaveWayland
func keyboardLeaveWayland(serial C.uint32_t, sf *C.struct_wl_surface) {
	if keyboardHandler != nil {
		if win := windowFromWayland(sf); win != nil {
			keyboardHandler.KeyboardLeave(win)
		}
	}
}

//export keyboardKeyWayland
func keyboardKeyWayland(serial, millis, key, state C.uint32_t) {
	if keyboardHandler != nil {
		key := keyFrom(int(key))
		pressed := state == C.WL_KEYBOARD_KEY_STATE_PRESSED
		keyboardHandler.KeyboardKey(key, pressed)
	}
}

//export keyboardModifiersWayland
func keyboardModifiersWayland(serial, depressed, latched, locked, group C.uint32_t) {
	// XXX
	const (
		shift = 1 << iota
		capsLock
		ctrl
		alt
	)
	if keyboardHandler != nil {
		// TODO: Track previous state to avoid
		// needless notifications.
		var modMask Modifier
		if depressed&shift != 0 {
			modMask |= ModShift
		}
		if depressed&ctrl != 0 {
			modMask |= ModCtrl
		}
		if depressed&alt != 0 {
			modMask |= ModAlt
		}
		if locked&capsLock != 0 {
			modMask |= ModCapsLock
		}
		keyboardHandler.KeyboardModifiers(modMask)
	}
}

//export keyboardRepeatInfoWayland
func keyboardRepeatInfoWayland(rate, delay C.int32_t) {
	// TODO
}

// DisplayWayland returns the Wayland display (*C.struct_wl_display).
// It must not be called if Wayland is not the platform in use.
func DisplayWayland() unsafe.Pointer { return unsafe.Pointer(dpyWayland) }

// SurfaceWayland returns the Wayland surface (*C.struct_wl_surface)
// of the given window.
// win must refer to a valid window created by NewWindow
// (note that Close invalidates the window).
// It must not be called if Wayland is not the platform in use.
func SurfaceWayland(win Window) unsafe.Pointer {
	if win != nil {
		return unsafe.Pointer(win.(*windowWayland).wsf)
	}
	return nil
}
