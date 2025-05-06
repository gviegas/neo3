// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build unix && !android && !darwin

package wsi

// #cgo linux LDFLAGS: -ldl
// #include <dlfcn.h>
// #include <stdlib.h>
// #include <string.h>
// #include <wsi_xcb.h>
import "C"

import (
	"errors"
	"unsafe"
)

// Handle for the shared object.
var hXCB unsafe.Pointer

// Common XCB variables.
var (
	connXCB      *C.xcb_connection_t
	visualXCB    C.xcb_visualid_t
	rootXCB      C.xcb_window_t
	whitePixXCB  C.uint32_t
	blackPixXCB  C.uint32_t
	protoAtomXCB C.xcb_atom_t
	delAtomXCB   C.xcb_atom_t
	titleAtomXCB C.xcb_atom_t
	utf8AtomXCB  C.xcb_atom_t
	classAtomXCB C.xcb_atom_t
)

// openXCB opens the shared library and gets function pointers.
// It is not safe to call any of the C wrappers unless this
// function succeeds.
func openXCB() error {
	if hXCB == nil {
		lib := C.CString("libxcb.so.1")
		defer C.free(unsafe.Pointer(lib))
		hXCB := C.dlopen(lib, C.RTLD_LAZY|C.RTLD_LOCAL)
		if hXCB == nil {
			return errors.New("wsi: failed to open libxcb")
		}
		for i := range C.nameXCB {
			C.ptrXCB[i] = C.dlsym(hXCB, C.nameXCB[i])
			if C.ptrXCB[i] == nil {
				C.dlclose(hXCB)
				hXCB = nil
				return errors.New("wsi: failed to fetch XCB symbol")
			}
		}
	}
	return nil
}

// closeXCB closes the shared library.
// It is not safe to call any of the C wrappers after
// calling this function.
func closeXCB() {
	if hXCB != nil {
		C.dlclose(hXCB)
		hXCB = nil
	}
}

// initXCB initializes the XCB platform.
func initXCB() error {
	if connXCB != nil {
		return nil
	}
	if err := openXCB(); err != nil {
		return err
	}

	connXCB = C.connectXCB(nil, nil)
	if res := C.connectionHasErrorXCB(connXCB); res != 0 {
		if connXCB != nil {
			C.disconnectXCB(connXCB)
			connXCB = nil
		}
		return errors.New("wsi: connectXCB failed")
	}
	setup := C.getSetupXCB(connXCB)
	if setup == nil {
		C.disconnectXCB(connXCB)
		connXCB = nil
		return errors.New("wsi: getSetupXCB failed")
	}
	screenIt := C.setupRootsIteratorXCB(setup)
	visualXCB = screenIt.data.root_visual
	rootXCB = screenIt.data.root
	whitePixXCB = screenIt.data.white_pixel
	blackPixXCB = screenIt.data.black_pixel

	var genErr *C.xcb_generic_error_t

	atoms := [...]struct {
		name *C.char
		dst  *C.xcb_atom_t
	}{
		{C.CString("WM_PROTOCOLS"), &protoAtomXCB},
		{C.CString("WM_DELETE_WINDOW"), &delAtomXCB},
		{C.CString("WM_NAME"), &titleAtomXCB},
		{C.CString("UTF8_STRING"), &utf8AtomXCB},
		{C.CString("WM_CLASS"), &classAtomXCB},
	}
	for i := range atoms {
		defer C.free(unsafe.Pointer(atoms[i].name))
		nameLen := C.uint16_t(C.strlen(atoms[i].name))
		cookie := C.internAtomXCB(connXCB, 0, nameLen, atoms[i].name)
		reply := C.internAtomReplyXCB(connXCB, cookie, &genErr)
		defer C.free(unsafe.Pointer(reply))
		if genErr != nil || reply == nil {
			C.free(unsafe.Pointer(genErr))
			C.disconnectXCB(connXCB)
			connXCB = nil
			return errors.New("wsi: internAtomXCB failed")
		}
		*atoms[i].dst = reply.atom
	}

	// NOTE: This leaks to the whole desktop environment in
	// certain cases, so turning it off for now.
	if false {
		valMask := C.uint32_t(C.XCB_KB_AUTO_REPEAT_MODE)
		valList := C.uint32_t(C.XCB_AUTO_REPEAT_MODE_OFF)
		cookie := C.changeKeyboardControlCheckedXCB(connXCB, valMask, unsafe.Pointer(&valList))
		genErr = C.requestCheckXCB(connXCB, cookie)
		if genErr != nil {
			C.free(unsafe.Pointer(genErr))
			C.disconnectXCB(connXCB)
			connXCB = nil
			return errors.New("wsi: changeKeyboardControlCheckedXCB failed")
		}
	}

	if C.flushXCB(connXCB) <= 0 {
		C.disconnectXCB(connXCB)
		connXCB = nil
		return errors.New("wsi: flushXCB failed")
	}

	newWindow = newWindowXCB
	dispatch = dispatchXCB
	setAppName = setAppNameXCB
	platform = XCB
	return nil
}

// deinitXCB deinitializes the XCB platform.
func deinitXCB() {
	if windowCount > 0 {
		for _, w := range createdWindows {
			if w != nil {
				w.Close()
			}
		}
	}
	if connXCB != nil {
		C.disconnectXCB(connXCB)
		connXCB = nil
	}
	closeXCB()
	initDummy()
}

// windowXCB implements Window.
type windowXCB struct {
	id     C.xcb_window_t
	width  int
	height int
	title  string
	hidden bool
}

// newWindowXCB creates a new window.
func newWindowXCB(width, height int, title string) (Window, error) {
	id := C.generateIdXCB(connXCB)
	wdt := C.uint16_t(width)
	hgt := C.uint16_t(height)
	wclass := C.uint16_t(C.XCB_WINDOW_CLASS_INPUT_OUTPUT)
	valMask := C.uint32_t(C.XCB_CW_BACK_PIXEL | C.XCB_CW_EVENT_MASK)
	valList := [2]C.uint32_t{
		0: whitePixXCB,
		1: C.XCB_EVENT_MASK_KEY_PRESS | C.XCB_EVENT_MASK_KEY_RELEASE | C.XCB_EVENT_MASK_BUTTON_PRESS | C.XCB_EVENT_MASK_BUTTON_RELEASE |
			C.XCB_EVENT_MASK_ENTER_WINDOW | C.XCB_EVENT_MASK_LEAVE_WINDOW | C.XCB_EVENT_MASK_POINTER_MOTION | C.XCB_EVENT_MASK_BUTTON_MOTION |
			C.XCB_EVENT_MASK_EXPOSURE | C.XCB_EVENT_MASK_STRUCTURE_NOTIFY | C.XCB_EVENT_MASK_FOCUS_CHANGE,
	}
	cookie := C.createWindowCheckedXCB(connXCB, 0, id, rootXCB, 0, 0, wdt, hgt, 0, wclass, visualXCB, valMask, unsafe.Pointer(&valList[0]))
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		if id != 0 {
			C.destroyWindowXCB(connXCB, id)
		}
		return nil, errors.New("wsi: createWindowCheckedXCB failed")
	}

	if err := setTitleXCB(title, id); err != nil {
		C.destroyWindowXCB(connXCB, id)
		return nil, err
	}
	class := [2]string{"", ""}
	if err := setClassXCB(class, id); err != nil {
		C.destroyWindowXCB(connXCB, id)
		return nil, err
	}
	if err := setDeleteXCB(true, id); err != nil {
		C.destroyWindowXCB(connXCB, id)
		return nil, err
	}

	return &windowXCB{
		id:     id,
		width:  width,
		height: height,
		title:  title,
		hidden: true,
	}, nil
}

// Show makes the window visible.
func (w *windowXCB) Show() error {
	if !w.hidden {
		return nil
	}
	cookie := C.mapWindowCheckedXCB(connXCB, w.id)
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		return errors.New("wsi: mapWindowCheckedXCB failed")
	}
	w.hidden = false
	return nil
}

// Hide hides the window.
func (w *windowXCB) Hide() error {
	if w.hidden {
		return nil
	}
	cookie := C.unmapWindowCheckedXCB(connXCB, w.id)
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		return errors.New("wsi: unmapWindowCheckedXCB failed")
	}
	w.hidden = true
	return nil
}

// Resize resizes the window.
func (w *windowXCB) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return errors.New("wsi: width/height less than or equal 0")
	}
	if width == w.width && height == w.height {
		return nil
	}
	valMask := C.uint32_t(C.XCB_CONFIG_WINDOW_WIDTH | C.XCB_CONFIG_WINDOW_HEIGHT)
	valList := [2]C.uint32_t{C.uint32_t(width), C.uint32_t(height)}
	cookie := C.configureWindowCheckedXCB(connXCB, w.id, valMask, unsafe.Pointer(&valList[0]))
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		return errors.New("wsi: configureWindowCheckedXCB failed")
	}
	w.width = width
	w.height = height
	return nil
}

// SetTitle sets the window's title.
func (w *windowXCB) SetTitle(title string) error {
	if title != w.title {
		if err := setTitleXCB(title, w.id); err != nil {
			return err
		}
		w.title = title
	}
	return nil
}

// Close closes the window.
func (w *windowXCB) Close() {
	if w != nil {
		closeWindow(w)
		if connXCB != nil {
			C.destroyWindowXCB(connXCB, w.id)
		}
		*w = windowXCB{}
	}
}

// Width returns the window's width.
func (w *windowXCB) Width() int { return w.width }

// Height returns the windows's height.
func (w *windowXCB) Height() int { return w.height }

// Title returns the window's title.
func (w *windowXCB) Title() string { return w.title }

// setTitleXCB sets the title of the given window.
func setTitleXCB(title string, id C.xcb_window_t) error {
	s := C.CString(title)
	defer C.free(unsafe.Pointer(s))
	slen := C.uint32_t(C.strlen(s))
	cookie := C.changePropertyCheckedXCB(connXCB, C.XCB_PROP_MODE_REPLACE, id, titleAtomXCB, utf8AtomXCB, 8, slen, unsafe.Pointer(s))
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		return errors.New("wsi: changePropertyCheckedXCB failed")
	}
	return nil
}

// setClassXCB sets the class of the given window.
func setClassXCB(class [2]string, id C.xcb_window_t) error {
	var s []byte
	if len(class[0]) > 0 {
		s = append(s, class[0]...)
	}
	s = append(s, 0)
	s = append(s, class[1]...)
	s = append(s, 0)
	slen := C.uint32_t(len(s))
	cookie := C.changePropertyCheckedXCB(connXCB, C.XCB_PROP_MODE_REPLACE, id, classAtomXCB, C.XCB_ATOM_STRING, 8, slen, unsafe.Pointer(unsafe.SliceData(s)))
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		return errors.New("wsi: changePropertyCheckedXCB failed")
	}
	return nil
}

// setDeleteXCB sets the delete property of the given window.
func setDeleteXCB(t bool, id C.xcb_window_t) error {
	var atom C.xcb_atom_t
	if t {
		atom = delAtomXCB
	} else {
		atom = C.XCB_ATOM_NONE
	}
	cookie := C.changePropertyCheckedXCB(connXCB, C.XCB_PROP_MODE_REPLACE, id, protoAtomXCB, C.XCB_ATOM_ATOM, 32, 1, unsafe.Pointer(&atom))
	genErr := C.requestCheckXCB(connXCB, cookie)
	if genErr != nil {
		C.free(unsafe.Pointer(genErr))
		return errors.New("wsi: changePropertyCheckedXCB failed")
	}
	return nil
}

// pollXCB process the next event.
// It returns false if there are no events to process.
func pollXCB() bool {
	event := C.pollForEventXCB(connXCB)
	if event != nil {
		defer C.free(unsafe.Pointer(event))
		typ := event.response_type & 127
		switch typ {
		case C.XCB_KEY_PRESS, C.XCB_KEY_RELEASE:
			keyEventXCB(event)
		case C.XCB_BUTTON_PRESS, C.XCB_BUTTON_RELEASE:
			buttonEventXCB(event)
		case C.XCB_MOTION_NOTIFY:
			motionEventXCB(event)
		case C.XCB_ENTER_NOTIFY:
			enterEventXCB(event)
		case C.XCB_LEAVE_NOTIFY:
			leaveEventXCB(event)
		case C.XCB_FOCUS_IN:
			focusInEventXCB(event)
		case C.XCB_FOCUS_OUT:
			focusOutXCB(event)
		case C.XCB_EXPOSE:
			// TODO
		case C.XCB_CONFIGURE_NOTIFY:
			configureEventXCB(event)
		case C.XCB_CLIENT_MESSAGE:
			clientEventXCB(event)
		}
		return true
	}
	return false
}

// windowFromXCB returns the window in createdWindows
// whose id field matches id, or nil if none does.
func windowFromXCB(id C.xcb_window_t) Window {
	for _, w := range createdWindows {
		if w != nil && w.(*windowXCB).id == id {
			return w
		}
	}
	return nil
}

// Tracks modifier state.
var (
	modCapsXCB  Modifier
	modLeftXCB  Modifier
	modRightXCB Modifier
)

// keyEventXCB handles key press/release events.
func keyEventXCB(event *C.xcb_generic_event_t) {
	evt := (*C.xcb_key_press_event_t)(unsafe.Pointer(event))
	key := keyFrom(int(evt.detail - 8)) // XXX
	pressed := evt.response_type&127 == C.XCB_KEY_PRESS
	prevModMask := modCapsXCB | modLeftXCB | modRightXCB
	if pressed {
		switch key {
		case KeyCapsLock:
			if evt.state&C.XCB_MOD_MASK_LOCK == 0 {
				modCapsXCB = ModCapsLock
			} else {
				modCapsXCB = 0
			}
		case KeyLShift:
			modLeftXCB |= ModShift
		case KeyRShift:
			modRightXCB |= ModShift
		case KeyLCtrl:
			modLeftXCB |= ModCtrl
		case KeyRCtrl:
			modRightXCB |= ModCtrl
		case KeyLAlt:
			modLeftXCB |= ModAlt
		case KeyRAlt:
			modRightXCB |= ModAlt
		}
	} else {
		switch key {
		case KeyLShift:
			modLeftXCB &^= ModShift
		case KeyRShift:
			modRightXCB &^= ModShift
		case KeyLCtrl:
			modLeftXCB &^= ModCtrl
		case KeyRCtrl:
			modRightXCB &^= ModCtrl
		case KeyLAlt:
			modLeftXCB &^= ModAlt
		case KeyRAlt:
			modRightXCB &^= ModAlt
		}
	}
	if keyboardKeyHandler != nil {
		keyboardKeyHandler.KeyboardKey(key, pressed)
	}
	modMask := modCapsXCB | modLeftXCB | modRightXCB
	if modMask != prevModMask && keyboardModifierHandler != nil {
		keyboardModifierHandler.KeyboardModifier(modMask)
	}
}

// buttonEventXCB handles button press/release events.
func buttonEventXCB(event *C.xcb_generic_event_t) {
	if pointerButtonHandler != nil {
		evt := (*C.xcb_button_press_event_t)(unsafe.Pointer(event))
		btn := BtnUnknown
		switch evt.detail {
		case C.XCB_BUTTON_INDEX_1:
			btn = BtnLeft
		case C.XCB_BUTTON_INDEX_2:
			btn = BtnMiddle
		case C.XCB_BUTTON_INDEX_3:
			btn = BtnRight
		case C.XCB_BUTTON_INDEX_4, C.XCB_BUTTON_INDEX_5:
			// TODO: Scroll.
		}
		pressed := evt.response_type&127 == C.XCB_BUTTON_PRESS
		pointerButtonHandler.PointerButton(btn, pressed)
	}
}

// motionEventXCB handles motion notify events.
func motionEventXCB(event *C.xcb_generic_event_t) {
	if pointerMotionHandler != nil {
		evt := (*C.xcb_motion_notify_event_t)(unsafe.Pointer(event))
		newX := int(evt.event_x)
		newY := int(evt.event_y)
		pointerMotionHandler.PointerMotion(newX, newY)
	}
}

// enterEventXCB handles enter notify events.
func enterEventXCB(event *C.xcb_generic_event_t) {
	if pointerEnterHandler != nil {
		evt := (*C.xcb_enter_notify_event_t)(unsafe.Pointer(event))
		win := windowFromXCB(evt.event)
		x := int(evt.event_x)
		y := int(evt.event_y)
		pointerEnterHandler.PointerEnter(win, x, y)
	}
}

// leaveEventXCB handles leave notify events.
func leaveEventXCB(event *C.xcb_generic_event_t) {
	if pointerLeaveHandler != nil {
		evt := (*C.xcb_leave_notify_event_t)(unsafe.Pointer(event))
		win := windowFromXCB(evt.event)
		pointerLeaveHandler.PointerLeave(win)
	}
}

// focusInXCB handles focus in events.
func focusInEventXCB(event *C.xcb_generic_event_t) {
	if keyboardEnterHandler != nil {
		evt := (*C.xcb_focus_in_event_t)(unsafe.Pointer(event))
		win := windowFromXCB(evt.event)
		keyboardEnterHandler.KeyboardEnter(win)
	}
}

// focusOutXCB handles focus out events.
func focusOutXCB(event *C.xcb_generic_event_t) {
	if keyboardLeaveHandler != nil {
		evt := (*C.xcb_focus_out_event_t)(unsafe.Pointer(event))
		win := windowFromXCB(evt.event)
		keyboardLeaveHandler.KeyboardLeave(win)
	}
}

// configureEventXCB handles configure notify events.
func configureEventXCB(event *C.xcb_generic_event_t) {
	evt := (*C.xcb_configure_notify_event_t)(unsafe.Pointer(event))
	win := windowFromXCB(evt.event)
	newWidth := int(evt.width)
	newHeight := int(evt.height)
	if win != nil {
		win := win.(*windowXCB)
		win.width = newWidth
		win.height = newHeight
	}
	if windowResizeHandler != nil {
		windowResizeHandler.WindowResize(win, newWidth, newHeight)
	}
}

// clientEventXCB handles client message events.
func clientEventXCB(event *C.xcb_generic_event_t) {
	if windowCloseHandler != nil {
		evt := (*C.xcb_client_message_event_t)(unsafe.Pointer(event))
		data := *(*C.uint32_t)(unsafe.Pointer(&evt.data))
		if evt._type == protoAtomXCB && data == delAtomXCB {
			win := windowFromXCB(evt.window)
			windowCloseHandler.WindowClose(win)
		}
	}
}

// dispatchXCB dispatches queued events.
func dispatchXCB() {
	for pollXCB() {
	}
}

// setAppNameXCB updates the string used to identify the
// application.
func setAppNameXCB(s string) {
	class := [2]string{s, s}
	for _, w := range createdWindows {
		if w != nil {
			setClassXCB(class, w.(*windowXCB).id)
		}
	}
}

// ConnXCB returns the XCB connection (*C.xcb_connection_t).
// It must not be called if XCB is not the platform is use.
func ConnXCB() unsafe.Pointer { return unsafe.Pointer(connXCB) }

// VisualXCB returns the XCB visual ID (C.xcb_visualid_t).
// It must not be called if XCB is not the platform is use.
func VisualXCB() uint32 { return uint32(visualXCB) }

// WindowXCB returns the XCB window ID (C.xcb_window_t) of
// the given window.
// win must refer to a valid window created by NewWindow
// (note that Close invalidates the window).
// It must not be called if XCB is not the platform is use.
func WindowXCB(win Window) uint32 {
	if win != nil {
		return uint32(win.(*windowXCB).id)
	}
	return 0
}
