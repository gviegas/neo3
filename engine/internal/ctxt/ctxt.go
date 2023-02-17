// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package ctxt provides the GPU driver used in the engine.
package ctxt

import (
	"errors"
	"strings"

	"github.com/gviegas/scene/driver"
)

var (
	drv    driver.Driver
	gpu    driver.GPU
	limits driver.Limits
)

var errNoDriver = errors.New("ctxt: driver not found")

// loadDriver attempts to load any driver whose name contains
// the provided name string. It is case-sensitive.
// If name is the empty string, all drivers are considered.
// It assumes that the drv and gpu vars hold invalid values
// and replaces both on success. It also updates limits with
// a call to gpu.Limits().
func loadDriver(name string) error {
	drivers := driver.Drivers()
	err := errNoDriver
	for i := range drivers {
		if !strings.Contains(drivers[i].Name(), name) {
			continue
		}
		var u driver.GPU
		if u, err = drivers[i].Open(); err != nil {
			continue
		}
		drv = drivers[i]
		gpu = u
		limits = gpu.Limits()
		return nil
	}
	return err
}

// Driver returns the driver.Driver.
func Driver() driver.Driver { return drv }

// GPU returns the driver.GPU.
func GPU() driver.GPU { return gpu }

// Limits returns driver.Limits of the context's GPU.
// This value is retrieved only once. It must not be
// changed by the caller.
func Limits() *driver.Limits { return &limits }
