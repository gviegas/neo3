// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/wsi"
)

func TestSwapchain(t *testing.T) {
	dim := [...][2]int{{480, 360}, {256, 256}, {600, 800}}
	win := [len(dim)]wsi.Window{}
	var err error
	for i := range dim {
		win[i], err = wsi.NewWindow(dim[i][0], dim[i][1], "My window")
		if err != nil {
			t.Fatalf("wsi.NewWindow() failed, cannot test swapchain\n%v", err)
		}
		win[i].Map()
		defer win[i].Close()
	}
	nimg := [...]int{1, 2, 3, 4, 5, 6}
	zs := swapchain{}
	for i := range win {
		for j := range nimg {
			call := fmt.Sprintf("tDrv.NewSwapchain(%v, %d)", win[i], nimg[i])
			sc, err := tDrv.NewSwapchain(win[i], nimg[j])
			if err != nil {
				t.Fatalf("(error) %s: %v", call, err)
				continue
			}
			s := sc.(*swapchain)
			if s.d != &tDrv {
				t.Fatalf("%s: s.d\nhave %p\nwant %p", call, s.d, &tDrv)
			}
			if s.sf == zs.sf {
				t.Fatalf("%s: s.sf\nhave %v\nwant valid handle", call, s.sf)
			}
			if s.sc == zs.sc {
				t.Fatalf("%s: s.sc\nhave %v\nwant valid handle", call, s.sc)
			}
			if len(s.views) == 0 {
				t.Fatalf("%s: len(s.views)\nhave 0\nwant > 0", call)
			}
			iv := s.Views()
			for i := range iv {
				if iv[i] != s.views[i] {
					t.Fatalf("s.Views()[%d]\nhave %v\nwant %v", i, iv[i], s.views[i])
				}
			}
			pf := s.Format()
			if pf != s.pf {
				t.Fatalf("s.Format()\nhave %d\nwant %d", pf, s.pf)
			}
			call = "s.Destroy()"
			s.Destroy()
			if s.d != nil {
				t.Fatalf("%s: s.d\nhave %p\nwant nil", call, s.d)
			}
			if s.sf != zs.sf {
				t.Fatalf("%s: s.sf\nhave %v\nwant null handle", call, s.sf)
			}
			if s.sc != zs.sc {
				t.Fatalf("%s: s.sc\nhave %v\nwant null handle", call, s.sc)
			}
			if len(s.views) != 0 {
				t.Fatalf("%s: len(s.views)\nhave %d\nwant 0", call, len(s.views))
			}
		}
	}
}

func TestSwapchainRecreate(t *testing.T) {
	win, err := wsi.NewWindow(800, 600, "")
	if err != nil {
		t.Fatalf("wsi.NewWindow() failed, cannot test swapchain\n%v", err)
	}
	defer win.Close()
	win.Map()
	sc, err := tDrv.NewSwapchain(win, 3)
	if err != nil {
		t.Fatalf("tDrv.NewSwapchain() failed, cannot test swapchain.Recreate()\n%v", err)
	}
	defer sc.Destroy()
	s := sc.(*swapchain)
	sf := s.sf
	qfam := s.qfam
	win.Resize(480, 360)
	s.broken = true
	err = s.Recreate()
	call := "s.Recreate()"
	if err != nil {
		t.Fatalf("(error) %s: %v", call, err)
		return
	}
	if s.broken {
		t.Fatalf("%s: s.broken\nhave true\nwant false", call)
	}
	if s.sf != sf {
		t.Fatalf("%s: s.sf changed", call)
	}
	if s.qfam != qfam {
		t.Fatalf("%s: s.qfam changed", call)
	}
}
