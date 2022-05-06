// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"errors"
	"log"
	"os"
	"testing"
)

// Helpers for testing.

// tDrv is the driver managed by TestMain.
var tDrv = Driver{}

// TestMain runs the tests between calls to tDrv.Open and tDrv.Close.
func TestMain(m *testing.M) {
	if _, err := tDrv.Open(); err != nil {
		log.Fatalf("fatal: Driver.Open failed: %v", err)
	}
	name := tDrv.DeviceName()
	imaj, imin, ipat := tDrv.InstanceVersion()
	dmaj, dmin, dpat := tDrv.DeviceVersion()
	log.Printf("\n\tUsing %s\n\tVersion %d.%d.%d (inst), %d.%d.%d (dev)", name, imaj, imin, ipat, dmaj, dmin, dpat)
	c := m.Run()
	tDrv.Close()
	os.Exit(c)
}

// isError checks multiple errors for equality.
func isError(e error, targets ...error) bool {
	for _, x := range targets {
		if errors.Is(e, x) {
			return true
		}
	}
	return false
}
