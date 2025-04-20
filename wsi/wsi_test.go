// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	status := testWSI()
	if status == 0 {
		// testing: warning: no tests to run
		//status = m.Run()
	}
	os.Exit(status)
}

// NOTE: Must run on the main thread.
func testWSI() int {
	plat := PlatformInUse()
	fmt.Printf("Platform being tested is %s\n", plat)
	SetWindowCloseHandler(E{})
	SetWindowResizeHandler(E{})
	SetKeyboardEnterHandler(E{})
	SetKeyboardLeaveHandler(E{})
	SetKeyboardKeyHandler(E{})
	SetKeyboardModifierHandler(E{})
	SetPointerEnterHandler(E{})
	SetPointerLeaveHandler(E{})
	SetPointerMotionHandler(E{})
	SetPointerButtonHandler(E{})
	switch plat {
	case None:
		win, err := NewWindow(480, 360, "Will fail")
		if win != nil || err != errMissing {
			fmt.Printf("NewWindow: win, err\nhave %v, %v\nwant nil, %v", win, err, errMissing)
			return 1
		}
		if n := len(Windows()); n != 0 {
			fmt.Printf("len(Windows())\nhave %v\nwant 0", n)
			return 1
		}
		// Dummy Dispatch does nothing.
		Dispatch()
		// Dummy SetAppName does nothing.
		SetAppName("Won't be displayed")
	default:
		win, err := NewWindow(480, 360, "My window")
		if err != nil {
			fmt.Printf("NewWindow (error): %v", err)
			return 1
		}
		if win == nil {
			fmt.Printf("NewWindow: win, err\nhave %v, nil\n want non-nil, nil", win)
			return 1
		}
		if n := len(Windows()); n != 1 {
			fmt.Printf("len(Windows())\nhave %v\nwant 1", n)
			return 1
		}
		win.Unmap()
		win.Map()
		for range 100 {
			Dispatch()
			time.Sleep(time.Millisecond * time.Duration(24))
		}
		win.Resize(600, 300)
		win.SetTitle(time.Now().Format(time.RFC1123))
		if s := AppName(); s != "" {
			fmt.Printf("AppName\nhave %s\nwant \"\"", s)
			return 1
		}
		SetAppName("My app")
		if s := AppName(); s != "My app" {
			fmt.Printf("AppName\nhave %s\nwant My app", s)
			return 1
		}
		time.Sleep(time.Second * 2)
		win.Unmap()
		time.Sleep(time.Second + time.Second/2)
		win.Close()
		if n := len(Windows()); n != 0 {
			fmt.Printf("len(Windows())\nhave %v\nwant 0", n)
			return 1
		}
	}
	return 0
}

type E struct{}

func (E) WindowClose(win Window) {
	fmt.Printf("E.WindowClose: %v\n", win)
}

func (E) WindowResize(win Window, newWidth, newHeight int) {
	fmt.Printf("E.WindowResize: %v, %d, %d\n", win, newWidth, newHeight)
}

func (E) KeyboardEnter(win Window) {
	fmt.Printf("E.KeyboardEnter: %v\n", win)
}

func (E) KeyboardLeave(win Window) {
	fmt.Printf("E.KeyboardLeave: %v\n", win)
}

func (E) KeyboardKey(key Key, pressed bool) {
	fmt.Printf("E.KeyboardKey: %d, %t\n", key, pressed)
}

func (E) KeyboardModifier(modMask Modifier) {
	fmt.Printf("E.KeyboardModifier: %x\n", modMask)
}

func (E) PointerEnter(win Window, x, y int) {
	fmt.Printf("E.PointerEnter: %v, %d, %d\n", win, x, y)
}

func (E) PointerLeave(win Window) {
	fmt.Printf("E.PointerLeave: %v\n", win)
}

func (E) PointerMotion(newX, newY int) {
	fmt.Printf("E.PointerMotion: %d, %d\n", newX, newY)
}

func (E) PointerButton(btn Button, pressed bool) {
	fmt.Printf("E.PointerButton: %d, %t\n", btn, pressed)
}
