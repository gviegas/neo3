// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !android

package wsi

import (
	"os"
)

func init() {
	// TODO: Prefer Xorg for now since Wayland has no window decorations.
	_, useWL := os.LookupEnv("DO_USE_WAYLAND")
	if useWL && os.Getenv("WAYLAND_DISPLAY") != "" {
		if err := initWayland(); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		} else {
			return
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if err := initXCB(); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		} else {
			return
		}
	}
	initDummy()
}
