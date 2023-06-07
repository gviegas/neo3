// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !android && !windows

package vk

// #include <proc.h>
import "C"

import (
	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

func (s *swapchain) initXCBSurface() error {
	if !s.d.exts[extXCBSurface] {
		return driver.ErrCannotPresent
	}
	info := C.VkXcbSurfaceCreateInfoKHR{
		sType:      C.VK_STRUCTURE_TYPE_XCB_SURFACE_CREATE_INFO_KHR,
		connection: (*C.xcb_connection_t)(wsi.ConnXCB()),
		window:     C.uint32_t(wsi.WindowXCB(s.win)),
	}
	var sf C.VkSurfaceKHR
	err := checkResult(C.vkCreateXcbSurfaceKHR(s.d.inst, &info, nil, &sf))
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
