// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"testing"

	"gviegas/neo3/driver"
)

func TestDrivers(t *testing.T) {
	drivers := driver.Drivers()
	for i := range drivers {
		name := drivers[i].Name()
		for j := range i {
			if name == drivers[j].Name() {
				t.Error("driver.Drivers: Driver.Name is not unique")
			}
		}
	}
	drivers2 := driver.Drivers()
	if len(drivers) != len(drivers2) {
		t.Error("driver.Drivers: length mismatch")
	} else {
		for i := range drivers {
			if drivers[i].Name() != drivers2[i].Name() {
				t.Error("driver.Drivers: Driver.Name mismatch")
			}
		}
	}
}
