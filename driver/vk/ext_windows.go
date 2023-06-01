// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

func platformInstanceExts() extInfo {
	return extInfo{
		optional:  []int{extSurface, extWin32Surface},
		optionalS: []string{extSurfaceS, extWin32SurfaceS},
	}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && d.exts[extWin32Surface] {
		return extInfo{
			optional:  []int{extSwapchain},
			optionalS: []string{extSwapchainS},
		}
	}
	return extInfo{}
}
