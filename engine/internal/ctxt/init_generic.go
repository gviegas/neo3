// Copyright 2023 Gustavo C. Viegas. All rights reserved.

//go:build linux || windows

package ctxt

import (
	_ "gviegas/neo3/driver/vk"
)

func init() {
	if err := loadDriver("vulkan"); err != nil {
		// Try all drivers.
		if err = loadDriver(""); err != nil {
			panic(err)
		}
	}
}
