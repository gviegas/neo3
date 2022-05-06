// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"
	"strings"

	"github.com/gviegas/scene/driver"
)

var (
	drv driver.Driver
	gpu driver.GPU
)

var errNoDriver = errors.New("driver not found")

// loadDriver attempts to load any driver whose name contains
// the provided name string. It is case-sensitive.
// If name is the empty string, all drivers are considered.
// It assumes that the drv and gpu vars hold invalid values
// and replaces both on success.
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
		return nil
	}
	return err
}
