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

// TODO
func (s *swapchain) initWaylandSurface() error {
	if !s.d.exts[extWaylandSurface] {
		return driver.ErrCannotPresent
	}
	panic("not implemented")
}
