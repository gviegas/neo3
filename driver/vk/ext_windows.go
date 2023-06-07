// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

func platformInstanceExts() extInfo {
	return extInfo{
		optional: []extension{extSurface, extWin32Surface},
	}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && d.exts[extWin32Surface] {
		return extInfo{
			optional: []extension{extSwapchain},
		}
	}
	return extInfo{}
}
