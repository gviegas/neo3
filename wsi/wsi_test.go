// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"fmt"
	"testing"
	"time"
)

func TestWSI(t *testing.T) {
	SetWindowHandler(E{})
	SetPointerHandler(E{})
	SetKeyboardHandler(E{})
	switch PlatformInUse() {
	case None:
		win, err := NewWindow(480, 360, "Will fail")
		if win != nil || err != errMissing {
			t.Errorf("NewWindow: win, err\nhave %v, %v\nwant nil, %v", win, err, errMissing)
		}
		if n := len(Windows()); n != 0 {
			t.Errorf("len(Windows())\nhave %v\nwant 0", n)
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
			t.Errorf("NewWindow: win, err\nhave %v, nil\n want non-nil, nil", win)
			return
		}
		if n := len(Windows()); n != 1 {
			t.Errorf("len(Windows())\nhave %v\nwant 1", n)
		}
		win.Unmap()
		win.Map()
		for i := 0; i < 150; i++ {
			Dispatch()
			time.Sleep(time.Millisecond * 42)
		}
		win.Resize(600, 300)
		win.SetTitle(time.Now().Format(time.RFC1123))
		if s := AppName(); s != "" {
			t.Errorf("AppName\nhave %s\nwant \"\"", s)
		}
		SetAppName("My app")
		if s := AppName(); s != "My app" {
			t.Errorf("AppName\nhave %s\nwant My app", s)
		}
		time.Sleep(time.Second * 2)
		win.Unmap()
		time.Sleep(time.Second + time.Second/2)
		win.Close()
		if n := len(Windows()); n != 0 {
			t.Errorf("len(Windows())\nhave %v\nwant 0", n)
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

func (E) PointerIn(win Window, x, y int) {
	fmt.Printf("E.PointerIn: %v, %d, %d\n", win, x, y)
}

func (E) PointerOut(win Window) {
	fmt.Printf("E.PointerOut: %v\n", win)
}

func (E) PointerMotion(newX, newY int) {
	fmt.Printf("E.PointerMotion: %d, %d\n", newX, newY)
}

func (E) PointerButton(btn Button, pressed bool, x, y int) {
	fmt.Printf("E.PointerButton: %d, %t, %d, %d\n", btn, pressed, x, y)
}

func (E) KeyboardIn(win Window) {
	fmt.Printf("E.KeyboardIn: %v\n", win)
}

func (E) KeyboardOut(win Window) {
	fmt.Printf("E.KeyboardOut: %v\n", win)
}

func (E) KeyboardKey(key Key, pressed bool, modMask Modifier) {
	fmt.Printf("E.KeyboardKey: %d, %t, %x\n", key, pressed, modMask)
}
