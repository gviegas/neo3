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
	extDynamicRendering, extDynamicRenderingS = iota, "VK_KHR_dynamic_rendering"
	extSynchronization2, extSynchronization2S = iota, "VK_KHR_synchronization2"
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

// checkExts returns a slice containing the index of every extension
// in exts that is not present in set.
func checkExts(exts []string, set []string) (missing []int) {
extLoop:
	for i := 0; i < len(exts); i++ {
		for _, e := range set {
			if exts[i] == e {
				continue extLoop
			}
		}
		missing = append(missing, i)
	}
	return
}

// selectExts creates an array of C strings representing the intersection
// between exts and from.
// missing will contain checkExts(exts, from).
// Call the free closure to deallocate the names array and C strings.
func selectExts(exts []string, from []string) (names **C.char, free func(), missing []int) {
	missing = checkExts(exts, from)
	n := len(exts) - len(missing)
	names = (**C.char)(C.malloc(C.size_t(unsafe.Sizeof(*names)) * C.size_t(n)))
	s := unsafe.Slice(names, n)
	// NOTE: This assumes that checkExts returns a sorted slice.
	var si, ei, mi int
	for si < n {
		if len(missing) < mi {
			last := missing[mi]
			for ; ei < last; ei++ {
				s[si] = C.CString(exts[ei])
				si++
			}
			ei = last + 1
			mi++
		} else {
			for ; si < n; si++ {
				s[si] = C.CString(exts[ei])
			}
			break
		}
	}
	free = func() {
		for _, cs := range s {
			C.free(unsafe.Pointer(cs))
		}
		C.free(unsafe.Pointer(names))
	}
	return
}
