// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !linux && !windows

package vk

func platformInstanceExts() extInfo {
	return extInfo{
		optional:  []int{extSurface, extXCBSurface},
		optionalS: []string{extSurfaceS, extXCBSurfaceS},
	}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && d.exts[extXCBSurface] {
		return extInfo{
			optional:  []int{extSwapchain},
			optionalS: []string{extSwapchainS},
		}
	}
	return extInfo{}
}
