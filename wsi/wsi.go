// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package wsi provides window system integration (WSI)
// for GPU drivers.
// Because a system need not have a window system, WSI
// is conditionally supported. Moreover, WSI support in
// a driver is not guaranteed.
package wsi

import (
	"errors"
)

// Window is the interface that defines a drawable window.
// The purpose of a window is to provide a surface into
// which a GPU can draw.
type Window interface {
	// Map makes the window visible.
	Map() error

	// Unmap hides the window.
	Unmap() error

	// Resize resizes the window.
	Resize(width, height int) error

	// SetTitle sets the window's title.
	SetTitle(title string) error

	// Close closes the window.
	Close()

	// Width returns the window's width.
	Width() int

	// Height returns the window's height.
	Height() int

	// Title returns the window's title.
	Title() string
}

// NewWindow creates a new window.
func NewWindow(width, height int, title string) (Window, error) {
	if windowCount >= MaxWindows {
		return nil, errors.New("wsi: too many windows")
	}
	win, err := newWindow(width, height, title)
	if err != nil {
		return nil, err
	}
	for i := range createdWindows {
		if createdWindows[i] == nil {
			createdWindows[i] = win
			windowCount++
			break
		}
	}
	return win, nil
}

var newWindow func(int, int, string) (Window, error)

// The maximum number of windows that can exist at any
// given time.
const MaxWindows = 16

// Windows returns all created windows.
// The returned value becomes out of date after calls to
// NewWindow and Window.Close.
func Windows() []Window {
	if windowCount == 0 {
		return nil
	}
	wins := make([]Window, 0, windowCount)
	for i := range createdWindows {
		if createdWindows[i] != nil {
			wins = append(wins, createdWindows[i])
		}
	}
	return wins
}

// closeWindow removes win from createdWindows and
// decrements windowCount.
// It must be called by implementations on win.Close.
// Note that win must be comparable.
func closeWindow(win Window) {
	for i := range createdWindows {
		if createdWindows[i] == win {
			createdWindows[i] = nil
			windowCount--
			return
		}
	}
}

var (
	windowCount    int
	createdWindows [MaxWindows]Window
)

// Key is the type of keyboard keys.
type Key int

// Keyboard keys.
const (
	KeyUnknown Key = iota
	KeyGrave
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	Key0
	KeyMinus
	KeyEqual
	KeyBackspace
	KeyTab
	KeyQ
	KeyW
	KeyE
	KeyR
	KeyT
	KeyY
	KeyU
	KeyI
	KeyO
	KeyP
	KeyLBracket
	KeyRBracket
	KeyBackslash
	KeyCapsLock
	KeyA
	KeyS
	KeyD
	KeyF
	KeyG
	KeyH
	KeyJ
	KeyK
	KeyL
	KeySemicolon
	KeyApostrophe
	KeyReturn
	KeyLShift
	KeyZ
	KeyX
	KeyC
	KeyV
	KeyB
	KeyN
	KeyM
	KeyComma
	KeyDot
	KeySlash
	KeyRShift
	KeyLCtrl
	KeyLAlt
	KeyLMeta
	KeySpace
	KeyRMeta
	KeyRAlt
	KeyRCtrl
	KeyEsc
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyInsert
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeySysrq
	KeyScrollLock
	KeyPause
	KeyPadNumLock
	KeyPadSlash
	KeyPadStar
	KeyPadMinus
	KeyPadPlus
	KeyPad1
	KeyPad2
	KeyPad3
	KeyPad4
	KeyPad5
	KeyPad6
	KeyPad7
	KeyPad8
	KeyPad9
	KeyPad0
	KeyPadDot
	KeyPadEnter
	KeyPadEqual
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyF21
	KeyF22
	KeyF23
	KeyF24
)

// Modifier is the type of modifier flags.
type Modifier int

// Modifier flags.
const (
	ModCapsLock Modifier = 1 << iota
	ModShift
	ModCtrl
	ModAlt
)

// Button is the type of pointer buttons.
type Button int

// Pointer buttons.
const (
	BtnUnknown Button = iota
	BtnLeft
	BtnRight
	BtnMiddle
	BtnSide
	BtnForward
	BtnBackward
)

// WindowCloseHandler is a callback for window close events.
type WindowCloseHandler interface {
	// WindowClose is called when a window is closed.
	WindowClose(win Window)
}

// WindowResizeHandler is a callback for window resize events.
type WindowResizeHandler interface {
	// WindowResize is called when a window is resized.
	WindowResize(win Window, newWidth, newHeight int)
}

// WindowHandler is an interface that groups all window callbacks.
type WindowHandler interface {
	WindowCloseHandler
	WindowResizeHandler
}

// SetWindowCloseHandler sets the window close handler.
func SetWindowCloseHandler(wh WindowCloseHandler) { windowCloseHandler = wh }

// SetWindowResizeHandler sets the window resize handler.
func SetWindowResizeHandler(wh WindowResizeHandler) { windowResizeHandler = wh }

// SetWindowHandler sets the window handler.
func SetWindowHandler(wh WindowHandler) {
	SetWindowCloseHandler(wh)
	SetWindowResizeHandler(wh)
}

var (
	windowCloseHandler  WindowCloseHandler
	windowResizeHandler WindowResizeHandler
)

// KeyboardEnterHandler is a callback for keyboard enter events.
type KeyboardEnterHandler interface {
	// KeyboardEnter is called when focus is gained.
	KeyboardEnter(win Window)
}

// KeyboardLeaveHandler is a callback for keyboard leave events.
type KeyboardLeaveHandler interface {
	// KeyboardLeave is called when focus is lost.
	KeyboardLeave(win Window)
}

// KeyboardKeyHandler is a callback for keyboard key events.
type KeyboardKeyHandler interface {
	// KeyboardKey is called when a key is pressed/released.
	KeyboardKey(key Key, pressed bool)
}

// KeyboardModifierHandler is a callback for keyboard modifier events.
type KeyboardModifierHandler interface {
	// KeyboardModifier is called when modifier state changes.
	KeyboardModifier(modMask Modifier)
}

// SetKeyboardEnterHandler sets the keyboard enter handler.
func SetKeyboardEnterHandler(kh KeyboardEnterHandler) { keyboardEnterHandler = kh }

// SetKeyboardLeaveHandler sets the keyboard leave handler.
func SetKeyboardLeaveHandler(kh KeyboardLeaveHandler) { keyboardLeaveHandler = kh }

// SetKeyboardKeyHandler sets the keyboard key handler.
func SetKeyboardKeyHandler(kh KeyboardKeyHandler) { keyboardKeyHandler = kh }

// SetKeyboardModifierHandler sets the keyboard modifier handler.
func SetKeyboardModifierHandler(kh KeyboardModifierHandler) { keyboardModifierHandler = kh }

var (
	keyboardEnterHandler    KeyboardEnterHandler
	keyboardLeaveHandler    KeyboardLeaveHandler
	keyboardKeyHandler      KeyboardKeyHandler
	keyboardModifierHandler KeyboardModifierHandler
)

// PointerEnterHandler is a callback for pointer enter events.
type PointerEnterHandler interface {
	// PointerEnter is called when the pointer enters a window.
	PointerEnter(win Window, x, y int)
}

// PointerLeaveHandler is a callback for pointer leave events.
type PointerLeaveHandler interface {
	// PointerLeave is called when the pointer leaves a window.
	PointerLeave(win Window)
}

// PointerMotionHandler is a callback for pointer motion events.
type PointerMotionHandler interface {
	// PointerMotion is called when the pointer changes position.
	PointerMotion(newX, newY int)
}

// PointerButtonHandler is a callback for pointer button events.
type PointerButtonHandler interface {
	// PointerButton is called when a button is pressed/released.
	PointerButton(btn Button, pressed bool)
}

// SetPointerEnterHandler sets the pointer enter handler.
func SetPointerEnterHandler(ph PointerEnterHandler) { pointerEnterHandler = ph }

// SetPointerLeaveHandler sets the pointer leave handler.
func SetPointerLeaveHandler(ph PointerLeaveHandler) { pointerLeaveHandler = ph }

// SetPointerMotionHandler sets the pointer motion handler.
func SetPointerMotionHandler(ph PointerMotionHandler) { pointerMotionHandler = ph }

// SetPointerButtonHandler sets the pointer button handler.
func SetPointerButtonHandler(ph PointerButtonHandler) { pointerButtonHandler = ph }

var (
	pointerEnterHandler  PointerEnterHandler
	pointerLeaveHandler  PointerLeaveHandler
	pointerMotionHandler PointerMotionHandler
	pointerButtonHandler PointerButtonHandler
)

// Dispatch dispatches queued events.
func Dispatch() {
	dispatch()
}

var dispatch func()

// TODO: Add a Wait function with timeout.

// AppName returns the string used to identify the application.
// Its use is platform-specific.
func AppName() string {
	return appName
}

// SetAppName updates the string used to identify the
// application.
func SetAppName(s string) {
	setAppName(s)
	appName = s
}

var (
	appName    string
	setAppName func(string)
)

// Platform identifies an underlying platform used to
// implement wsi.
type Platform int

// Platforms.
const (
	// None means that wsi is not available.
	// In this case, calls to NewWindow will
	// always fail, and calls to Dispatch
	// will do nothing.
	None Platform = iota
	Android
	Wayland
	Win32
	XCB
)

// PlatformInUse identifies the underlying platform which
// wsi is using.
func PlatformInUse() Platform {
	return platform
}

var platform Platform
