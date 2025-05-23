// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

// #cgo LDFLAGS: -lgdi32
//
// #ifndef UNICODE
// #define UNICODE
// #endif
//
// #include <windows.h>
//
// LRESULT CALLBACK wndProcWrapper(HWND, UINT, WPARAM, LPARAM);
import "C"

import (
	"errors"
	"unicode/utf16"
	"unsafe"
)

// Handle to self.
var hinst C.HINSTANCE

// Class name.
var className C.LPCWSTR

// initWin32 initializes the Win32 platform.
func initWin32() error {
	if hinst = C.GetModuleHandle(nil); hinst == nil {
		return errors.New("wsi: failed to obtain Win32 instance handle")
	}
	className = stringToLPCWSTR("neo3/wsi")
	wc := C.WNDCLASSEX{
		cbSize:        C.sizeof_WNDCLASSEX,
		style:         C.CS_HREDRAW | C.CS_VREDRAW,
		lpfnWndProc:   C.WNDPROC(C.wndProcWrapper),
		cbClsExtra:    0,
		cbWndExtra:    0,
		hInstance:     hinst,
		hIcon:         C.LoadIcon(hinst, C.IDI_APPLICATION),
		hCursor:       C.LoadCursor(nil, C.IDC_ARROW),
		hbrBackground: C.HBRUSH(C.GetStockObject(C.WHITE_BRUSH)),
		lpszMenuName:  nil,
		lpszClassName: className,
		hIconSm:       nil, // TODO
	}
	if C.RegisterClassEx(&wc) == 0 {
		C.free(unsafe.Pointer(className))
		className = nil
		hinst = nil
		return errors.New("wsi: failed to register Win32 class")
	}
	newWindow = newWindowWin32
	dispatch = dispatchWin32
	setAppName = setAppNameWin32
	platform = Win32
	return nil
}

// deinitWin32 deinitializes the Win32 platform.
func deinitWin32() {
	if windowCount > 0 {
		for _, w := range createdWindows {
			if w != nil {
				w.Close()
			}
		}
	}
	if hinst != nil {
		if className != nil {
			C.UnregisterClass(className, hinst)
			C.free(unsafe.Pointer(className))
			className = nil
		}
		hinst = nil
	}
	initDummy()
}

// windowWin32 implements Window.
type windowWin32 struct {
	hwnd   C.HWND
	width  int
	height int
	title  string
	hidden bool
}

// newWindowWin32 creates a new window.
func newWindowWin32(width, height int, title string) (Window, error) {
	wrect := C.RECT{
		left:   0,
		top:    0,
		right:  C.LONG(width),
		bottom: C.LONG(height),
	}
	if C.AdjustWindowRect(&wrect, C.WS_OVERLAPPEDWINDOW, C.FALSE) == C.FALSE {
		return nil, errors.New("wsi: failed to adjust Win32 window rect")
	}
	var (
		estyle = C.DWORD(0)
		wname  = stringToLPCWSTR(title)
		style  = C.DWORD(C.WS_OVERLAPPEDWINDOW)
		x      = C.int(C.CW_USEDEFAULT)
		y      = C.int(C.CW_USEDEFAULT)
		w      = C.int(width)
		h      = C.int(height)
	)
	hwnd := C.CreateWindowEx(estyle, className, wname, style, x, y, w, h, nil, nil, hinst, nil)
	C.free(unsafe.Pointer(wname))
	if hwnd == nil {
		return nil, errors.New("wsi: failed to create Win32 window")
	}
	return &windowWin32{
		hwnd:   hwnd,
		width:  width,
		height: height,
		title:  title,
		hidden: true,
	}, nil
}

// Show makes the window visible.
func (w *windowWin32) Show() error {
	if !w.hidden {
		return nil
	}
	C.ShowWindow(w.hwnd, C.SW_NORMAL)
	w.hidden = false
	return nil
}

// Hide hides the window.
func (w *windowWin32) Hide() error {
	if w.hidden {
		return nil
	}
	C.ShowWindow(w.hwnd, C.SW_HIDE)
	w.hidden = true
	return nil
}

// Resize resizes the window.
func (w *windowWin32) Resize(width, height int) error {
	if width == w.width && height == w.height {
		return nil
	}
	var x, y C.int
	var rect C.RECT
	if C.GetWindowRect(w.hwnd, &rect) != C.FALSE {
		x = C.int(rect.left)
		y = C.int(rect.top)
	}
	if C.MoveWindow(w.hwnd, x, y, C.int(width), C.int(height), C.TRUE) == C.FALSE {
		return errors.New("wsi: failed to resize Win32 window")
	}
	w.width = width
	w.height = height
	return nil
}

// SetTitle sets the window's title.
func (w *windowWin32) SetTitle(title string) error {
	if title == w.title {
		return nil
	}
	var err error
	ws := stringToLPCWSTR(title)
	if C.SetWindowText(w.hwnd, ws) == C.FALSE {
		err = errors.New("wsi: failed to set title of Win32 window")
	} else {
		w.title = title
	}
	C.free(unsafe.Pointer(ws))
	return err
}

// Close closes the window.
func (w *windowWin32) Close() {
	if w != nil {
		closeWindow(w)
		if w.hwnd != nil {
			C.DestroyWindow(w.hwnd)
		}
		*w = windowWin32{}
	}
}

// Width returns the window's width.
func (w *windowWin32) Width() int { return w.width }

// Height returns the window's height.
func (w *windowWin32) Height() int { return w.height }

// Title returns the window's title.
func (w *windowWin32) Title() string { return w.title }

// dispatchWin32 dispatches queued events.
func dispatchWin32() {
	var msg C.MSG
	for C.PeekMessage(&msg, nil, 0, 0, C.PM_REMOVE) != 0 {
		C.TranslateMessage(&msg)
		C.DispatchMessage(&msg)
	}
}

//export wndProcWin32
func wndProcWin32(hwnd C.HWND, msg C.UINT, wprm C.WPARAM, lprm C.LPARAM) C.LRESULT {
	switch msg {
	//case C.WM_CREATE:
	//case C.WM_PAINT:
	//case C.WM_WINDOWPOSCHANGED:
	case C.WM_CLOSE:
		closeMsgWin32(hwnd)
		return 0
	case C.WM_SIZE:
		sizeMsgWin32(hwnd, lprm)
		return 0
	case C.WM_KEYDOWN, C.WM_KEYUP:
		keyMsgWin32(wprm, lprm)
		return 0
	case C.WM_SETFOCUS:
		setFocusMsgWin32(hwnd)
		return 0
	case C.WM_KILLFOCUS:
		killFocusMsgWin32(hwnd)
		return 0
	case C.WM_LBUTTONDOWN, C.WM_LBUTTONDBLCLK:
		buttonMsgWin32(lprm, BtnLeft, true)
		return 0
	case C.WM_LBUTTONUP:
		buttonMsgWin32(lprm, BtnLeft, false)
		return 0
	case C.WM_MBUTTONDOWN, C.WM_MBUTTONDBLCLK:
		buttonMsgWin32(lprm, BtnMiddle, true)
		return 0
	case C.WM_MBUTTONUP:
		buttonMsgWin32(lprm, BtnMiddle, false)
		return 0
	case C.WM_RBUTTONDOWN, C.WM_RBUTTONDBLCLK:
		buttonMsgWin32(lprm, BtnRight, true)
		return 0
	case C.WM_RBUTTONUP:
		buttonMsgWin32(lprm, BtnRight, false)
		return 0
	case C.WM_XBUTTONDOWN, C.WM_XBUTTONDBLCLK:
		btn := BtnSide
		switch x := C.WORD(wprm >> 16 & 0xffff); {
		case x == C.XBUTTON1:
			btn = BtnForward
		case x == C.XBUTTON2:
			btn = BtnBackward
		}
		buttonMsgWin32(lprm, btn, true)
		return C.TRUE
	case C.WM_XBUTTONUP:
		btn := BtnSide
		switch x := C.WORD(wprm >> 16 & 0xffff); {
		case x == C.XBUTTON1:
			btn = BtnForward
		case x == C.XBUTTON2:
			btn = BtnBackward
		}
		buttonMsgWin32(lprm, btn, false)
		return C.TRUE
	case C.WM_MOUSEMOVE:
		mouseMoveMsgWin32(hwnd, lprm)
		return 0
	case C.WM_MOUSELEAVE:
		mouseLeaveMsgWin32(hwnd)
		return 0
	case C.WM_DESTROY:
		C.PostQuitMessage(0)
		return 0
	default:
		return C.DefWindowProc(hwnd, msg, wprm, lprm)
	}
}

// windowFromWin32 returns the window in createdWindows
// whose hwnd field matches hwnd, or nil if none does.
func windowFromWin32(hwnd C.HWND) Window {
	if hwnd == nil {
		return nil
	}
	for _, w := range createdWindows {
		if w != nil && w.(*windowWin32).hwnd == hwnd {
			return w
		}
	}
	return nil
}

// closeMsgWin32 handles WM_CLOSE messages.
func closeMsgWin32(hwnd C.HWND) {
	if win := windowFromWin32(hwnd); win != nil {
		if windowCloseHandler != nil {
			windowCloseHandler.WindowClose(win)
		}
		win.Close()
	}
}

// sizeMsgWin32 handles WM_SIZE messages.
func sizeMsgWin32(hwnd C.HWND, lprm C.LPARAM) {
	if win := windowFromWin32(hwnd); win != nil {
		win := win.(*windowWin32)
		win.width = int(lprm & 0xffff)
		win.height = int(lprm >> 16 & 0xffff)
		if windowResizeHandler != nil {
			windowResizeHandler.WindowResize(win, win.width, win.height)
		}
	}
}

// Tracks modifier state.
var modMaskWin32 Modifier

// keyMsgWin32 handles WM_KEYDOWN/WM_KEYUP messages.
func keyMsgWin32(wprm C.WPARAM, lprm C.LPARAM) {
	const (
		// ?
		low  = C.SHORT(1)
		high = ^low
	)
	prevModMask := modMaskWin32
	modMaskWin32 = 0
	if C.GetKeyState(C.VK_CAPITAL)&low != 0 {
		modMaskWin32 |= ModCapsLock
	}
	if C.GetKeyState(C.VK_SHIFT)&high != 0 {
		modMaskWin32 |= ModShift
	}
	if C.GetKeyState(C.VK_CONTROL)&high != 0 {
		modMaskWin32 |= ModCtrl
	}
	if C.GetKeyState(C.VK_MENU)&high != 0 {
		modMaskWin32 |= ModAlt
	}
	if keyboardKeyHandler != nil {
		key := keyFrom(int(wprm))
		if key == KeyUnknown {
			scan := C.UINT(lprm >> 16 & 255)
			switch wprm {
			case C.VK_SHIFT:
				if C.MapVirtualKey(C.VK_LSHIFT, C.MAPVK_VK_TO_VSC) == scan {
					key = KeyLShift
				} else {
					key = KeyRShift
				}
			case C.VK_CONTROL:
				if C.MapVirtualKey(C.VK_LCONTROL, C.MAPVK_VK_TO_VSC) == scan {
					key = KeyLCtrl
				} else {
					key = KeyRCtrl
				}
			case C.VK_MENU:
				if C.MapVirtualKey(C.VK_LMENU, C.MAPVK_VK_TO_VSC) == scan {
					key = KeyLAlt
				} else {
					key = KeyRAlt
				}
			}
		}
		pressed := lprm&(1<<31) == 0
		keyboardKeyHandler.KeyboardKey(key, pressed)
	}
	if modMaskWin32 != prevModMask && keyboardModifierHandler != nil {
		keyboardModifierHandler.KeyboardModifier(modMaskWin32)
	}
}

// setFocusMsgWin32 handles WM_SETFOCUS messages.
func setFocusMsgWin32(hwnd C.HWND) {
	if keyboardEnterHandler != nil {
		if win := windowFromWin32(hwnd); win != nil {
			keyboardEnterHandler.KeyboardEnter(win)
		}
	}
}

// killFocusMsgWin32 handles WM_KILLFOCUS messages.
func killFocusMsgWin32(hwnd C.HWND) {
	if keyboardLeaveHandler != nil {
		if win := windowFromWin32(hwnd); win != nil {
			keyboardLeaveHandler.KeyboardLeave(win)
		}
	}
}

// buttonMsgWin32 handles WM_{L,M,R,X}BUTTON{DOWN,DBLCLK,UP} messages.
func buttonMsgWin32(lprm C.LPARAM, btn Button, pressed bool) {
	if pointerButtonHandler != nil {
		//x := int(C.SHORT(lprm & 0xffff))
		//y := int(C.SHORT(lprm >> 16 & 0xffff))
		pointerButtonHandler.PointerButton(btn, pressed)
	}
}

// Tracks the window which the mouse is over, if any.
var hwndMouse C.HWND

// mouseMoveMsgWin32 handles WM_MOUSEMOVE messages.
func mouseMoveMsgWin32(hwnd C.HWND, lprm C.LPARAM) {
	newX := int(C.SHORT(lprm & 0xffff))
	newY := int(C.SHORT(lprm >> 16 & 0xffff))
	// TODO: May want to skip this if an enter handler
	// is not installed.
	if hwndMouse != hwnd {
		tme := C.TRACKMOUSEEVENT{
			cbSize:    C.sizeof_TRACKMOUSEEVENT,
			dwFlags:   C.TME_LEAVE,
			hwndTrack: hwnd,
			//dwHoverTime: C.HOVER_DEFAULT,
		}
		if C.TrackMouseEvent(&tme) == C.FALSE {
			// ?
		}
		hwndMouse = hwnd
		if pointerEnterHandler != nil {
			if win := windowFromWin32(hwnd); win != nil {
				pointerEnterHandler.PointerEnter(win, newX, newY)
			}
		}
	}
	if pointerMotionHandler != nil {
		pointerMotionHandler.PointerMotion(newX, newY)
	}
}

// mouseLeaveMsgWin32 handles WM_MOUSELEAVE messages.
func mouseLeaveMsgWin32(hwnd C.HWND) {
	if hwndMouse == hwnd {
		hwndMouse = nil
	}
	if pointerLeaveHandler != nil {
		if win := windowFromWin32(hwnd); win != nil {
			pointerLeaveHandler.PointerLeave(win)
		}
	}
}

// setAppNameWin32 updates the string used to identify the
// application.
func setAppNameWin32(s string) {
	// TODO
}

// stringToLPCWSTR converts s to UTF16 and stores it
// in the C heap.
// Call free to deallocate the wide string.
func stringToLPCWSTR(s string) C.LPCWSTR {
	var n int
	for ; n < len(s); n++ {
		if s[n] == '\x00' {
			break
		}
	}
	if n == 0 {
		return nil
	}
	n++
	var ws C.LPCWSTR
	sz := C.size_t(unsafe.Sizeof(*ws) * uintptr(n))
	ws = C.LPCWSTR(C.malloc(sz))
	u16 := utf16.Encode([]rune(s[:n-1] + "\x00"))
	C.memcpy(unsafe.Pointer(ws), unsafe.Pointer(unsafe.SliceData(u16)), sz)
	return ws
}

// HinstWin32 returns the Win32 instance/module handle (C.HINSTANCE).
// It must not be called if Win32 is not the platform in use.
func HinstWin32() unsafe.Pointer { return unsafe.Pointer(hinst) }

// HwndWin32 returns the Win32 window handle (C.HWND) of the
// given window.
// win must refer to a valid window created by NewWindow
// (note that Close invalidates the window).
// It must not be called if Win32 is not the platform in use.
func HwndWin32(win Window) unsafe.Pointer {
	if win != nil {
		return unsafe.Pointer(win.(*windowWin32).hwnd)
	}
	return nil
}
