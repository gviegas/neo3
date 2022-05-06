// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !linux && !windows

package vk

// #include <proc.h>
import "C"

// setInstanceExts sets the extension fields of the info structure and
// updates the driver's exts array accordingly.
// It returns a closure that deallocates the C data that was set in info.
func (d *Driver) setInstanceExts(info *C.VkInstanceCreateInfo) func() {
	if from, err := instanceExts(); err == nil {
		exts := []string{extSurfaceS, extDisplayS}
		if names, free, err := selectExts(exts, from); err == nil {
			d.exts[extSurface] = true
			d.exts[extDisplay] = true
			info.enabledExtensionCount = C.uint32_t(len(exts))
			info.ppEnabledExtensionNames = names
			return free
		}
	}
	info.enabledExtensionCount = 0
	info.ppEnabledExtensionNames = nil
	return func() {}
}

// setDeviceExts sets the extension fields of the info structure and
// updates the driver's exts array accordingly.
// d.pdev must contain a valid physical device.
// It returns a closure that deallocates the C data that was set in info.
func (d *Driver) setDeviceExts(info *C.VkDeviceCreateInfo) func() {
	if d.exts[extSurface] && d.exts[extDisplay] {
		if from, err := deviceExts(d.pdev); err == nil {
			exts := []string{extSwapchainS, extDisplaySwapchainS}
			inds := [2]int{extSwapchain, extDisplaySwapchain}
			for len(exts) > 0 {
				if names, free, err := selectExts(exts, from); err == nil {
					for i := range exts {
						d.exts[inds[i]] = true
					}
					info.enabledExtensionCount = C.uint32_t(len(exts))
					info.ppEnabledExtensionNames = names
					return free
				}
				exts = exts[:len(exts)-1]
			}
		}
	}
	info.enabledExtensionCount = 0
	info.ppEnabledExtensionNames = nil
	return func() {}
}
