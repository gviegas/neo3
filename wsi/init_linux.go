// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !android

package wsi

import (
	"os"
)

func init() {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		// TODO
		os.Stderr.WriteString("note: wsi doesn't support wayland yet\n")
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
