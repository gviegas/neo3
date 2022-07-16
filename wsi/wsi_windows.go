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
	"unsafe"
)

// Handle to self.
var hinst C.HINSTANCE

// Class name.
var className C.LPCWSTR

// initWin32 initializes the Win32 platform.
func initWin32() error {
	if hinst = C.GetModuleHandle(nil); hinst == nil {
		return errors.New("Failed to obtain Win32 instance handle")
	}
	// TODO
	n := C.size_t(unsafe.Sizeof(*className) * 16)
	className = C.LPCWSTR(C.malloc(n))
	s := unsafe.Slice(className, n)
	s[0] = 'w'
	s[1] = 's'
	s[2] = 'i'
	s[3] = 0
	wc := C.WNDCLASS{
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
	}
	if C.RegisterClass(&wc) == 0 {
		C.free(unsafe.Pointer(className))
		className = nil
		hinst = nil
		return errors.New("Failed to register Win32 class")
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
	mapped bool
}

// newWindowWin32 creates a new window.
func newWindowWin32(width, height int, title string) (Window, error) {
	// TODO
	var (
		estyle = C.DWORD(0)
		ptitle = className // C.LPCWSTR(nil)
		style  = C.DWORD(C.WS_OVERLAPPEDWINDOW)
		x      = C.int(C.CW_USEDEFAULT)
		y      = C.int(C.CW_USEDEFAULT)
		w      = C.int(width)
		h      = C.int(height)
	)
	hwnd := C.CreateWindowEx(estyle, className, ptitle, style, x, y, w, h, nil, nil, hinst, nil)
	if hwnd == nil {
		return nil, errors.New("Failed to create Win32 window")
	}
	return &windowWin32{
		hwnd:   hwnd,
		width:  width,
		height: height,
		title:  title,
		mapped: false,
	}, nil
}

// Map makes the window visible.
func (w *windowWin32) Map() error {
	if w.mapped {
		return nil
	}
	C.ShowWindow(w.hwnd, C.SW_NORMAL)
	w.mapped = true
	return nil
}

// Unmap hides the window.
func (w *windowWin32) Unmap() error {
	if !w.mapped {
		return nil
	}
	C.ShowWindow(w.hwnd, C.SW_HIDE)
	w.mapped = false
	return nil
}

// Resize resizes the window.
func (w *windowWin32) Resize(width, height int) error {
	// TODO
	panic("not implemented")
}

// SetTitle sets the window's title.
func (w *windowWin32) SetTitle(title string) error {
	// TODO
	panic("not implemented")
}

// Close closes the window.
func (w *windowWin32) Close() {
	if w == nil {
		return
	}
	if w.hwnd != nil {
		C.DestroyWindow(w.hwnd)
	}
	*w = windowWin32{}
}

// Width returns the window's width.
func (w *windowWin32) Width() int {
	// TODO
	panic("not implemented")
}

// Height returns the window's height.
func (w *windowWin32) Height() int {
	// TODO
	panic("not implemented")
}

// Title returns the window's title.
func (w *windowWin32) Title() string {
	// TODO
	panic("not implemented")
}

// dispatchWin32 dispatches queued events.
func dispatchWin32() {
	// TODO
	var msg C.MSG
	for C.PeekMessage(&msg, nil, 0, 0, C.PM_REMOVE) != 0 {
		C.TranslateMessage(&msg)
		C.DispatchMessage(&msg)
	}
}

//export wndProcWin32
func wndProcWin32(hwnd C.HWND, msg C.UINT, wprm C.WPARAM, lprm C.LPARAM) C.LRESULT {
	// TODO
	switch msg {
	//case C.WM_CREATE:
	//case C.WM_PAINT:
	//case C.WM_SIZE:
	case C.WM_DESTROY:
		C.PostQuitMessage(0)
		return 0
	default:
		return C.DefWindowProc(hwnd, msg, wprm, lprm)
	}
}

// setAppNameWin32 updates the string used to identify the
// application.
func setAppNameWin32(s string) {
	// TODO
	panic("not implemented")
}
