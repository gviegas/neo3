// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package ctx

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
