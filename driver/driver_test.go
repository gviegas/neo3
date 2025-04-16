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

func TestDriverOpenClose(t *testing.T) {
	name := drv.Name()
	if name == "" {
		t.Error("Driver.Name: name is empty")
	}
	drv.Close()
	if nm := drv.Name(); nm != name {
		t.Errorf("Driver.Name: unexpected name after call to Close\nhave %v\nwant %v", nm, name)
	}
	var err error
	gpu, err = drv.Open()
	if err != nil {
		t.Fatalf("Failed to re-Open drv: %v", err)
	}
	if nm := drv.Name(); nm != name {
		t.Errorf("Driver.Name: unexpected name after call to Open\nhave %v\nwant %v", nm, name)
	}
	g, err := drv.Open()
	if err != nil {
		t.Errorf("Driver.Open: unexpected non-nil error: %v", err)
	}
	if g != gpu {
		t.Fatal("Driver.Open: unexpected GPU value")
	}
}
