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
			optional:  []int{extSurface, extWaylandSurface},
			optionalS: []string{extSurfaceS, extWaylandSurfaceS},
		}
	case wsi.XCB:
		return extInfo{
			optional:  []int{extSurface, extXCBSurface},
			optionalS: []string{extSurfaceS, extXCBSurfaceS},
		}
	}
	return extInfo{}
}

func platformDeviceExts(d *Driver) extInfo {
	if d.exts[extSurface] && (d.exts[extWaylandSurface] || d.exts[extXCBSurface]) {
		return extInfo{
			optional:  []int{extSwapchain},
			optionalS: []string{extSwapchainS},
		}
	}
	return extInfo{}
}
