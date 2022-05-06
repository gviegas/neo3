// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"testing"
)

func TestInit(t *testing.T) {
	// If we didn't panic during initialization,
	// then drv and gpu must have been set.
	if drv == nil {
		t.Error("unexpected nil drv")
	}
	if gpu == nil {
		t.Error("unexpected nil gpu")
	}
}
