// Copyright 2022 Gustavo C. Viegas. All rights reserved.

//go:build !darwin && !linux && !windows

package wsi

import (
	"os"
)

func init() {
	if os.Getenv("DISPLAY") != "" {
		if err := initXCB(); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		} else {
			return
		}
	}
	initDummy()
}
