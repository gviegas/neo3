// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package ctxt

import (
	"testing"
)

func TestInit(t *testing.T) {
	// If we didn't panic during initialization,
	// then drv and gpu must have been set,
	// limits must contain gpu.Limits() and
	// features must contain gpu.Features().
	if drv == nil {
		t.Error("unexpected nil drv")
	}
	if gpu == nil {
		t.Error("unexpected nil gpu")
	} else {
		if limits != gpu.Limits() {
			t.Error("unexpected limits value")
		}
		if features != gpu.Features() {
			t.Error("unexpected features value")
		}
	}
}
