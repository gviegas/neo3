// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"errors"
	"testing"
)

var tProcErr error

func init() {
	// C.getInstanceProcAddr should be valid only after proc.open is called.
	d := Driver{}
	if tProcErr = checkProcOpen(); tProcErr == nil {
		tProcErr = errors.New("checkProcOpen(): unexpected nil error")
		return
	}
	if tProcErr = d.open(); tProcErr != nil {
		return
	}
	defer d.Close()
	if tProcErr = checkProcOpen(); tProcErr != nil {
		return
	}

	// Global and instance-level procs other than C.getInstanceProcAddr should
	// be valid only after d.initInstance is called.
	if tProcErr = checkProcInstance(); tProcErr == nil {
		tProcErr = errors.New("checkProcInstance(): unexpected nil error")
		return
	}
	if d.initInstance() != nil {
		// Not a proc error.
		return
	}
	if tProcErr = checkProcInstance(); tProcErr != nil {
		return
	}

	// Device-level procs other than C.getDeviceProcAddr should be valid only
	// after d.initDevice is called.
	if tProcErr = checkProcDevice(); tProcErr == nil {
		tProcErr = errors.New("checkProcDevice(): unexpected nil error")
		return
	}
	if d.initDevice() != nil {
		// Not a proc error.
		return
	}
	tProcErr = checkProcDevice()
}

// NOTE: The bulk of this test is done on init because proc's function pointers
// are defined as C global variables, which are never cleared.
func TestProc(t *testing.T) {
	if tProcErr != nil {
		t.Fatal(tProcErr)
	}
}
