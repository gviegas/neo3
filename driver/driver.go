// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package driver defines a set of interfaces encompassing
// common GPU functionality.
// It is designed to allow platform-specific APIs to be
// implemented in a mostly straightforward manner.
package driver

import (
	"errors"
)

// Driver is the interface that provides methods for
// loading and unloading an underlying implementation.
type Driver interface {
	// Open initializes the driver.
	// If it succeeds, further calls with the same receiver
	// have no effect and must return the same GPU instance.
	// Callers should assume that Open is not safe for
	// parallel execution.
	Open() (GPU, error)

	// Name returns the name of the driver.
	// It must not cause the driver to be opened.
	Name() string

	// Close deinitializes the driver.
	// Closing a driver that is not open has no effect.
	// Callers should assume that Close is not safe for
	// parallel execution.
	Close()
}

// ErrNotInstalled means that a platform-specific library
// required for the driver to work is not present in the
// system.
var ErrNotInstalled = errors.New("missing required library")

// ErrNoDevice means that no suitable device could be
// found.
var ErrNoDevice = errors.New("no suitable device found")

// ErrNoHostMemory means that host memory could not be
// allocated.
var ErrNoHostMemory = errors.New("out of host memory")

// ErrNoDeviceMemory means that device memory could not
// be allocated.
var ErrNoDeviceMemory = errors.New("out of device memory")

// ErrFatal means that the driver is in an unrecoverable
// state. Upon encountering such an error, the application
// must destroy everything that it created using the
// driver's GPU and then call the Close method. It may call
// Open again to reinitialize the driver for further use.
var ErrFatal = errors.New("fatal error")

// Drivers returns the registered Drivers.
// Client code imports specific driver packages, and then
// call this function from init. As such, drivers that do
// not register themselves on init will not be considered
// for selection.
func Drivers() []Driver {
	lock <- 1
	drv := make([]Driver, len(drivers))
	copy(drv, drivers)
	<-lock
	return drv
}

// Register registers a Driver.
// Driver implementations are expected to call this function
// once, from init.
func Register(drv Driver) {
	lock <- 1
	for i := range drivers {
		if drivers[i].Name() == drv.Name() {
			drivers[i] = drv
			goto registered
		}
	}
	drivers = append(drivers, drv)
registered:
	<-lock
}

// Variables used for driver registration.
var (
	lock    chan int = make(chan int, 1)
	drivers []Driver = make([]Driver, 0, 4)
)
