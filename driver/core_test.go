// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"testing"
	//"gviegas/neo3/driver"
)

func TestGPUDriver(t *testing.T) {
	g, _ := drv.Open()
	if gpu.Driver() != drv || gpu.Driver() != g.Driver() {
		t.Error("GPU.Driver: unexpected Driver value")
	}
}
