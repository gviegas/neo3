// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"log"
	"os"
	"runtime"
	"strings"
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
	testOpen()
	testName()
	testClose()
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
		log.Fatal("checkProcOpen(): unexpected nil error")
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
		log.Fatal("checkProcInstance(): unexpected nil error")
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
		log.Fatal("checkProcDevice(): unexpected nil error")
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

func testOpen() {
	d := Driver{}
	gpu, err := d.Open()
	defer d.Close()
	switch err {
	default:
		if d.inst != nil || d.dev != nil {
			log.Fatal("d.Open(): Driver\nhave non-zero\nwant Driver{}")
		}
		if gpu != nil {
			log.Fatal("d.Open(): GPU\nhave non-nil\nwant nil")
		}
	case nil:
		if d.inst == nil {
			log.Fatal("d.Open(): d.inst\nhave nil\nwant non-nil")
		}
		if d.ivers == 0 {
			log.Fatal("d.Open(): d.ivers\nhave 0\nwant > 0")
		}
		if d.pdev == nil {
			log.Fatal("d.Open(): d.pdev\nhave nil\nwant non-nil")
		}
		if d.dvers == 0 {
			log.Fatal("d.Open(): d.dvers\nhave 0\nwant > 0")
		}
		if d.dev == nil {
			log.Fatal("d.Open(): d.dev\nhave nil\nwant non-nil")
		}
		if d.ques == nil {
			log.Fatal("d.Open(): d.ques\nhave nil\nwant non-nil")
		}
		if len(d.mused) != int(d.mprop.memoryHeapCount) {
			log.Fatalf("d.Open(): len(d.mused)\nhave %d\nwant %d", len(d.mused), d.mprop.memoryHeapCount)
		}
		for i, n := range d.mused {
			if n != 0 {
				log.Fatalf("d.Open(): d.mused[%d]\nhave %d\nwant 0", i, n)
			}
		}
		if gpu == nil {
			log.Fatal("d.Open(): GPU\nhave nil\nwant non-nil")
		}
		if x, ok := gpu.(*Driver); ok {
			if x == nil {
				log.Fatalf("d.Open(): GPU\nhave %#v\nwant d", (*Driver)(nil))
			}
			if x != &d {
				log.Fatalf("d.Open(): GPU\nhave %p\nwant %p", x, &d)
			}
		} else {
			log.Fatalf("d.Open(): GPU\nhave %T\nwant %T", gpu, &d)
		}
	}
	// Subsequent calls to Open should return the same GPU and not fail.
	if err == nil {
		if g, e := d.Open(); g != gpu || e != nil {
			log.Fatalf("d.Open()\nhave %p, %v\nwant %p, %v", g, e, gpu, err)
		}
	} else {
		log.Print("WARNING: d.Open failed, cannot test multiple calls on open driver")
	}
}

func testName() {
	// Name should not require an open driver.
	d := &Driver{}
	s := d.Name()
	if s == "" {
		log.Fatal("d.Name()\nhave \"\"\nwant non-empty")
	} else if !strings.HasPrefix(s, "vulkan") {
		log.Fatalf("d.Name()\nhave %s\nwant vulkan*", s)
	}
	if d.inst != nil || d.dev != nil {
		log.Fatalf("d.Name(): Driver\nhave %v\nwant Driver{}", d)
	}
	// Name should not require a valid driver.
	d = nil
	defer func() {
		if x := recover(); x != nil {
			log.Fatalf("unexpected panic: %v", x)
		}
	}()
	if x := d.Name(); x != s {
		log.Fatalf("d.Name()\nhave %s\nwant %s (differs from previous call)", x, s)
	}
	// Name should not change for open driver.
	d = &Driver{}
	if _, err := d.Open(); err != nil {
		log.Print("WARNING: testName: d.Open failed")
		return
	}
	if x := d.Name(); x != s {
		log.Fatalf("d.Name()\nhave %s\nwant %s (differs from previous call)", x, s)
	}
	d.Close()
}

func testClose() {
	// Close should not require an open driver.
	d := Driver{}
	d.Close()
	// Close should set d to the zero value.
	if _, err := d.Open(); err != nil {
		log.Print("WARNING: testClose: d.Open failed")
		return
	}
	d.Close()
	if d.inst != nil || d.dev != nil {
		log.Fatalf("d.Close(): Driver\nhave %v\nwant Driver{}", d)
	}
}

func TestDriver(t *testing.T) {
	var d *Driver
	if x, ok := d.Driver().(*Driver); !ok || x != nil {
		t.Fatalf("d.Driver()\nhave %#v\nwant %#v", x, (*Driver)(nil))
	}
	d = new(Driver)
	if x := d.Driver(); x != d {
		t.Fatalf("d.Driver()\nhave %p\nwant %p", x, d)
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
		{[]string{extSurfaceS}, []string{extSurfaceS}, []int{}},
		{[]string{extSurfaceS}, []string{extSurfaceS, extDisplayS}, []int{}},
		{[]string{extDisplayS, extSurfaceS}, []string{extSurfaceS, extDisplayS}, []int{}},
		{[]string{extDisplayS}, []string{extSurfaceS, extDisplayS}, []int{}},
		{[]string{extSurfaceS}, nil, []int{0}},
		{[]string{extDisplayS}, []string{extSurfaceS}, []int{0}},
		{[]string{extSurfaceS, extDisplayS}, []string{extSurfaceS}, []int{1}},
		{[]string{extSurfaceS, extDisplayS}, []string{}, []int{0, 1}},
	}
	for _, c := range cases {
		a, f, m := selectExts(c.exts, c.from)
		if !ok(c) {
			t.Fatalf("selectExts()\nhave _, _, %v\nwant %v", m, c.want)
		}
		if len(c.want) == 0 {
			if a == nil && len(c.exts) > 0 {
				t.Fatal("selectExts()\nhave nil, _, _\nwant non-nil")
			} else if err := checkCStrings(c.exts, unsafe.Pointer(a)); err != nil {
				t.Fatal(err)
			}
		}
		if f == nil {
			t.Fatal("selectExts()\nhave _, nil, _\nwant non-nil")
		}
		f()
	}
}

func TestExtSanity(t *testing.T) {
	if !tDrv.exts[extSurface] {
		if tDrv.exts[extDisplay] {
			t.Fatal("tDrv.exts[extDisplay]\nhave true\nwant false")
		}
		if tDrv.exts[extAndroidSurface] {
			t.Fatal("tDrv.exts[extAndroidSurface]\nhave true\nwant false")
		}
		if tDrv.exts[extWaylandSurface] {
			t.Fatal("tDrv.exts[extWaylandSurface]\nhave true\nwant false")
		}
		if tDrv.exts[extWin32Surface] {
			t.Fatal("tDrv.exts[extWin32Surface]\nhave true\nwant false")
		}
		if tDrv.exts[extXCBSurface] {
			t.Fatal("tDrv.exts[extXCBSurface]\nhave true\nwant false")
		}
		if tDrv.exts[extSwapchain] {
			t.Fatal("tDrv.exts[extSwapchain]\nhave true\nwant false")
		}
	}

	if tDrv.exts[extDisplay] {
		if !tDrv.exts[extSwapchain] && tDrv.exts[extDisplaySwapchain] {
			t.Fatal("tDrv.exts[extDisplaySwapchain]\nhave true\nwant false")
		}
	} else if tDrv.exts[extDisplaySwapchain] {
		t.Fatal("tDrv.exts[extDisplaySwapchain]\nhave true\nwant false")
	}

	var bads []string
	var badi []int
	switch runtime.GOOS {
	default:
		bads = []string{extAndroidSurfaceS, extWaylandSurfaceS, extWin32SurfaceS, extXCBSurfaceS}
		badi = []int{extAndroidSurface, extWaylandSurface, extWin32Surface, extXCBSurface}
	case "android":
		bads = []string{extDisplayS, extWaylandSurfaceS, extWin32SurfaceS, extXCBSurfaceS}
		badi = []int{extDisplay, extWaylandSurface, extWin32Surface, extXCBSurface}
	case "linux":
		bads = []string{extAndroidSurfaceS, extWin32SurfaceS}
		badi = []int{extAndroidSurface, extWin32Surface}
	case "windows":
		bads = []string{extDisplayS, extAndroidSurfaceS, extWaylandSurfaceS, extXCBSurfaceS}
		badi = []int{extDisplay, extAndroidSurface, extWaylandSurface, extXCBSurface}
	}
	for i := range badi {
		if tDrv.exts[badi[i]] {
			t.Fatalf("tDrv.exts[<%s>]\nhave true\nwant false", bads[i])
		}
	}
}
