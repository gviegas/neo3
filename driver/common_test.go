// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"log"

	"gviegas/neo3/driver"
	_ "gviegas/neo3/driver/vk"
)

var (
	drv driver.Driver
	gpu driver.GPU
)

// TODO: Update when other backends are implemented.
func init() {
	// Select a driver to use.
	drivers := driver.Drivers()
drvLoop:
	for i := range drivers {
		switch drivers[i].Name() {
		case "vulkan":
			drv = drivers[i]
			break drvLoop
		}
	}
	if drv == nil {
		log.Fatal("driver.Drivers(): driver not found")
	}
	var err error
	gpu, err = drv.Open()
	if err != nil {
		log.Fatal(err)
	}
	// Ideally, we should call drv.Close somewhere.
}
