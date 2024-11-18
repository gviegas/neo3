// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"strings"
	"testing"
	"time"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
)

// check checks that tex is valid.
func (tex *Texture) check(t *testing.T) {
	if len(tex.views) < 1 {
		t.Fatal("Texture.views: unexpected len < 1")
	}
	img := tex.views[0].Image()
	for i := 1; i < len(tex.views); i++ {
		// Should be comparable in any case.
		if x := tex.views[i].Image(); x != img {
			t.Fatalf("Texture.views[%d].Image: differs from [0]\nhave %v\nwant %v", i, x, img)
		}
	}
	usg := ^(driver.UCopySrc | driver.UCopyDst | driver.UShaderRead | driver.UShaderWrite | driver.UShaderSample | driver.URenderTarget)
	if tex.usage == 0 || tex.usage&usg != 0 {
		t.Fatalf("Texture.usage: unexpected flag(s) set:\n0x%x", tex.usage&usg)
	}
}

func Test2D(t *testing.T) {
	tex, err := New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), texPrefix) {
			t.Fatalf("New2D: unexpected error:\n%#v", err)
		}
	}
	tex.check(t)

	// param must not be nil.
	_, err = NewTarget(nil)
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Depth must be 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  1,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Layers must be greater than 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  0,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Levels must be greater than 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  0,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Samples must be greater than 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 0,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Width must be no greater than the driver-imposed limit.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1 + ctxt.Limits().MaxImage2D,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Height must be no greater than the driver-imposed limit.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1 + ctxt.Limits().MaxImage2D,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Layers must be no greater than the driver-imposed limit.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1 + ctxt.Limits().MaxLayers,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Levels must be less than 1 + log₂(maxDim).
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  12,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Samples must be a power of two.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 3,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Either Levels or Samples must be 1.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  2,
		Samples: 4,
	})
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}
}

func TestCube(t *testing.T) {
	tex, err := NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  6,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), texPrefix) {
			t.Fatalf("NewCube: unexpected error:\n%#v", err)
		}
	}
	tex.check(t)

	// param must not be nil.
	_, err = NewTarget(nil)
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Depth must be 0.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  1,
		},
		Layers:  6,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Layers must be greater than 0.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  0,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Levels must be greater than 0.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  6,
		Levels:  0,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Width must be no greater than the driver-imposed limit.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1 + ctxt.Limits().MaxImageCube,
			Height: 1024,
			Depth:  0,
		},
		Layers:  6,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Height must be no greater than the driver-imposed limit.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1 + ctxt.Limits().MaxImageCube,
			Depth:  0,
		},
		Layers:  6,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Width and Height must be equal.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 512,
			Depth:  0,
		},
		Layers:  6,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Layers must be no greater than the driver-imposed limit.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1 + ctxt.Limits().MaxLayers,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Layers must be a multiple of 6.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Levels must be less than 1 + log₂(maxDim).
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  6,
		Levels:  12,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Samples must be 1.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1024,
			Height: 1024,
			Depth:  0,
		},
		Layers:  6,
		Levels:  1,
		Samples: 4,
	})
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}
}

func TestTarget(t *testing.T) {
	tex, err := NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), texPrefix) {
			t.Fatalf("NewTarget: unexpected error:\n%#v", err)
		}
	}
	tex.check(t)

	// param must not be nil.
	_, err = NewTarget(nil)
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Depth must be 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  1,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Layers must be greater than 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  0,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Levels must be greater than 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  0,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Samples must be greater than 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 0,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Width must be no greater than the driver-imposed limit.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1 + ctxt.Limits().MaxRenderSize[0],
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Height must be no greater than the driver-imposed limit.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 1 + ctxt.Limits().MaxRenderSize[1],
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Layers must be no greater than the driver-imposed limit.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1 + ctxt.Limits().MaxRenderLayers,
		Levels:  1,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Levels must be less than 1 + log₂(maxDim).
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  12,
		Samples: 1,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Samples must be a power of two.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  1,
		Samples: 3,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Either Levels or Samples must be 1.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D: driver.Dim3D{
			Width:  1280,
			Height: 720,
			Depth:  0,
		},
		Layers:  1,
		Levels:  2,
		Samples: 4,
	})
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}
}

// check checks that s is valid.
func (s *Sampler) check(t *testing.T) {
	if s.sampler == nil {
		t.Fatal("Sampler.sampler: unexpected nil value")
	}
}

func TestSampler(t *testing.T) {
	s, err := NewSampler(&SplrParam{
		Min:      driver.FNearest,
		Mag:      driver.FNearest,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		DoCmp:    false,
		Cmp:      driver.CNever,
		MinLOD:   0,
		MaxLOD:   0.25,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), texPrefix) {
			t.Fatalf("NewSampler: unexpected error:\n%#v", err)
		}
	}
	s.check(t)

	s, err = NewSampler(&SplrParam{
		Min:      driver.FNearest,
		Mag:      driver.FNearest,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		DoCmp:    true,
		Cmp:      driver.CLess,
		MinLOD:   0,
		MaxLOD:   0,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), texPrefix) {
			t.Fatalf("NewSampler: unexpected error:\n%#v", err)
		}
	}
	s.check(t)

	// param must not be nil.
	_, err = NewSampler(nil)
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewSampler: unexpected error:\n%#v", err)
	}

	// MaxAniso must be greater than or equal to 1.0.
	_, err = NewSampler(&SplrParam{
		Min:      driver.FNearest,
		Mag:      driver.FNearest,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 0,
		MinLOD:   0,
		MaxLOD:   0.25,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewSampler: unexpected error:\n%#v", err)
	}

	// MinLOD must be greater than or equal to 0.0.
	_, err = NewSampler(&SplrParam{
		Min:      driver.FNearest,
		Mag:      driver.FNearest,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		MinLOD:   -1,
		MaxLOD:   0.25,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewSampler: unexpected error:\n%#v", err)
	}

	// MaxLOD must be greater than or equal to 0.0.
	_, err = NewSampler(&SplrParam{
		Min:      driver.FNearest,
		Mag:      driver.FNearest,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		MinLOD:   0,
		MaxLOD:   -1,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewSampler: unexpected error:\n%#v", err)
	}

	// MinLOD must be no greater than MaxLOD.
	_, err = NewSampler(&SplrParam{
		Min:      driver.FNearest,
		Mag:      driver.FNearest,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		DoCmp:    true,
		Cmp:      driver.CAlways,
		MinLOD:   1,
		MaxLOD:   0.25,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), texPrefix):
		t.Fatalf("NewSampler: unexpected error:\n%#v", err)
	}
}

func TestTextureFree(t *testing.T) {
	texs := make([]*Texture, 0, 4)
	for i, x := range [4]TexParam{
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D: driver.Dim3D{
				Width:  1024,
				Height: 1024,
				Depth:  0,
			},
			Layers:  1,
			Levels:  1,
			Samples: 1,
		},
		{
			PixelFmt: driver.RGBA8SRGB,
			Dim3D: driver.Dim3D{
				Width:  512,
				Height: 512,
				Depth:  0,
			},
			Layers:  3,
			Levels:  10,
			Samples: 1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D: driver.Dim3D{
				Width:  1024,
				Height: 1024,
				Depth:  0,
			},
			Layers:  6,
			Levels:  1,
			Samples: 1,
		},
		{
			PixelFmt: driver.RGBA16Float,
			Dim3D: driver.Dim3D{
				Width:  1920,
				Height: 1080,
				Depth:  0,
			},
			Layers:  1,
			Levels:  1,
			Samples: 4,
		},
	} {
		var tex *Texture
		var err error
		switch i {
		case 0, 1:
			tex, err = New2D(&x)
			if err != nil {
				t.Fatalf("New2D failed:\n%#v", err)
			}
		case 2:
			tex, err = NewCube(&x)
			if err != nil {
				t.Fatalf("NewCube failed:\n%#v", err)
			}
		default:
			tex, err = NewTarget(&x)
			if err != nil {
				t.Fatalf("NewTarget failed:\n%#v", err)
			}
		}
		texs = append(texs, tex)
	}

	for _, x := range texs {
		x.check(t)
		x.Free()
		if x.views != nil || x.usage != 0 || x.param != (TexParam{}) {
			t.Fatal("Texture.Free: unexpected non-zero value:\n", *x)
		}
	}
}

func TestSamplerFree(t *testing.T) {
	splrs := make([]*Sampler, 0, 3)
	for _, x := range [3]SplrParam{
		{
			Min:      driver.FNearest,
			Mag:      driver.FNearest,
			Mipmap:   driver.FNoMipmap,
			AddrU:    driver.AWrap,
			AddrV:    driver.AWrap,
			AddrW:    driver.AWrap,
			MaxAniso: 1,
			MinLOD:   0,
			MaxLOD:   0.25,
		},
		{
			Min:      driver.FLinear,
			Mag:      driver.FLinear,
			Mipmap:   driver.FNearest,
			AddrU:    driver.AClamp,
			AddrV:    driver.AClamp,
			AddrW:    driver.AClamp,
			MaxAniso: 1,
			DoCmp:    false,
			Cmp:      driver.CLess,
			MinLOD:   0,
			MaxLOD:   0.5,
		},
		{
			Min:      driver.FLinear,
			Mag:      driver.FNearest,
			Mipmap:   driver.FLinear,
			AddrU:    driver.AWrap,
			AddrV:    driver.AClamp,
			AddrW:    driver.AMirror,
			MaxAniso: 1,
			DoCmp:    true,
			Cmp:      driver.CGreater,
			MinLOD:   0.25,
			MaxLOD:   1.0,
		},
	} {
		splr, err := NewSampler(&x)
		if err != nil {
			t.Fatalf("NewSampler failed:\n%#v", err)
		}
		splrs = append(splrs, splr)
	}

	for _, x := range splrs {
		x.check(t)
		x.Free()
		if *x != (Sampler{}) {
			t.Fatal("Sampler.Free: unexpected non-zero value:\n", *x)
		}
	}
}

func TestTexStgBuffer(t *testing.T) {
	var s *texStgBuffer
	var err error

	check := func(nbuf, nbv int) {
		if err != nil {
			t.Fatalf("driver.NewBuffer failed:\n%#v", err)
		}
		if x := int(s.buf.Cap()); x != nbuf {
			t.Fatalf("newTexStg: buf.Cap\nhave %d\nwant %d", x, nbuf)
		}
		if x := s.bv.Len(); x != nbv {
			t.Fatalf("newTexStg: bv.Len\nhave %d\nwant %d", x, nbv)
		}
	}
	checkFree := func() {
		if s.buf != nil {
			t.Fatalf("texStgBuffer.free: buf\nhave %v\nwant nil", s.buf)
		}
		if x := s.bv.Len(); x != 0 {
			t.Fatalf("texStgBuffer.free: bv.Len\nhave %d\nwant 0", x)
		}
	}

	const n = texStgBlock * texStgNBit

	s, err = newTexStg(n)
	check(n, texStgNBit)
	s.free()
	checkFree()

	s, err = newTexStg(n - 1)
	check(n, texStgNBit)
	s.free()
	checkFree()

	s, err = newTexStg(n + 1)
	check(n*2, texStgNBit*2)
	s.free()
	checkFree()

	s, err = newTexStg(1)
	check(n, texStgNBit)
	s.free()
	checkFree()

	s, err = newTexStg(n + n - 1)
	check(n*2, texStgNBit*2)
	s.free()
	checkFree()

	x := 2048 * 2048 * 4
	s, err = newTexStg(x)
	x = (x + n - 1) &^ (n - 1)
	check(x, x/texStgBlock)
	s.free()
	checkFree()
}

func TestTexStgInit(t *testing.T) {
	var s []*texStgBuffer
	for i := 0; i < cap(texStg); i++ {
		select {
		case x := <-texStg:
			if x.wk == nil || x.buf == nil {
				if x.wk != nil || x.buf != nil {
					t.Fatal("texStg: unexpected non-nil wk or buf")
				}
				continue
			}
			if cap(x.wk) != 1 {
				t.Fatalf("texStg: cap(wk)\nhave %d\nwant 1", cap(x.wk))
			}
			wk := <-x.wk
			if len(wk.Work) != 1 {
				t.Fatalf("texStg: len((<-wk).Work)\nhave %d\nwant 1", len(wk.Work))
			}
			if wk.Work[0] == nil {
				t.Fatal("texStg: (<-wk).Work[0]\nhave nil\nwant non-nil")
			}
			if wk.Err != nil {
				t.Fatalf("texStg: (<-wk).Err\nhave %#v\nwant nil", wk.Err)
			}
			if x.buf.Cap() != texStgBlock*texStgNBit {
				t.Fatalf("texStg: buf.Cap:\nhave %d\nwant %d", x.buf.Cap(), texStgBlock*texStgNBit)
			}
			x.wk <- wk
			s = append(s, x)
		default:
		}
	}
	if len(s) != cap(texStg) {
		t.Fatalf("texStg: unexpected len != cap\nhave %d\nwant %d", len(s), cap(texStg))
	}
	for i := range s {
		texStg <- s[i]
	}
}

// checkData checks that a's contents match b's.
// Their lengths may differ.
func checkData[T comparable](a, b []T, t *testing.T) {
	n := len(a)
	if x := len(b); x < n {
		n = x
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			t.Fatalf("checkData: %v != %v (at index %d)", a[i], b[i], i)
		}
	}
}

// TODO: Cube texture; multiple mip levels.
func TestViewCopy(t *testing.T) {
	// One layer.
	for _, param := range [...]TexParam{
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 256, Height: 256},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 512, Height: 512},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8SRGB,
			Dim3D:    driver.Dim3D{Width: 256, Height: 256},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
			Layers:   10,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 2048, Height: 2048},
			Layers:   6,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 800, Height: 600},
			Layers:   2,
			Levels:   1,
			Samples:  1,
		},
	} {
		tex, err := New2D(&param)
		if err != nil {
			t.Fatalf("New2D failed:\n%#v", err)
		}
		view := param.Layers - 1
		n := param.Size() * param.Width * param.Height

		data := make([]byte, n)
		for i := 0; i < param.Height; i++ {
			for j := 0; j < param.Width; j++ {
				px := [4]byte{byte(i * j % 256), byte(i % 256), byte(j % 256), 255}
				copy(data[4*i*param.Width+4*j:], px[:])
			}
		}
		dst := make([]byte, n)

		if err = tex.CopyToView(view, data, true); err != nil {
			t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
		}
		if x, err := tex.CopyFromView(view, dst); x != n || err != nil {
			t.Fatalf("Texture.CopyFromView:\nhave %d, %#v\nwant %d, nil", x, err, n)
		}
		checkData(data, dst, t)
		tex.Free()
	}

	// Each layer.
	for _, param := range [...]TexParam{
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
			Layers:   5,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8SRGB,
			Dim3D:    driver.Dim3D{Width: 256, Height: 256},
			Layers:   2,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 1600, Height: 900},
			Layers:   8,
			Levels:   1,
			Samples:  1,
		},
	} {
		tex, err := New2D(&param)
		if err != nil {
			t.Fatalf("New2D failed:\n%#v", err)
		}
		n := param.Size() * param.Width * param.Height

		data := make([]byte, n*param.Layers)
		for k := 0; k < param.Layers; k++ {
			for i := 0; i < param.Height; i++ {
				for j := 0; j < param.Width; j++ {
					px := [4]byte{byte((k + 127) % 256), byte(i % 256), byte(j % 256), 255}
					copy(data[k*n+4*i*param.Width+4*j:], px[:])
				}
			}
		}
		dst := make([]byte, n*param.Layers)

		for k := 0; k < param.Layers; k++ {
			if err = tex.CopyToView(k, data[k*n:k*n+n], true); err != nil {
				t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
			}
			if x, err := tex.CopyFromView(k, dst[k*n:k*n+n]); x != n || err != nil {
				t.Fatalf("Texture.CopyFromView:\nhave %d, %#v\nwant %d, nil", x, err, n)
			}
		}
		checkData(data, dst, t)
		tex.Free()
	}

	// Layer array.
	for _, param := range [...]TexParam{
		{
			PixelFmt: driver.RGBA8Unorm,
			Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
			Layers:   6,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8SRGB,
			Dim3D:    driver.Dim3D{Width: 2048, Height: 1024},
			Layers:   3,
			Levels:   1,
			Samples:  1,
		},
	} {
		tex, err := New2D(&param)
		if err != nil {
			t.Fatalf("New2D failed:\n%#v", err)
		}
		view := param.Layers
		n := param.Size() * param.Width * param.Height

		data := make([]byte, n*param.Layers)
		for k := 0; k < param.Layers; k++ {
			for i := 0; i < param.Height; i++ {
				for j := 0; j < param.Width; j++ {
					px := [4]byte{byte((k + 127) % 256), byte(i % 256), byte(j % 256), 255}
					copy(data[k*n+4*i*param.Width+4*j:], px[:])
				}
			}
		}
		dst := make([]byte, n*param.Layers)

		if err = tex.CopyToView(view, data, true); err != nil {
			t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
		}
		if x, err := tex.CopyFromView(view, dst); x != n*param.Layers || err != nil {
			t.Fatalf("Texture.CopyFromView:\nhave %d, %#v\nwant %d, nil", x, err, n*param.Layers)
		}
		checkData(data, dst, t)
		tex.Free()
	}
}

func TestViewCopyPending(t *testing.T) {
	param := TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 512},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	}
	for param.Size()*param.Width*param.Height > texStgBlock*texStgNBit {
		param.Width /= 2
		param.Height /= 2
	}
	tex, err := NewTarget(&param)
	if err != nil {
		t.Fatalf("NewTarget failed:\n%#v", err)
	}
	data := make([]byte, param.Size()*param.Width*param.Height)
	for i := range data {
		data[i] = 255
	}

	// Stage uncommitted.
	if err = tex.CopyToView(0, data, false); err != nil {
		t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
	}

	defer func() {
		if x := recover(); x != nil {
			const want = "layout already pending"
			if x != want {
				t.Fatalf("Texture.CopyToView: recover():\nhave %v\nwant %s", x, want)
			}
		} else {
			t.Fatal("Texture.CopyToView: should have panicked")
		}

		// Panicking corrupts the global state;
		// reset it so other tests can run safely.
		texStgMu.Lock()
		// Likely to block forever.
		//for i := 0; i < cap(texStg); i++ {
		//	(<-texStg).free()
		//}
		texStg = make(chan *texStgBuffer, cap(texStg))
		for i := 0; i < cap(texStg); i++ {
			s, err := newTexStg(texStgBlock * texStgNBit)
			if err != nil {
				s = &texStgBuffer{}
			}
			texStg <- s
		}
		texStgCache = texStgCache[:0]
		wk := <-texStgWk
		wk.Work = wk.Work[:0]
		wk.Err = nil
		texStgWk <- wk
		texStgMu.Unlock()

		tex.Free()
	}()

	// Stage a second copy operation.
	// Must panic.
	tex.CopyToView(0, data, true)
	t.Fatal("Texture.CopyToView: expected to be unreachable")
}

func TestViewCopyPendingNoPanic(t *testing.T) {
	param := TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 512},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	}
	for param.Size()*param.Width*param.Height > texStgBlock*texStgNBit {
		param.Width /= 2
		param.Height /= 2
	}
	tex, err := NewTarget(&param)
	if err != nil {
		t.Fatalf("NewTarget failed:\n%#v", err)
	}
	data := make([]byte, param.Size()*param.Width*param.Height)
	for i := range data {
		data[i] = 255
	}

	// Stage and commit.
	if err = tex.CopyToView(0, data, true); err != nil {
		t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
	}

	// Stage a second copy operation.
	// Must not panic.
	if err := tex.CopyToView(0, data, true); err != nil {
		t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
	}

	// Stage a third copy operation, uncommitted.
	// Must not panic.
	if err := tex.CopyToView(0, data, false); err != nil {
		t.Fatalf("Texture.CopyToView:\nhave %#v\nwant nil", err)
	}
	// Cannot free this while pending.
	//tex.Free()
}

func TestCommitTexStg(t *testing.T) {
	concCommit := func() {
		const n = 8
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			go func() {
				time.Sleep(time.Nanosecond * 20)
				errs <- commitTexStg()
			}()
		}
		for i := n; i > 0; i-- {
			if err := <-errs; err != nil {
				t.Fatalf("commitTexStg failed:\n%#v", err)
			}
		}
	}

	concCommit()

	param := TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	}
	for param.Size()*param.Width*param.Height*2 > texStgBlock*texStgNBit {
		param.Width /= 2
		param.Height /= 2
	}

	data := make([]byte, 10*param.Size()*param.Width*param.Height)
	for i := 0; i < len(data); i += 4 {
		copy(data[i:i+4], []byte{byte(i) % 255, byte(i+i) % 255, byte(i*i) % 255, 255})
	}
	dst := make([]byte, len(data))

	tex1, err := New2D(&param)
	if err != nil {
		t.Fatalf("New2D failed:\n%#v", err)
	}
	param.Layers = 4
	tex2, err := New2D(&param)
	if err != nil {
		t.Fatalf("New2D failed:\n%#v", err)
	}

	// Copy single-layer texture uncommitted
	// and then commit.
	if err = tex1.CopyToView(0, data, false); err != nil {
		t.Fatalf("Texture.CopyToView failed:\n%#v", err)
	}
	if tex1.layouts[0].Load() != invalLayout {
		t.Fatal("Texture.CopyToView: should not have committed")
	}
	concCommit()
	if tex1.layouts[0].Load() == invalLayout {
		t.Fatal("commitTexStg: should have set a valid layout")
	}
	n := tex1.ViewSize(0)
	if x, err := tex1.CopyFromView(0, dst); err != nil || x != n {
		t.Fatalf("Texture.CopyFromView: unexpected result\nhave %d, %#v\nwant %d, nil", x, err, n)
	}
	checkData(data, dst[:n], t)

	// Copy different layers of different textures
	// uncommitted and then commit.
	if err = tex2.CopyToView(1, data[16:], false); err != nil {
		t.Fatalf("Texture.CopyToView failed:\n%#v", err)
	}
	if err = tex1.CopyToView(0, data[48:], false); err != nil {
		t.Fatalf("Texture.CopyToView failed:\n%#v", err)
	}
	if tex2.layouts[1].Load() != invalLayout || tex1.layouts[0].Load() != invalLayout {
		t.Fatal("Texture.CopyToView: should not have committed")
	}
	concCommit()
	if tex2.layouts[1].Load() == invalLayout || tex1.layouts[0].Load() == invalLayout {
		t.Fatal("commitTexStg: should have set a valid layout")
	}
	n = tex2.ViewSize(1)
	if x, err := tex2.CopyFromView(1, dst); err != nil || x != n {
		t.Fatalf("Texture.CopyFromView: unexpected result\nhave %d, %#v\nwant %d, nil", x, err, n)
	}
	n = tex2.ViewSize(1)
	if x, err := tex1.CopyFromView(0, dst[n:]); err != nil || x != n {
		t.Fatalf("Texture.CopyFromView: unexpected result\nhave %d, %#v\nwant %d, nil", x, err, n)
	}
	checkData(data[16:], dst[:n], t)
	checkData(data[48:], dst[n:n*2], t)

	// Copy different layers of the same texture
	// uncommitted and then commit.
	if err = tex2.CopyToView(0, data[256:], false); err != nil {
		t.Fatalf("Texture.CopyToView failed:\n%#v", err)
	}
	if err = tex2.CopyToView(1, data[len(data)/3:], false); err != nil {
		t.Fatalf("Texture.CopyToView failed:\n%#v", err)
	}
	if tex2.layouts[0].Load() != invalLayout || tex2.layouts[1].Load() != invalLayout {
		t.Fatal("Texture.CopyToView: should not have committed")
	}
	concCommit()
	if tex2.layouts[0].Load() == invalLayout || tex2.layouts[1].Load() == invalLayout {
		t.Fatal("commitTexStg: should have set a valid layout")
	}
	n = tex2.ViewSize(0)
	if x, err := tex2.CopyFromView(0, dst); err != nil || x != n {
		t.Fatalf("Texture.CopyFromView: unexpected result\nhave %d, %#v\nwant %d, nil", x, err, n)
	}
	n = tex2.ViewSize(1)
	if x, err := tex2.CopyFromView(1, dst[n:]); err != nil || x != n {
		t.Fatalf("Texture.CopyFromView: unexpected result\nhave %d, %#v\nwant %d, nil", x, err, n)
	}
	checkData(data[256:], dst[:n], t)
	checkData(data[len(data)/3:], dst[n:n*2], t)

	tex2.Free()
	tex1.Free()
}

func TestTransition(t *testing.T) {
	tex1, err := NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 768},
		Layers:   3,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("NewTarget failed:\n%#v", err)
	}
	tex2, err := NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 256, Height: 256},
		Layers:   6,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("NewCube failed:\n%#v", err)
	}
	wk := make(chan *driver.WorkItem, 1)
	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		t.Fatalf("driver.GPU.NewCmdBuffer failed:\n%#v", err)
	}
	if err = cb.Begin(); err != nil {
		t.Fatalf("driver.CmdBuffer.Begin failed:\n%#v", err)
	}

	checkTex1 := func(l0, l1, l2 driver.Layout) {
		if x := tex1.layouts[0].Load(); x != int64(l0) {
			t.Fatalf("tex1.layouts[0]:\nhave %d\nwant %d", x, l0)
		}
		if x := tex1.layouts[1].Load(); x != int64(l1) {
			t.Fatalf("tex1.layouts[1]:\nhave %d\nwant %d", x, l1)
		}
		if x := tex1.layouts[2].Load(); x != int64(l2) {
			t.Fatalf("tex1.layouts[2]:\nhave %d\nwant %d", x, l2)
		}
	}
	checkTex2 := func(l0, l1, l2, l3, l4, l5 driver.Layout) {
		if x := tex2.layouts[0].Load(); x != int64(l0) {
			t.Fatalf("tex2.layouts[0]:\nhave %d\nwant %d", x, l0)
		}
		if x := tex2.layouts[1].Load(); x != int64(l1) {
			t.Fatalf("tex2.layouts[1]:\nhave %d\nwant %d", x, l1)
		}
		if x := tex2.layouts[2].Load(); x != int64(l2) {
			t.Fatalf("tex2.layouts[2]:\nhave %d\nwant %d", x, l2)
		}
		if x := tex2.layouts[3].Load(); x != int64(l3) {
			t.Fatalf("tex2.layouts[3]:\nhave %d\nwant %d", x, l3)
		}
		if x := tex2.layouts[4].Load(); x != int64(l4) {
			t.Fatalf("tex2.layouts[4]:\nhave %d\nwant %d", x, l4)
		}
		if x := tex2.layouts[5].Load(); x != int64(l5) {
			t.Fatalf("tex2.layouts[5]:\nhave %d\nwant %d", x, l5)
		}
	}

	checkTex1(driver.LUndefined, driver.LUndefined, driver.LUndefined)
	checkTex2(driver.LUndefined, driver.LUndefined, driver.LUndefined, driver.LUndefined, driver.LUndefined, driver.LUndefined)

	tex1.transition(0, cb, driver.LColorTarget, driver.Barrier{})
	checkTex1(invalLayout, driver.LUndefined, driver.LUndefined)
	tex1.transition(2, cb, driver.LShaderRead, driver.Barrier{})
	checkTex1(invalLayout, driver.LUndefined, invalLayout)
	tex2.transition(0, cb, driver.LShaderRead, driver.Barrier{})
	checkTex2(invalLayout, invalLayout, invalLayout, invalLayout, invalLayout, invalLayout)

	if err = cb.End(); err != nil {
		t.Fatalf("driver.CmdBuffer.End failed:\n%#v", err)
	}
	if err = ctxt.GPU().Commit(&driver.WorkItem{Work: []driver.CmdBuffer{cb}}, wk); err != nil {
		t.Fatalf("driver.GPU.Commit failed:\n%#v", err)
	}
	if err = (<-wk).Err; err != nil {
		t.Fatalf("driver.GPU.Commit: (<-ch).Err\n%#v", err)
	}

	tex2.setLayout(0, driver.LShaderRead)
	checkTex2(driver.LShaderRead, driver.LShaderRead, driver.LShaderRead, driver.LShaderRead, driver.LShaderRead, driver.LShaderRead)
	tex1.setLayout(0, driver.LColorTarget)
	checkTex1(driver.LColorTarget, driver.LUndefined, invalLayout)
	tex1.setLayout(2, driver.LShaderRead)
	checkTex1(driver.LColorTarget, driver.LUndefined, driver.LShaderRead)

	if err = cb.Begin(); err != nil {
		t.Fatalf("driver.CmdBuffer.Begin failed:\n%#v", err)
	}

	tex2.transition(0, cb, driver.LCopyDst, driver.Barrier{})
	checkTex2(invalLayout, invalLayout, invalLayout, invalLayout, invalLayout, invalLayout)
	tex1.transition(2, cb, driver.LCopyDst, driver.Barrier{})
	checkTex1(driver.LColorTarget, driver.LUndefined, invalLayout)

	cb.Reset()

	tex2.setLayout(0, driver.LUndefined)
	checkTex2(driver.LUndefined, driver.LUndefined, driver.LUndefined, driver.LUndefined, driver.LUndefined, driver.LUndefined)
	tex1.setLayout(2, driver.LUndefined)
	checkTex1(driver.LColorTarget, driver.LUndefined, driver.LUndefined)

	cb.Destroy()
	tex2.Free()
	tex1.Free()
}

func TestTransitionPanic(t *testing.T) {
	tex, err := NewTarget(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 768},
		Layers:   10,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("NewTarget failed:\n%#v", err)
	}
	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		t.Fatalf("driver.GPU.NewCmdBuffer failed:\n%#v", err)
	}
	if err = cb.Begin(); err != nil {
		t.Fatalf("driver.CmdBuffer.Begin failed:\n%#v", err)
	}

	defer func() {
		if x := recover(); x != nil {
			const want = "layout already pending"
			if x != want {
				t.Fatalf("Texture.transition: recover():\nhave %v\nwant %s", x, want)
			}
			for _, i := range [...]int{1, 0, 4} {
				if x := tex.layouts[i].Load(); x != invalLayout {
					t.Fatalf("Texture.transition: Texture.layouts[%d]:\nhave %d\nwant %d", i, x, invalLayout)
				}
			}
			for _, i := range [...]int{2, 3, 5, 6, 7, 8, 9} {
				if x := tex.layouts[i].Load(); x != int64(driver.LUndefined) {
					t.Fatalf("Texture.transition: Texture.layouts[%d]:\nhave %d\nwant %d", i, x, driver.LUndefined)
				}
			}
		} else {
			t.Fatal("Texture.transition: should have panicked")
		}
		cb.Destroy()
		tex.Free()
	}()

	// Ok.
	tex.transition(1, cb, driver.LColorTarget, driver.Barrier{})
	// Ok.
	tex.transition(0, cb, driver.LColorTarget, driver.Barrier{})
	// Ok.
	tex.transition(4, cb, driver.LShaderRead, driver.Barrier{})
	// Must panic.
	tex.transition(tex.param.Layers, cb, driver.LColorTarget, driver.Barrier{})
	t.Fatal("Texture.transition: expected to be unreachable")
}

func TestSetLayoutPanic(t *testing.T) {
	tex, err := NewCube(&TexParam{
		PixelFmt: driver.RGBA8Unorm,
		Dim3D:    driver.Dim3D{Width: 512, Height: 512},
		Layers:   12,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("NewCube failed:\n%#v", err)
	}
	wk := make(chan *driver.WorkItem, 1)
	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		t.Fatalf("driver.GPU.NewCmdBuffer failed:\n%#v", err)
	}
	if err = cb.Begin(); err != nil {
		t.Fatalf("driver.CmdBuffer.Begin failed:\n%#v", err)
	}

	defer func() {
		if x := recover(); x != nil {
			const want = "layout not pending"
			if x != want {
				t.Fatalf("Texture.setLayout: recover():\nhave %v\nwant %s", x, want)
			}
			for i := 6; i < 12; i++ {
				if x := tex.layouts[i].Load(); x != int64(driver.LShaderRead) {
					t.Fatalf("Texture.setLayout: Texture.layouts[%d]:\nhave %d\nwant %d", i, x, driver.LShaderRead)
				}
			}
			for i := 0; i < 6; i++ {
				if x := tex.layouts[i].Load(); x != int64(driver.LUndefined) {
					t.Fatalf("Texture.setLayout: Texture.layouts[%d]:\nhave %d\nwant %d", i, x, driver.LUndefined)
				}
			}
		} else {
			t.Fatal("Texture.setLayout: should have panicked")
		}
		cb.Destroy()
		tex.Free()
	}()

	tex.transition(1, cb, driver.LColorTarget, driver.Barrier{})

	if err = cb.End(); err != nil {
		t.Fatalf("driver.CmdBuffer.End failed:\n%#v", err)
	}
	if err = ctxt.GPU().Commit(&driver.WorkItem{Work: []driver.CmdBuffer{cb}}, wk); err != nil {
		t.Fatalf("driver.GPU.Commit failed:\n%#v", err)
	}
	if err = (<-wk).Err; err != nil {
		t.Fatalf("driver.GPU.Commit: (<-ch).Err\n%#v", err)
	}

	// Ok.
	tex.setLayout(1, driver.LShaderRead)
	// Must panic.
	tex.setLayout(0, driver.LShaderRead)
	t.Fatal("Texture.setLayout: expected to be unreachable")
}
