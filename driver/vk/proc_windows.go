// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <windows.h>
// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"unsafe"

	"github.com/gviegas/scene/driver"
)

// BUG: Not tested.

// proc is responsible for loading and unloading the Vulkan library.
type proc struct {
	h unsafe.Pointer
}

// open loads the Vulkan library and fetches vkGetInstanceProcAddr.
func (p *proc) open() error {
	lib := C.CString("vulkan-1.dll")
	defer C.free(unsafe.Pointer(lib))
	h := C.LoadLibrary(lib)
	if h == nil {
		return driver.ErrNotInstalled
	}
	sym := C.CString("vkGetInstanceProcAddr")
	defer C.free(unsafe.Pointer(sym))
	f := C.GetProcAddress(h, sym)
	if f == nil {
		C.FreeLibrary(h)
		return driver.ErrNotInstalled
	}
	p.h = h
	C.getInstanceProcAddr = C.PFN_getInstanceProcAddr(f)
	return nil
}

// close unloads the Vulkan library and invalidates all symbols.
func (p *proc) close() {
	if p.h != nil {
		C.FreeLibrary(p.h)
	}
	C.getInstanceProcAddr = nil
	*p = proc{}
}
