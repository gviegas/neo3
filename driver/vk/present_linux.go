// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !android

package vk

// #include <proc.h>
import "C"

import (
	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/wsi"
)

func (s *swapchain) initSurface() error {
	switch wsi.PlatformInUse() {
	case wsi.None:
		return s.initDisplaySurface()
	case wsi.Wayland:
		return s.initWaylandSurface()
	case wsi.XCB:
		return s.initXCBSurface()
	}
	return driver.ErrCannotPresent
}

func (s *swapchain) initWaylandSurface() error {
	if !s.d.exts[extWaylandSurface] {
		return driver.ErrCannotPresent
	}
	info := C.VkWaylandSurfaceCreateInfoKHR{
		sType:   C.VK_STRUCTURE_TYPE_WAYLAND_SURFACE_CREATE_INFO_KHR,
		display: (*C.struct_wl_display)(wsi.DisplayWayland()),
		surface: (*C.struct_wl_surface)(wsi.SurfaceWayland(s.win)),
	}
	var sf C.VkSurfaceKHR
	err := checkResult(C.vkCreateWaylandSurfaceKHR(s.d.inst, &info, nil, &sf))
	if err != nil {
		return err
	}
	qfam, err := s.d.presQueueFor(sf)
	if err != nil {
		C.vkDestroySurfaceKHR(s.d.inst, sf, nil)
		return err
	}
	s.qfam = qfam
	s.sf = sf
	return nil
}
