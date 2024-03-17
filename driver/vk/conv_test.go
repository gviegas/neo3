// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"testing"

	"gviegas/neo3/driver"
)

func TestPixelFmt(t *testing.T) {
	pfs := [...]driver.PixelFmt{
		driver.FInvalid,
		driver.RGBA8un,
		driver.RGBA8n,
		driver.RGBA8ui,
		driver.RGBA8i,
		driver.RGBA8sRGB,
		driver.BGRA8un,
		driver.BGRA8sRGB,
		driver.RG8un,
		driver.RG8n,
		driver.RG8ui,
		driver.RG8i,
		driver.R8un,
		driver.R8n,
		driver.R8ui,
		driver.R8i,
		driver.RGBA16f,
		driver.RGBA16ui,
		driver.RGBA16i,
		driver.RG16f,
		driver.RG16ui,
		driver.RG16i,
		driver.R16f,
		driver.R16ui,
		driver.R16i,
		driver.RGBA32f,
		driver.RGBA32ui,
		driver.RGBA32i,
		driver.RG32f,
		driver.RG32ui,
		driver.RG32i,
		driver.R32f,
		driver.R32ui,
		driver.R32i,
		driver.D16un,
		driver.D32f,
		driver.S8ui,
		driver.D24unS8ui,
		driver.D32fS8ui,
	}
	for _, f := range pfs {
		if x := convPixelFmt(f); x < 0 || f.IsInternal() {
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
