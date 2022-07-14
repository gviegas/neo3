// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"os"
)

func init() {
	if err := initWin32(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
	} else {
		return
	}
	initDummy()
}
