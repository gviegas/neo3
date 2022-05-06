// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package wsi

import (
	"errors"
)

var errMissing = errors.New("no wsi implementation")

func initDummy() {
	newWindow = newWindowDummy
	dispatch = dispatchDummy
	setAppName = setAppNameDummy
	platform = None
}

func newWindowDummy(int, int, string) (Window, error) {
	return nil, errMissing
}

func dispatchDummy()         {}
func setAppNameDummy(string) {}
