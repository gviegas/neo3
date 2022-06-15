// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !windows

package vk

// #cgo linux LDFLAGS: -ldl
// #include <dlfcn.h>
// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gviegas/scene/driver"
)

// proc is responsible for loading and unloading the Vulkan library.
type proc struct {
	h unsafe.Pointer
}

// open loads the Vulkan library and fetches vkGetInstanceProcAddr.
func (p *proc) open() error {
	var lib *C.char
	switch runtime.GOOS {
	default:
		panic("unsupported OS: " + runtime.GOOS)
	case "android":
		lib = C.CString("libvulkan.so")
	case "linux":
		lib = C.CString("libvulkan.so.1")
	}
	defer C.free(unsafe.Pointer(lib))
	h := C.dlopen(lib, C.RTLD_LAZY|C.RTLD_GLOBAL)
	if h == nil {
		return driver.ErrNotInstalled
	}
	sym := C.CString("vkGetInstanceProcAddr")
	defer C.free(unsafe.Pointer(sym))
	f := C.dlsym(h, sym)
	if f == nil {
		C.dlclose(h)
		return driver.ErrNotInstalled
	}
	p.h = h
	C.getInstanceProcAddr = C.PFN_vkGetInstanceProcAddr(f)
	return nil
}

// close unloads the Vulkan library and invalidates all symbols.
func (p *proc) close() {
	if p.h != nil {
		C.dlclose(p.h)
	}
	C.getInstanceProcAddr = nil
	*p = proc{}
}
