// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"testing"

	"gviegas/neo3/engine/internal/ctxt"
)

func TestCtxt(t *testing.T) {
	drv := ctxt.Driver()
	if drv == nil {
		t.Fatal("ctxt.Driver: unexpected nil driver.Driver")
	}
	gpu := ctxt.GPU()
	if gpu == nil {
		t.Fatal("ctxt.GPU: unexpected nil driver.GPU")
	}
	if d := ctxt.Driver(); d != drv {
		t.Fatalf("ctxt.Driver: ctxt mismatch\nhave %v\nwant %v", d, drv)
	}
	if u := ctxt.GPU(); u != gpu {
		t.Fatalf("ctxt.GPU: ctxt mismatch\nhave %v\nwant %v", u, gpu)
	}
	if u, err := drv.Open(); u != gpu || err != nil {
		t.Fatalf("drv.Open: ctxt mismatch\nhave %v, %v\nwant %v, nil", u, err, gpu)
	}
}
