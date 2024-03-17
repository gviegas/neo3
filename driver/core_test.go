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

func TestCmdBuffer(t *testing.T) {
	cb, err := gpu.NewCmdBuffer()
	if err != nil {
		t.Errorf("GPU.NewCmdBuffer failed: %#v", err)
		return
	}
	defer cb.Destroy()
	for range 2 {
		if cb.IsRecording() {
			t.Error("CmdBuffer.isRecording:\nhave true\nwant false")
		}
		err := cb.Begin()
		if err != nil {
			t.Errorf("CmdBuffer.Begin failed: %#v", err)
			return
		}
		if !cb.IsRecording() {
			t.Error("CmdBuffer.isRecording:\nhave false\nwant true")
		}
		err = cb.End()
		if err != nil {
			t.Errorf("CmdBuffer.End failed: %#v", err)
			return
		}
		err = cb.Reset()
		if err != nil {
			t.Errorf("CmdBuffer.Reset failed: %#v", err)
			return
		}
	}
}
