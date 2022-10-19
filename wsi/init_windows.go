// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"os"
	"runtime"
)

func init() {
	runtime.LockOSThread()
	if err := initWin32(); err != nil {
		runtime.UnlockOSThread()
		os.Stderr.WriteString(err.Error() + "\n")
	} else {
		return
	}
	initDummy()
}
