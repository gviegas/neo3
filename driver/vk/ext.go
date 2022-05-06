// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"unsafe"
)

const (
	// Instance extensions.
	extSurface, extSurfaceS               = iota, "VK_KHR_surface"
	extDisplay, extDisplayS               = iota, "VK_KHR_display"
	extAndroidSurface, extAndroidSurfaceS = iota, "VK_KHR_android_surface"
	extWaylandSurface, extWaylandSurfaceS = iota, "VK_KHR_wayland_surface"
	extWin32Surface, extWin32SurfaceS     = iota, "VK_KHR_win32_surface"
	extXCBSurface, extXCBSurfaceS         = iota, "VK_KHR_xcb_surface"

	// Device extensions.
	extSwapchain, extSwapchainS               = iota, "VK_KHR_swapchain"
	extDisplaySwapchain, extDisplaySwapchainS = iota, "VK_KHR_display_swapchain"

	extN = iota
)

// instanceExts returns a list containing the names of all instance extensions
// advertised by the Vulkan implementation.
func instanceExts() (exts []string, err error) {
	var n C.uint32_t
	if err = checkResult(C.vkEnumerateInstanceExtensionProperties(nil, &n, nil)); err != nil {
		return
	}
	if n == 0 {
		return
	}
	p := (*C.VkExtensionProperties)(C.malloc(C.sizeof_VkExtensionProperties * C.size_t(n)))
	defer C.free(unsafe.Pointer(p))
	if err = checkResult(C.vkEnumerateInstanceExtensionProperties(nil, &n, p)); err != nil {
		return
	}
	props := unsafe.Slice(p, n)
	exts = make([]string, n)
	for i, prop := range props {
		prop.extensionName[len(prop.extensionName)-1] = 0
		exts[i] = C.GoString(&prop.extensionName[0])
	}
	return
}

// deviceExts returns a list containing the names of all device extensions
// advertised by the Vulkan implementation.
func deviceExts(d C.VkPhysicalDevice) (exts []string, err error) {
	if d == nil {
		panic("vk.deviceExts called with nil physical device")
	}
	var n C.uint32_t
	if err = checkResult(C.vkEnumerateDeviceExtensionProperties(d, nil, &n, nil)); err != nil {
		return
	}
	if n == 0 {
		return
	}
	p := (*C.VkExtensionProperties)(C.malloc(C.sizeof_VkExtensionProperties * C.size_t(n)))
	defer C.free(unsafe.Pointer(p))
	if err = checkResult(C.vkEnumerateDeviceExtensionProperties(d, nil, &n, p)); err != nil {
		return
	}
	props := unsafe.Slice(p, n)
	exts = make([]string, n)
	for i, prop := range props {
		prop.extensionName[len(prop.extensionName)-1] = 0
		exts[i] = C.GoString(&prop.extensionName[0])
	}
	return
}

// selectExts creates an array of C strings that matches the contents of exts.
// exts must be a subset of from, otherwise errNoExtension is returned.
// Call the free closure to deallocate the names array and C strings.
func selectExts(exts []string, from []string) (names **C.char, free func(), err error) {
extLoop:
	for _, e := range exts {
		for _, f := range from {
			if e == f {
				continue extLoop
			}
		}
		err = errNoExtension
		return
	}
	names = (**C.char)(C.malloc(C.size_t(unsafe.Sizeof(*names)) * C.size_t(len(exts))))
	s := unsafe.Slice(names, len(exts))
	for i, e := range exts {
		s[i] = C.CString(e)
	}
	free = func() {
		for _, cs := range s {
			C.free(unsafe.Pointer(cs))
		}
		C.free(unsafe.Pointer(names))
	}
	return
}
