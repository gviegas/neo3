// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

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

// TODO
func (s *swapchain) initWin32Surface() error {
	if !s.d.exts[extWin32Surface] {
		return driver.ErrCannotPresent
	}
	panic("not implemented")
}
