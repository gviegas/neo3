// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !android

package vk

import (
	"gviegas/neo3/wsi"
)

func platformInstanceExts() extInfo {
	switch wsi.PlatformInUse() {
	case wsi.Wayland:
		return extInfo{
			optional: []extension{extSurface, extWaylandSurface},
		}
	case wsi.XCB:
		return extInfo{
			optional: []extension{extSurface, extXCBSurface},
		}
	}
	return extInfo{}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && (d.exts[extWaylandSurface] || d.exts[extXCBSurface]) {
		return extInfo{
			optional: []extension{extSwapchain},
		}
	}
	return extInfo{}
}
