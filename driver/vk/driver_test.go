// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"log"
	"os"
	"runtime"
	"testing"
	"unsafe"
)

// tDrv is the driver managed by TestMain.
var tDrv = Driver{}

// TestMain runs the tests between calls to tDrv.Open and tDrv.Close.
func TestMain(m *testing.M) {
	// Tests that need to Open/Close their own Driver instance cannot
	// run between tDrv.Open and tDrv.Close because the C code uses
	// global variables.
	testProc()
	if _, err := tDrv.Open(); err != nil {
		log.Fatalf("Driver.Open failed: %v", err)
	}
	name := tDrv.DeviceName()
	imaj, imin, ipat := tDrv.InstanceVersion()
	dmaj, dmin, dpat := tDrv.DeviceVersion()
	log.Printf("\n\tUsing %s\n\tVersion %d.%d.%d (inst), %d.%d.%d (dev)", name, imaj, imin, ipat, dmaj, dmin, dpat)
	c := m.Run()
	tDrv.Close()
	os.Exit(c)
}

// NOTE: Must be the first test to run.
func testProc() {
	// C.getInstanceProcAddr should be valid only after proc.open is called.
	d := Driver{}
	if err := checkProcOpen(); err == nil {
		log.Fatal("checkProcOpen: unexpected nil error")
	}
	if err := d.open(); err != nil {
		log.Print("WARNING: testProc: d.open failed")
		return
	}
	if err := checkProcOpen(); err != nil {
		log.Fatal(err)
	}

	// Global and instance-level procs other than C.getInstanceProcAddr should
	// be valid only after d.initInstance is called.
	if err := checkProcInstance(); err == nil {
		log.Fatal("checkProcInstance: unexpected nil error")
	}
	if d.initInstance() != nil {
		log.Print("WARNING: testProc: d.initInstance failed")
		d.Close()
		return
	}
	if err := checkProcInstance(); err != nil {
		log.Fatal(err)
	}

	// Device-level procs other than C.getDeviceProcAddr should be valid only
	// after d.initDevice is called.
	if err := checkProcDevice(); err == nil {
		log.Fatal("checkProcDevice: unexpected nil error")
	}
	if d.initDevice() != nil {
		log.Print("WARNING: testProc: d.initDevice failed")
		d.Close()
		return
	}
	if err := checkProcDevice(); err != nil {
		log.Fatal(err)
	}

	d.Close()
	if err := checkProcClear(); err != nil {
		log.Fatal(err)
	}
}

func TestSelectExts(t *testing.T) {
	type Case struct {
		exts, from []string
		want       []int
	}
	ok := func(c Case) bool {
		have := []int{}
	extLoop:
		for i := range c.exts {
			for j := range c.from {
				if c.exts[i] == c.from[j] {
					continue extLoop
				}
			}
			have = append(have, i)
		}
		if len(c.want) != len(have) {
			return false
		}
		// Callers may assume that this slice is sorted.
		for i := range c.want {
			if c.want[i] != have[i] {
				return false
			}
		}
		return true
	}
	cases := [...]Case{
		{nil, nil, []int{}},
		{[]string{}, []string{}, []int{}},
		{nil, []string{}, []int{}},
		{[]string{}, nil, []int{}},
		{[]string{extSwapchain.name()}, []string{extSwapchain.name()}, []int{}},
		{[]string{extSwapchain.name()}, []string{extSwapchain.name(), extDynamicRendering.name()}, []int{}},
		{[]string{extDynamicRendering.name(), extSwapchain.name()}, []string{extSwapchain.name(), extDynamicRendering.name()}, []int{}},
		{[]string{extDynamicRendering.name()}, []string{extSwapchain.name(), extDynamicRendering.name()}, []int{}},
		{[]string{extSwapchain.name()}, nil, []int{0}},
		{[]string{extDynamicRendering.name()}, []string{extSwapchain.name()}, []int{0}},
		{[]string{extSwapchain.name(), extDynamicRendering.name()}, []string{extSwapchain.name()}, []int{1}},
		{[]string{extSwapchain.name(), extDynamicRendering.name()}, []string{}, []int{0, 1}},
	}
	for _, c := range cases {
		a, f, m := selectExts(c.exts, c.from)
		if !ok(c) {
			t.Fatalf("selectExts:\nhave _, _, %v\nwant %v", m, c.want)
		}
		if len(c.want) == 0 {
			if a == nil && len(c.exts) > 0 {
				t.Fatal("selectExts:\nhave nil, _, _\nwant non-nil")
			} else if err := checkCStrings(c.exts, unsafe.Pointer(a)); err != nil {
				t.Fatal(err)
			}
		}
		if f == nil {
			t.Fatal("selectExts:\nhave _, nil, _\nwant non-nil")
		}
		f()
	}
}

func TestExtSanity(t *testing.T) {
	for _, e := range globalInstanceExts.required {
		if !tDrv.exts[e] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave false\nwant true", e.name())
		}
	}
	for _, e := range globalDeviceExts.required {
		if !tDrv.exts[e] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave false\nwant true", e.name())
		}
	}
	if !tDrv.exts[extSurface] {
		if tDrv.exts[extAndroidSurface] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave true\nwant false", extAndroidSurface.name())
		}
		if tDrv.exts[extWaylandSurface] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave true\nwant false", extWaylandSurface.name())
		}
		if tDrv.exts[extWin32Surface] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave true\nwant false", extWin32Surface.name())
		}
		if tDrv.exts[extXCBSurface] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave true\nwant false", extXCBSurface.name())
		}
		if tDrv.exts[extSwapchain] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave true\nwant false", extSwapchain.name())
		}
	}
	var bad []extension
	switch runtime.GOOS {
	case "android":
		bad = []extension{extWaylandSurface, extWin32Surface, extXCBSurface}
	case "linux":
		bad = []extension{extAndroidSurface, extWin32Surface}
	case "windows":
		bad = []extension{extAndroidSurface, extWaylandSurface, extXCBSurface}
	default:
		bad = []extension{extAndroidSurface, extWaylandSurface, extWin32Surface, extXCBSurface}
	}
	for _, e := range bad {
		if tDrv.exts[e] {
			t.Fatalf("tDrv.exts[<%s>]:\nhave true\nwant false", e.name())
		}
	}
}
