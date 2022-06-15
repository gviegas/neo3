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
	if tProcErr = checkProcOpen(); tProcErr != nil {
		d.Close()
		return
	}

	// Global and instance-level procs other than C.getInstanceProcAddr should
	// be valid only after d.initInstance is called.
	if tProcErr = checkProcInstance(); tProcErr == nil {
		tProcErr = errors.New("checkProcInstance(): unexpected nil error")
		d.Close()
		return
	}
	if d.initInstance() != nil {
		// Not a proc error.
		d.Close()
		return
	}
	if tProcErr = checkProcInstance(); tProcErr != nil {
		d.Close()
		return
	}

	// Device-level procs other than C.getDeviceProcAddr should be valid only
	// after d.initDevice is called.
	if tProcErr = checkProcDevice(); tProcErr == nil {
		tProcErr = errors.New("checkProcDevice(): unexpected nil error")
		d.Close()
		return
	}
	if d.initDevice() != nil {
		// Not a proc error.
		d.Close()
		return
	}
	if tProcErr = checkProcDevice(); tProcErr != nil {
		d.Close()
		return
	}

	d.Close()
	tProcErr = checkProcClear()
}

// NOTE: The bulk of this test is done on init to ensure that no other
// calls to Driver's Open/Close happen before the testing.
func TestProc(t *testing.T) {
	if tProcErr != nil {
		t.Fatal(tProcErr)
	}
}
