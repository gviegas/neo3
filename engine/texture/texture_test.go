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
	if tex.image == nil {
		t.Fatal("Texture.image: unexpected nil value")
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
