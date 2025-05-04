// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// TODO: Consider adding a TestMain function here to
// ensure that examples using package wsi run on the
// main thread (doesn't seem necessary currently).

package driver_test

import (
	"log"
	"unsafe"

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

const (
	NFrame   = 3
	Samples  = 4
	DepthFmt = driver.D16Unorm
)

var (
	// Vertex positions (CCW).
	cubePos = [24 * 3]float32{
		-1, -1, +1,
		-1, +1, +1,
		-1, +1, -1,
		-1, -1, -1,

		+1, -1, -1,
		+1, +1, -1,
		+1, +1, +1,
		+1, -1, +1,

		+1, -1, -1,
		+1, -1, +1,
		-1, -1, +1,
		-1, -1, -1,

		-1, +1, -1,
		-1, +1, +1,
		+1, +1, +1,
		+1, +1, -1,

		-1, -1, -1,
		-1, +1, -1,
		+1, +1, -1,
		+1, -1, -1,

		+1, -1, +1,
		+1, +1, +1,
		-1, +1, +1,
		-1, -1, +1,
	}
	// Vertex UVs.
	cubeUV = [24 * 2]float32{
		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,
	}
	// Input assembly indices.
	cubeIdx = [36]uint32{
		0, 1, 2,
		0, 2, 3,
		4, 5, 6,
		4, 6, 7,
		8, 9, 10,
		8, 10, 11,
		12, 13, 14,
		12, 14, 15,
		16, 17, 18,
		16, 18, 19,
		20, 21, 22,
		20, 22, 23,
	}
)

const (
	cubePosSize = int64(unsafe.Sizeof(cubePos))
	cubeUVSize  = int64(unsafe.Sizeof(cubeUV))
	cubeIdxSize = int64(unsafe.Sizeof(cubeIdx))
)

var (
	// Vertex positions (CCW).
	triPos = [9]float32{
		0, -1, 0,
		-1, 1, 0,
		1, 1, 0,
	}
	// Vertex colors.
	triCol = [12]float32{
		0, 1, 1, 1,
		1, 0, 1, 1,
		1, 1, 0, 1,
	}
	// Transform.
	triM = [16]float32{
		0.7, 0, 0, 0,
		0, 0.7, 0, 0,
		0, 0, 0.7, 0,
		0, 0, 0, 1,
	}
)

const (
	triPosSize = int64(unsafe.Sizeof(triPos))
	triColSize = int64(unsafe.Sizeof(triCol))
	triMSize   = int64(unsafe.Sizeof(triM))
)
