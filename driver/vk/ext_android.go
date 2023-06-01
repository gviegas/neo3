// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

func platformInstanceExts() extInfo {
	return extInfo{
		optional:  []int{extSurface, extAndroidSurface},
		optionalS: []string{extSurfaceS, extAndroidSurfaceS},
	}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && d.exts[extAndroidSurface] {
		return extInfo{
			optional:  []int{extSwapchain},
			optionalS: []string{extSwapchainS},
		}
	}
	return extInfo{}
}
