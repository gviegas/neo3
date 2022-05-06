// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !android

package vk

// #include <proc.h>
import "C"

import (
	//"os"

	"github.com/gviegas/scene/wsi"
)

func (d *Driver) setInstanceExts(info *C.VkInstanceCreateInfo) func() {
	var exts []string
	var inds []int
	var names **C.char
	free := func() {}

	if from, err := instanceExts(); err == nil {
		//if os.Getenv("WAYLAND_DISPLAY") != "" {
		if wsi.PlatformInUse() == wsi.Wayland {
			exts = []string{extSurfaceS, extWaylandSurfaceS}
			if names, free, err = selectExts(exts, from); err == nil {
				inds = []int{extSurface, extWaylandSurface}
				goto valueSet
			}
		}
		//if os.Getenv("DISPLAY") != "" {
		if wsi.PlatformInUse() == wsi.XCB {
			exts = []string{extSurfaceS, extXCBSurfaceS}
			if names, free, err = selectExts(exts, from); err == nil {
				inds = []int{extSurface, extXCBSurface}
				goto valueSet
			}
		}
		exts = []string{extSurfaceS, extDisplayS}
		if names, free, err = selectExts(exts, from); err == nil {
			inds = []int{extSurface, extDisplay}
			goto valueSet
		}
		exts = nil
	}

valueSet:
	for _, e := range inds {
		d.exts[e] = true
	}
	info.enabledExtensionCount = C.uint32_t(len(exts))
	info.ppEnabledExtensionNames = names
	return free
}

func (d *Driver) setDeviceExts(info *C.VkDeviceCreateInfo) func() {
	if d.exts[extSurface] {
		if from, err := deviceExts(d.pdev); err == nil {
			exts := []string{extSwapchainS}
			inds := []int{extSwapchain}
			if d.exts[extDisplay] {
				exts = append(exts, extDisplaySwapchainS)
				inds = append(inds, extDisplaySwapchain)
			}
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
