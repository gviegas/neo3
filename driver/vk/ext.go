// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"unsafe"
)

// extension identifies a Vulkan extension.
type extension int

const (
	// Instance extensions.
	extGetPhysicalDeviceProperties2 extension = iota
	extSurface
	extAndroidSurface
	extWaylandSurface
	extWin32Surface
	extXCBSurface

	// Device extensions.
	extMultiview
	extMaintenance2
	extCreateRenderPass2
	extDepthStencilResolve
	extDynamicRendering
	extSynchronization2
	extSwapchain

	extN int = iota
)

// name returns the extension name as a Go string.
func (e extension) name() string {
	switch e {
	case extGetPhysicalDeviceProperties2:
		return "VK_KHR_get_physical_device_properties2"
	case extSurface:
		return "VK_KHR_surface"
	case extAndroidSurface:
		return "VK_KHR_android_surface"
	case extWaylandSurface:
		return "VK_KHR_wayland_surface"
	case extWin32Surface:
		return "VK_KHR_win32_surface"
	case extXCBSurface:
		return "VK_KHR_xcb_surface"
	case extMultiview:
		return "VK_KHR_multiview"
	case extMaintenance2:
		return "VK_KHR_maintenance2"
	case extCreateRenderPass2:
		return "VK_KHR_create_renderpass2"
	case extDepthStencilResolve:
		return "VK_KHR_depth_stencil_resolve"
	case extDynamicRendering:
		return "VK_KHR_dynamic_rendering"
	case extSynchronization2:
		return "VK_KHR_synchronization2"
	case extSwapchain:
		return "VK_KHR_swapchain"
	}
	panic("you have to update vk.extension.name when adding new extensions")
}

// makeExtNames returns a new slice containing the name of every extension
// present in exts.
// Order is preserved.
func makeExtNames(exts []extension) []string {
	s := make([]string, 0, len(exts))
	for _, e := range exts {
		s = append(s, e.name())
	}
	return s
}

// instanceExts returns a list containing the names of all instance extensions
// advertised by the Vulkan implementation.
func instanceExts() (exts []string, err error) {
	if C.enumerateInstanceExtensionProperties == nil {
		panic("vk.instanceExts called with invalid global procedures")
	}
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
		panic("vk.deviceExts called with invalid physical device")
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
// Indices in missing are sorted in increasing order.
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
// Indices in missing indicate which exts's elements weren't selected.
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
				ei++
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

// extInfo describes required and optional extensions.
type extInfo struct {
	required, optional []extension
}

// requiredNames is equivalent to makeExtNames(i.required).
func (i *extInfo) requiredNames() []string { return makeExtNames(i.required) }

// optionalNames is equivalent to makeExtNames(i.optional).
func (i *extInfo) optionalNames() []string { return makeExtNames(i.optional) }

// These are platform-independent.
var (
	globalInstanceExts = extInfo{
		required: []extension{extGetPhysicalDeviceProperties2},
	}
	globalDeviceExts = extInfo{
		required: []extension{
			extMultiview,
			extMaintenance2,
			extCreateRenderPass2,
			extDepthStencilResolve,
			extDynamicRendering,
			extSynchronization2,
		},
	}
)

// setInstanceExts sets the enableExtensionCount/ppEnabledExtensionNames
// fields of info. It also updates d.ext to reflect the selected extensions.
// Call the free closure to deallocate the C array/strings.
func (d *Driver) setInstanceExts(info *C.VkInstanceCreateInfo) (free func(), err error) {
	var set []string
	if set, err = instanceExts(); err != nil {
		free = func() {}
		return
	}
	platform := platformInstanceExts()
	return d.setExts(&globalInstanceExts, &platform, set,
		&info.enabledExtensionCount, &info.ppEnabledExtensionNames)
}

// setDeviceExts sets the enableExtensionCount/ppEnabledExtensionNames
// fields of info. It also updates d.ext to reflect the selected extensions.
// Call the free closure to deallocate the C array/strings.
func (d *Driver) setDeviceExts(info *C.VkDeviceCreateInfo) (free func(), err error) {
	var set []string
	if set, err = deviceExts(d.pdev); err != nil {
		free = func() {}
		return
	}
	platform := platformDeviceExts(d)
	return d.setExts(&globalDeviceExts, &platform, set,
		&info.enabledExtensionCount, &info.ppEnabledExtensionNames)
}

// setExts generalizes the set*Exts methods.
// Do not call it directly - call d.setInstanceExts/d.setDeviceExts instead.
func (d *Driver) setExts(global *extInfo, platform *extInfo, set []string,
	dstCount *C.uint32_t, dstNames ***C.char) (func(), error) {

	exts := append(global.requiredNames(), platform.requiredNames()...)
	if len(checkExts(exts, set)) != 0 {
		// TODO: Consider logging what is missing.
		return func() {}, errNoExtension
	}

	// Let selectExts filter optional extensions.
	off := len(exts)
	exts = append(append(exts, global.optionalNames()...), platform.optionalNames()...)
	names, free, missing := selectExts(exts, set)
	*dstCount = C.uint32_t(len(exts) - len(missing))
	*dstNames = names
	for _, e := range global.required {
		d.exts[e] = true
	}
	for _, e := range platform.required {
		d.exts[e] = true
	}

	// We known for sure that required extensions are supported,
	// so any missing extension has to be optional.
	opt := append(append([]extension{}, global.optional...), platform.optional...)
	for i := range opt {
		if len(missing) == 0 {
			for _, e := range opt[i:] {
				d.exts[e] = true
			}
			break
		}
		if i == missing[0]-off {
			// TODO: Consider logging what is missing.
			missing = missing[1:]
		} else {
			d.exts[opt[i]] = true
		}
	}
	return free, nil
}
