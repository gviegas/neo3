// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"fmt"
	"testing"
	"time"
)

func TestWSI(t *testing.T) {
	SetWindowCloseHandler(E{})
	SetWindowResizeHandler(E{})
	SetKeyboardEnterHandler(E{})
	SetKeyboardLeaveHandler(E{})
	SetKeyboardKeyHandler(E{})
	SetKeyboardModifierHandler(E{})
	SetPointerHandler(E{})
	switch PlatformInUse() {
	case None:
		win, err := NewWindow(480, 360, "Will fail")
		if win != nil || err != errMissing {
			t.Fatalf("NewWindow: win, err\nhave %v, %v\nwant nil, %v", win, err, errMissing)
		}
		if n := len(Windows()); n != 0 {
			t.Fatalf("len(Windows())\nhave %v\nwant 0", n)
		}
		// Dummy Dispatch does nothing.
		Dispatch()
		// Dummy SetAppName does nothing.
		SetAppName("Won't be displayed")
	default:
		win, err := NewWindow(480, 360, "My window")
		if err != nil {
			t.Logf("NewWindow (error): %v", err)
			return
		}
		if win == nil {
			t.Fatalf("NewWindow: win, err\nhave %v, nil\n want non-nil, nil", win)
			return
		}
		if n := len(Windows()); n != 1 {
			t.Fatalf("len(Windows())\nhave %v\nwant 1", n)
		}
		win.Unmap()
		win.Map()
		for i := 0; i < 100; i++ {
			Dispatch()
			time.Sleep(time.Millisecond * 42)
		}
		win.Resize(600, 300)
		win.SetTitle(time.Now().Format(time.RFC1123))
		if s := AppName(); s != "" {
			t.Fatalf("AppName\nhave %s\nwant \"\"", s)
		}
		SetAppName("My app")
		if s := AppName(); s != "My app" {
			t.Fatalf("AppName\nhave %s\nwant My app", s)
		}
		time.Sleep(time.Second * 2)
		win.Unmap()
		time.Sleep(time.Second + time.Second/2)
		win.Close()
		if n := len(Windows()); n != 0 {
			t.Fatalf("len(Windows())\nhave %v\nwant 0", n)
		}
	}
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
