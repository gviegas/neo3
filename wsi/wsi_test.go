// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"testing"
	"time"
)

func TestWSI(t *testing.T) {
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
		for i := 0; i < 100; i++ {
			Dispatch()
			time.Sleep(time.Millisecond * 15)
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
