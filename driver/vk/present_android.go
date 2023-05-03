// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <proc.h>
import "C"

import (
	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

func (s *swapchain) initSurface() error {
	if wsi.PlatformInUse() == wsi.Android {
		return s.initAndroidSurface()
	}
	return driver.ErrCannotPresent
}

// TODO
func (s *swapchain) initAndroidSurface() error {
	if !s.d.exts[extAndroidSurface] {
		return driver.ErrCannotPresent
	}
	panic("not implemented")
}
