// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !linux && !windows

package vk

import (
	"gviegas/neo3/driver"
	"gviegas/neo3/wsi"
)

// initSurface creates a new surface from s.win.
// s.d and s.win must have been set to valid values.
// It sets the qfam and sf fields of s.
func (s *swapchain) initSurface() error {
	if wsi.PlatformInUse() == wsi.XCB {
		return s.initXCBSurface()
	}
	return driver.ErrCannotPresent
}
