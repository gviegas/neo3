// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"testing"

	"gviegas/neo3/driver"
)

func TestPixelFmt(t *testing.T) {
	pfs := [...]driver.PixelFmt{
		driver.FInvalid,
		driver.RGBA8Unorm,
		driver.RGBA8Norm,
		driver.RGBA8Uint,
		driver.RGBA8Int,
		driver.RGBA8SRGB,
		driver.BGRA8Unorm,
		driver.BGRA8SRGB,
		driver.RG8Unorm,
		driver.RG8Norm,
		driver.RG8Uint,
		driver.RG8Int,
		driver.R8Unorm,
		driver.R8Norm,
		driver.R8Uint,
		driver.R8Int,
		driver.RGBA16Float,
		driver.RGBA16Uint,
		driver.RGBA16Int,
		driver.RG16Float,
		driver.RG16Uint,
		driver.RG16Int,
		driver.R16Float,
		driver.R16Uint,
		driver.R16Int,
		driver.RGBA32Float,
		driver.RGBA32Uint,
		driver.RGBA32Int,
		driver.RG32Float,
		driver.RG32Uint,
		driver.RG32Int,
		driver.R32Float,
		driver.R32Uint,
		driver.R32Int,
		driver.D16Unorm,
		driver.D32Float,
		driver.S8Uint,
		driver.D24UnormS8Uint,
		driver.D32FloatS8Uint,
	}
	for _, f := range pfs {
		if x := int32(convPixelFmt(f)); x < 0 || f.IsInternal() {
			t.Fatalf("convPixelFmt(%v):\nhave %v\nwant >= 0", f, x)
		}
	}

	vfs := [...]_Ctype_VkFormat{
		1000066013,
		107,
		125,
		32,
		33,
		51,
		1,
		6,
		7,
		8,
		121,
		122,
		123,
	}
	for _, f := range vfs {
		if x := internalFmt(f); x >= 0 || !x.IsInternal() {
			t.Fatalf("internalFmt(%v):\nhave %v\nwant < 0", f, x)
		} else if y := convPixelFmt(x); y != f {
			t.Fatalf("convPixelFmt(%v):\nhave %v\nwant %v", x, y, f)
		}
	}
}
