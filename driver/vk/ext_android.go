// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

func platformInstanceExts() extInfo {
	return extInfo{
		optional: []extension{extSurface, extAndroidSurface},
	}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && d.exts[extAndroidSurface] {
		return extInfo{
			optional: []extension{extSwapchain},
		}
	}
	return extInfo{}
}
