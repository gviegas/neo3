// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

func TestOpen(t *testing.T) {
	d := Driver{}
	gpu, err := d.Open()
	defer d.Close()
	t.Logf("d.Open()\n%+v", gpu)
	switch err {
	default:
		if d.inst != nil || d.dev != nil {
			t.Error("d.Open(): Driver\nhave non-zero\nwant Driver{}")
		}
		if gpu != nil {
			t.Error("d.Open(): GPU\nhave non-nil\nwant nil")
		}
	case nil:
		if d.inst == nil {
			t.Error("d.Open(): d.inst\nhave nil\nwant non-nil")
		}
		if d.ivers == 0 {
			t.Error("d.Open(): d.ivers\nhave 0\nwant > 0")
		}
		if d.pdev == nil {
			t.Error("d.Open(): d.pdev\nhave nil\nwant non-nil")
		}
		if d.dvers == 0 {
			t.Error("d.Open(): d.dvers\nhave 0\nwant > 0")
		}
		if d.dev == nil {
			t.Error("d.Open(): d.dev\nhave nil\nwant non-nil")
		}
		if d.ques == nil {
			t.Error("d.Open(): d.ques\nhave nil\nwant non-nil")
		}
		if len(d.mused) == 0 {
			t.Error("d.Open(): len(d.mused)\nhave 0\nwant > 0")
		}
		if gpu == nil {
			t.Error("d.Open(): GPU\nhave nil\nwant non-nil")
		} else if x, ok := gpu.(*Driver); ok {
			if x == nil {
				t.Errorf("d.Open(): GPU\nhave %#v\nwant d", (*Driver)(nil))
			} else if x != &d {
				t.Errorf("d.Open(): GPU\nhave %p\nwant %p", x, &d)
			}
		} else {
			t.Errorf("d.Open(): GPU\nhave %T\nwant %T", gpu, &d)
		}
	}
	// Subsequent calls to Open should return the same GPU and not fail.
	if err == nil {
		if g, e := d.Open(); g != gpu || e != nil {
			t.Errorf("d.Open()\nhave %p, %v\nwant %p, %v", g, e, gpu, err)
		}
	} else {
		t.Log("d.Open failed, cannot test multiple calls on open driver")
	}
}

func TestName(t *testing.T) {
	// Name should not require an open driver.
	d := &Driver{}
	s := d.Name()
	if s == "" {
		t.Error("d.Name()\nhave \"\"\nwant non-empty")
	} else if !strings.HasPrefix(s, "vulkan") {
		t.Errorf("d.Name()\nhave %s\nwant vulkan*", s)
	}
	if d.inst != nil || d.dev != nil {
		t.Errorf("d.Name(): Driver\nhave %v\nwant Driver{}", d)
	}
	// Name should not require a valid driver.
	d = nil
	defer func() {
		if x := recover(); x != nil {
			t.Errorf("unexpected panic: %v", x)
		}
	}()
	if x := d.Name(); x != s {
		t.Errorf("d.Name()\nhave %s\nwant %s (differs from previous call)", x, s)
	}
	// Name should not change for open driver.
	d = &Driver{}
	if _, err := d.Open(); err != nil {
		t.Log("d.Open() failed, cannot test Name method with open driver")
	} else if x := d.Name(); x != s {
		t.Errorf("d.Name()\nhave %s\nwant %s (differs from previous call)", x, s)
	}
}

func TestClose(t *testing.T) {
	// Close should not require an open driver.
	d := Driver{}
	d.Close()
	// Close should set d to the zero value.
	if _, err := d.Open(); err != nil {
		t.Log("d.Open() failed, cannot test Close method with open driver")
	} else {
		d.Close()
		if d.inst != nil || d.dev != nil {
			t.Errorf("d.Close(): Driver\nhave %v\nwant Driver{}", d)
		}
	}
}

func TestDriver(t *testing.T) {
	var d *Driver
	if x, ok := d.Driver().(*Driver); !ok || x != nil {
		t.Errorf("d.Driver()\nhave %#v\nwant %#v", x, (*Driver)(nil))
	}
	d = new(Driver)
	if x := d.Driver(); x != d {
		t.Errorf("d.Driver()\nhave %p\nwant %p", x, d)
	}
}

func TestSelectExts(t *testing.T) {
	cases := [...]struct {
		exts, from []string
		want       error
	}{
		{nil, nil, nil},
		{[]string{}, []string{}, nil},
		{nil, []string{}, nil},
		{[]string{}, nil, nil},
		{[]string{extSurfaceS}, []string{extSurfaceS}, nil},
		{[]string{extSurfaceS}, []string{extSurfaceS, extDisplayS}, nil},
		{[]string{extDisplayS, extSurfaceS}, []string{extSurfaceS, extDisplayS}, nil},
		{[]string{extDisplayS}, []string{extSurfaceS, extDisplayS}, nil},
		{[]string{extSurfaceS}, nil, errNoExtension},
		{[]string{extDisplayS}, []string{extSurfaceS}, errNoExtension},
		{[]string{extSurfaceS, extDisplayS}, []string{extSurfaceS}, errNoExtension},
		{[]string{extSurfaceS, extDisplayS}, []string{}, errNoExtension},
	}
	for _, c := range cases {
		a, f, e := selectExts(c.exts, c.from)
		if e != c.want {
			t.Errorf("selectExts()\nhave _, _, %v\nwant %v", e, c.want)
		}
		if e == nil {
			// The array should be valid only when exts is not nil/empty.
			if a == nil && len(c.exts) > 0 {
				t.Error("selectExts()\nhave nil, _, _\nwant non-nil")
			} else if err := checkCStrings(c.exts, unsafe.Pointer(a)); err != nil {
				t.Error(err)
			}
			if f == nil {
				t.Error("selectExts()\nhave _, nil, _\nwant non-nil")
			} else {
				// It should be safe to call the closure even for nil/empty exts.
				f()
			}
		} else {
			if a != nil {
				t.Error("selectExts()\nhave non-nil, _, _\nwant nil")
			}
			if f != nil {
				t.Error("selectExts()\nhave _, non-nil, _\nwant nil")
			}
		}
	}
}

func TestMemSanity(t *testing.T) {
	d := Driver{}
	if _, err := d.Open(); err != nil {
		t.Error("d.Open() failed, cannot test memory sanity")
		return
	}
	defer d.Close()
	if len(d.mused) != int(d.mprop.memoryHeapCount) {
		t.Errorf("len(d.mused)\nhave %d\nwant %d", len(d.mused), d.mprop.memoryHeapCount)
	}
	for i, n := range d.mused {
		if n != 0 {
			t.Errorf("d.mused[%d]\nhave %d\nwant 0", i, n)
		}
	}
}

func TestExtSanity(t *testing.T) {
	d := Driver{}
	if _, err := d.Open(); err != nil {
		t.Error("d.Open() failed, cannot test extension sanity")
		return
	}
	defer d.Close()

	if !d.exts[extSurface] {
		if d.exts[extDisplay] {
			t.Error("d.exts[extDisplay]\nhave true\nwant false")
		}
		if d.exts[extAndroidSurface] {
			t.Error("d.exts[extAndroidSurface]\nhave true\nwant false")
		}
		if d.exts[extWaylandSurface] {
			t.Error("d.exts[extWaylandSurface]\nhave true\nwant false")
		}
		if d.exts[extWin32Surface] {
			t.Error("d.exts[extWin32Surface]\nhave true\nwant false")
		}
		if d.exts[extXCBSurface] {
			t.Error("d.exts[extXCBSurface]\nhave true\nwant false")
		}
		if d.exts[extSwapchain] {
			t.Error("d.exts[extSwapchain]\nhave true\nwant false")
		}
	}

	if d.exts[extDisplay] {
		if !d.exts[extSwapchain] && d.exts[extDisplaySwapchain] {
			t.Error("d.exts[extDisplaySwapchain]\nhave true\nwant false")
		}
	} else if d.exts[extDisplaySwapchain] {
		t.Error("d.exts[extDisplaySwapchain]\nhave true\nwant false")
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
		if d.exts[badi[i]] {
			t.Errorf("d.exts[<%s>]\nhave true\nwant false", bads[i])
		}
	}
}
