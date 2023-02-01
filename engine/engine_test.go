// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"testing"

	"github.com/gviegas/scene/engine/internal/ctx"
)

func TestCtx(t *testing.T) {
	drv := ctx.Driver()
	if drv == nil {
		t.Fatal("ctx.Driver: unexpected nil driver.Driver")
	}
	gpu := ctx.GPU()
	if gpu == nil {
		t.Fatal("ctx.GPU: unexpected nil driver.GPU")
	}
	if d := ctx.Driver(); d != drv {
		t.Fatalf("ctx.Driver: ctx mismatch\nhave %v\nwant %v", d, drv)
	}
	if u := ctx.GPU(); u != gpu {
		t.Fatalf("ctx.GPU: ctx mismatch\nhave %v\nwant %v", u, gpu)
	}
	if u, err := drv.Open(); u != gpu || err != nil {
		t.Fatalf("drv.Open: ctx mismatch\nhave %v, %v\nwant %v, nil", u, err, gpu)
	}
}
