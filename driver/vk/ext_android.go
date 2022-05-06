// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <proc.h>
import "C"

func (d *Driver) setInstanceExts(info *C.VkInstanceCreateInfo) func() {
	if from, err := instanceExts(); err == nil {
		exts := []string{extSurfaceS, extAndroidSurfaceS}
		if names, free, err := selectExts(exts, from); err == nil {
			d.exts[extSurface] = true
			d.exts[extAndroidSurface] = true
			info.enabledExtensionCount = C.uint32_t(len(exts))
			info.ppEnabledExtensionNames = names
			return free
		}
	}
	info.enabledExtensionCount = 0
	info.ppEnabledExtensionNames = nil
	return func() {}
}

func (d *Driver) setDeviceExts(info *C.VkDeviceCreateInfo) func() {
	if d.exts[extSurface] && d.exts[extAndroidSurface] {
		if from, err := deviceExts(d.pdev); err == nil {
			exts := []string{extSwapchainS}
			if names, free, err := selectExts(exts, from); err == nil {
				d.exts[extSwapchain] = true
				info.enabledExtensionCount = C.uint32_t(len(exts))
				info.ppEnabledExtensionNames = names
				return free
			}
		}
	}
	info.enabledExtensionCount = 0
	info.ppEnabledExtensionNames = nil
	return func() {}
}
