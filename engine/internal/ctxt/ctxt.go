// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package ctxt provides the GPU driver used in the engine.
package ctxt

import (
	"errors"
	"strings"

	"gviegas/neo3/driver"
)

var (
	drv      driver.Driver
	gpu      driver.GPU
	limits   driver.Limits
	features driver.Features
)

var errNoDriver = errors.New("ctxt: driver not found")

// loadDriver attempts to load any driver whose name
// contains the name string. It is case insensitive.
// If name is the empty string, then all registered
// drivers are considered.
// It assumes that the drv and gpu vars hold invalid
// values and replaces both on success.
// The limits and features vars are queried from the
// new gpu.
func loadDriver(name string) error {
	drivers := driver.Drivers()
	err := errNoDriver
	name = strings.ToLower(name)
	for i := range drivers {
		if !strings.Contains(strings.ToLower(drivers[i].Name()), name) {
			continue
		}
		var u driver.GPU
		if u, err = drivers[i].Open(); err != nil {
			continue
		}
		drv = drivers[i]
		gpu = u
		limits = gpu.Limits()
		features = gpu.Features()
		return nil
	}
	return err
}

// Driver returns the driver.Driver.
func Driver() driver.Driver { return drv }

// GPU returns the driver.GPU.
func GPU() driver.GPU { return gpu }

// Limits returns GPU().Limits().
// This value is retrieved only once. It must not be
// changed by the caller.
func Limits() *driver.Limits { return &limits }

// Features returns GPU().Features().
// This value is retrieved only once. It must not be
// changed by the caller.
func Features() *driver.Features { return &features }
