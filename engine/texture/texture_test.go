// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package texture

import (
	"strings"
	"testing"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
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
	usg := ^(driver.UShaderRead | driver.UShaderWrite | driver.UShaderSample | driver.URenderTarget)
	if tex.usage == 0 || tex.usage&usg != 0 {
		t.Fatalf("Texture.usage: unexpected flag(s) set:\n0x%x", tex.usage&usg)
	}
}

func Test2D(t *testing.T) {
	tex, err := New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
		if strings.HasPrefix(err.Error(), prefix) {
			t.Fatalf("New2D: unexpected error:\n%#v", err)
		}
	}
	tex.check(t)

	// param must not be nil.
	_, err = NewTarget(nil)
	switch {
	case err == nil:
		t.Fatal("New2D: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Depth must be 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Layers must be greater than 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Levels must be greater than 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Samples must be greater than 0.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Width must be no greater than the driver-imposed limit.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Height must be no greater than the driver-imposed limit.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Layers must be no greater than the driver-imposed limit.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Levels must be less than 1 + log₂(maxDim).
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Samples must be a power of two.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}

	// Either Levels or Samples must be 1.
	_, err = New2D(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("New2D: unexpected error:\n%#v", err)
	}
}

func TestCube(t *testing.T) {
	tex, err := NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
		if strings.HasPrefix(err.Error(), prefix) {
			t.Fatalf("NewCube: unexpected error:\n%#v", err)
		}
	}
	tex.check(t)

	// param must not be nil.
	_, err = NewTarget(nil)
	switch {
	case err == nil:
		t.Fatal("NewCube: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Depth must be 0.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Layers must be greater than 0.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Levels must be greater than 0.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Width must be no greater than the driver-imposed limit.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Height must be no greater than the driver-imposed limit.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Width and Height must be equal.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Layers must be no greater than the driver-imposed limit.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Layers must be a multiple of 6.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Levels must be less than 1 + log₂(maxDim).
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}

	// Samples must be 1.
	_, err = NewCube(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewCube: unexpected error:\n%#v", err)
	}
}

func TestTarget(t *testing.T) {
	tex, err := NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
		if strings.HasPrefix(err.Error(), prefix) {
			t.Fatalf("NewTarget: unexpected error:\n%#v", err)
		}
	}
	tex.check(t)

	// param must not be nil.
	_, err = NewTarget(nil)
	switch {
	case err == nil:
		t.Fatal("NewTarget: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Depth must be 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Layers must be greater than 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Levels must be greater than 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Samples must be greater than 0.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Width must be no greater than the driver-imposed limit.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Height must be no greater than the driver-imposed limit.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Layers must be no greater than the driver-imposed limit.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Levels must be less than 1 + log₂(maxDim).
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Samples must be a power of two.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewTarget: unexpected error:\n%#v", err)
	}

	// Either Levels or Samples must be 1.
	_, err = NewTarget(&TexParam{
		PixelFmt: driver.RGBA8un,
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
	case !strings.HasPrefix(err.Error(), prefix):
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
		Cmp:      driver.CAlways,
		MinLOD:   0,
		MaxLOD:   0.25,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), prefix) {
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
		Cmp:      driver.CAlways,
		MinLOD:   0,
		MaxLOD:   0,
	})
	switch {
	case err != nil:
		if strings.HasPrefix(err.Error(), prefix) {
			t.Fatalf("NewSampler: unexpected error:\n%#v", err)
		}
	}
	s.check(t)

	// param must not be nil.
	_, err = NewSampler(nil)
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
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
		Cmp:      driver.CAlways,
		MinLOD:   0,
		MaxLOD:   0.25,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
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
		Cmp:      driver.CAlways,
		MinLOD:   -1,
		MaxLOD:   0.25,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
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
		Cmp:      driver.CAlways,
		MinLOD:   0,
		MaxLOD:   -1,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
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
		Cmp:      driver.CAlways,
		MinLOD:   1,
		MaxLOD:   0.25,
	})
	switch {
	case err == nil:
		t.Fatal("NewSampler: unexpected success")
	case !strings.HasPrefix(err.Error(), prefix):
		t.Fatalf("NewSampler: unexpected error:\n%#v", err)
	}
}

func TestTextureFree(t *testing.T) {
	texs := make([]*Texture, 0, 4)
	for i, x := range [4]TexParam{
		{
			PixelFmt: driver.RGBA8un,
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
			PixelFmt: driver.RGBA8sRGB,
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
			PixelFmt: driver.RGBA8un,
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
			PixelFmt: driver.RGBA16f,
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
				t.Fatalf("NewTarget failed:/%#v", err)
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
			Cmp:      driver.CAlways,
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

func TestStaging(t *testing.T) {
	var s *stagingBuffer
	var err error

	check := func(nbuf, nbm int) {
		if err != nil {
			t.Fatalf("driver.NewBuffer failed:\n%#v", err)
		}
		if x := int(s.buf.Cap()); x != nbuf {
			t.Fatalf("newStaging: buf.Cap\nhave %d\nwant %d", x, nbuf)
		}
		if x := s.bm.Len(); x != nbm {
			t.Fatalf("newStaging: bm.Len\nhave %d\nwant %d", x, nbm)
		}
	}
	checkFree := func() {
		if s.buf != nil {
			t.Fatalf("stagingBuffer.free: buf\nhave %v\nwant nil", s.buf)
		}
		if x := s.bm.Len(); x != 0 {
			t.Fatalf("stagingBuffer.free: bm.Len\nhave %d\nwant 0", x)
		}
	}

	const n = blockSize * nbit

	s, err = newStaging(n)
	check(n, nbit)
	s.free()
	checkFree()

	s, err = newStaging(n - 1)
	check(n, nbit)
	s.free()
	checkFree()

	s, err = newStaging(n + 1)
	check(n*2, nbit*2)
	s.free()
	checkFree()

	s, err = newStaging(1)
	check(n, nbit)
	s.free()
	checkFree()

	s, err = newStaging(n + n - 1)
	check(n*2, nbit*2)
	s.free()
	checkFree()

	x := 2048 * 2048 * 4
	s, err = newStaging(x)
	x = (x + n - 1) &^ (n - 1)
	check(x, x/blockSize)
	s.free()
	checkFree()
}

func TestInit(t *testing.T) {
	var s []*stagingBuffer
	for i := 0; i < cap(staging); i++ {
		select {
		case x := <-staging:
			if x.wk == nil || x.buf == nil {
				if x.wk != nil || x.buf != nil {
					t.Fatal("staging: unexpected non-nil wk or buf")
				}
				continue
			}
			if cap(x.wk) != 1 {
				t.Fatalf("staging: cap(wk)\nhave %d\nwant 1", cap(x.wk))
			}
			wk := <-x.wk
			if len(wk.Work) != 1 {
				t.Fatalf("staging: len((<-wk).Work)\nhave %d\nwant 1", len(wk.Work))
			}
			if wk.Work[0] == nil {
				t.Fatal("staging: (<-wk).Work[0]\nhave nil\nwant non-nil")
			}
			if wk.Err != nil {
				t.Fatalf("staging: (<-wk).Err\nhave %#v\nwant nil", wk.Err)
			}
			if x.buf.Cap() != blockSize*nbit {
				t.Fatalf("staging: buf.Cap:\nhave %d\nwant %d", x.buf.Cap(), blockSize*nbit)
			}
			x.wk <- wk
			s = append(s, x)
		default:
		}
	}
	if len(s) != cap(staging) {
		t.Fatalf("staging: unexpected len != cap\nhave %d\nwant %d", len(s), cap(staging))
	}
	for i := range s {
		staging <- s[i]
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
func TestCopy(t *testing.T) {
	// One layer.
	for _, param := range [...]TexParam{
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{256, 256, 0},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{512, 512, 0},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8sRGB,
			Dim3D:    driver.Dim3D{256, 256, 0},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{1024, 1024, 0},
			Layers:   1,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{1024, 1024, 0},
			Layers:   10,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{2048, 2048, 0},
			Layers:   6,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{800, 600, 0},
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
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{1024, 1024, 0},
			Layers:   5,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8sRGB,
			Dim3D:    driver.Dim3D{256, 256, 0},
			Layers:   2,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{1600, 900, 0},
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
			PixelFmt: driver.RGBA8un,
			Dim3D:    driver.Dim3D{1024, 1024, 0},
			Layers:   6,
			Levels:   1,
			Samples:  1,
		},
		{
			PixelFmt: driver.RGBA8sRGB,
			Dim3D:    driver.Dim3D{2048, 1024, 0},
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

func TestCopyPending(t *testing.T) {
	param := TexParam{
		PixelFmt: driver.RGBA8un,
		Dim3D:    driver.Dim3D{1024, 512, 0},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	}
	for param.Size()*param.Width*param.Height > blockSize*nbit {
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
			const want = "Texture.setPending: pending already"
			if x != want {
				t.Fatalf("Texture.CopyToView: recover():\nhave %v\nwant %s", x, want)
			}
		} else {
			t.Fatal("Texture.CopyToView: should have panicked")
		}
		tex.Free()
	}()

	// Stage a second copy operation.
	// Must panic.
	tex.CopyToView(0, data, true)
	t.Fatal("Texture.CopyToView: expected to be unreachable")
}

func TestCopyPendingNoPanic(t *testing.T) {
	param := TexParam{
		PixelFmt: driver.RGBA8un,
		Dim3D:    driver.Dim3D{1024, 512, 0},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	}
	for param.Size()*param.Width*param.Height > blockSize*nbit {
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
	tex.Free()
}
