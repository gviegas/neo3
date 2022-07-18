// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <windows.h>
// #include <proc.h>
import "C"

import (
	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/wsi"
)

func (s *swapchain) initSurface() error {
	if wsi.PlatformInUse() == wsi.Win32 {
		return s.initWin32Surface()
	}
	return driver.ErrCannotPresent
}

func (s *swapchain) initWin32Surface() error {
	if !s.d.exts[extWin32Surface] {
		return driver.ErrCannotPresent
	}
	info := C.VkWin32SurfaceCreateInfoKHR{
		sType:     C.VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR,
		hinstance: C.HINSTANCE(wsi.HinstWin32()),
		hwnd:      C.HWND(wsi.HwndWin32(s.win)),
	}
	var sf C.VkSurfaceKHR
	err := checkResult(C.vkCreateWin32SurfaceKHR(s.d.inst, &info, nil, &sf))
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
