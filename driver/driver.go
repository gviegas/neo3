// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package driver defines a set of interfaces encompassing
// common GPU functionality.
// It is designed to allow platform-specific APIs to be
// implemented in a mostly straightforward manner.
package driver

import (
	"errors"
	"log"
	"sync"
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
var ErrNotInstalled = errors.New("driver: missing required library")

// ErrNoDevice means that no suitable device could be
// found.
var ErrNoDevice = errors.New("driver: no suitable device found")

// ErrNoHostMemory means that host memory could not be
// allocated.
var ErrNoHostMemory = errors.New("driver: out of host memory")

// ErrNoDeviceMemory means that device memory could not
// be allocated.
var ErrNoDeviceMemory = errors.New("driver: out of device memory")

// ErrFatal means that the driver is in an unrecoverable
// state. Upon encountering such an error, the application
// must destroy everything that it created using the
// driver's GPU and then call the Close method. It may call
// Open again to reinitialize the driver for further use.
var ErrFatal = errors.New("driver: fatal error")

// Drivers returns the registered Drivers.
// Client code imports specific driver packages, and then
// call this function from init. As such, drivers that do
// not register themselves on init will not be considered
// for selection.
func Drivers() []Driver {
	mu.Lock()
	defer mu.Unlock()
	drv := make([]Driver, len(drivers))
	copy(drv, drivers)
	return drv
}

// Register registers a Driver.
// Driver implementations are expected to call Register
// exactly once, from an init function.
// If a driver with the same name has already been
// registered, it will be replaced by drv.
func Register(drv Driver) {
	mu.Lock()
	defer mu.Unlock()
	for i := range drivers {
		if drivers[i].Name() == drv.Name() {
			drivers[i] = drv
			log.Printf("[!] driver '%s' replaced", drv.Name())
			return
		}
	}
	drivers = append(drivers, drv)
	log.Printf("driver '%s' registered", drv.Name())
}

// Variables used for driver registration.
var (
	// NOTE: Currently, this mutex is unnecessary.
	mu      sync.Mutex
	drivers []Driver = make([]Driver, 0, 1)
)
