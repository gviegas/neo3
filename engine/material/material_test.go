// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package material

import (
	"strings"
	"testing"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/texture"
)

func TestNew(t *testing.T) {
	color, err := texture.New2D(&texture.TexParam{
		PixelFmt: driver.RGBA8sRGB,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
		Layers:   4,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("texture.New2D failed:\n%#v", err)
	}

	occMetal, err := texture.New2D(&texture.TexParam{
		PixelFmt: driver.RGBA8un,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
		Layers:   3,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("texture.New2D failed:\n%#v", err)
	}

	normal, err := texture.New2D(&texture.TexParam{
		PixelFmt: driver.RGBA8un,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
		Layers:   2,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("texture.New2D failed:\n%#v", err)
	}

	emissive, err := texture.New2D(&texture.TexParam{
		PixelFmt: driver.RGBA8sRGB,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("texture.New2D failed:\n%#v", err)
	}

	splr, err := texture.NewSampler(&texture.SplrParam{
		Min:      driver.FLinear,
		Mag:      driver.FLinear,
		Mipmap:   driver.FNearest,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		Cmp:      driver.CNever,
		MinLOD:   0,
		MaxLOD:   0,
	})
	if err != nil {
		t.Fatalf("texture.NewSampler failed:\n%#v", err)
	}

	check := func(mat *Material, err error) {
		if err != nil || mat == nil {
			t.Fatalf("New*:\nhave %v, %#v\nwant non-nil, nil", mat, err)
		}
		if mat.prop == nil {
			t.Fatal("Material.prop: is any(nil)")
		}
		switch x := mat.prop.(type) {
		case *PBR:
			if x == nil {
				t.Fatal("Material.prop: is (*PBR)(nil)")
			}
		case *Unlit:
			if x == nil {
				t.Fatal("Material.prop: is (*Unlit)(nil)")
			}
		default:
			t.Fatalf("Material.prop: invalid type\nhave %T\nwant %T or %T", mat.prop, &PBR{}, &Unlit{})
		}
	}

	checkFail := func(mat *Material, err error, reason string) {
		if err == nil || mat != nil {
			t.Fatalf("New*:\nhave %v, %#v\nwant nil, non-nil", mat, err)
		}
		if !strings.HasSuffix(err.Error(), reason) {
			t.Fatalf("New*: error.Error\nhave \"%s\"\nwant \"%s%s\"", err.Error(), prefix, reason)
		}
	}

	// New calls that must succeed.
	t.Run("PBR", func(t *testing.T) {
		mat, err := New(&PBR{})
		check(mat, err)

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1,
				Roughness: 0.5,
			},
			Normal: Normal{
				TexRef: TexRef{normal, 0, splr, UVSet0},
				Scale:  1,
			},
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet0},
				Strength: 0.5,
			},
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{1, 1, 1},
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 1, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 0,
				Roughness: 1,
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: 0.5,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{1, 1, 1, 0.75},
			},
			MetalRough: MetalRough{
				TexRef:    TexRef{},
				Metalness: 1,
				Roughness: 0.25,
			},
			Normal: Normal{
				TexRef: TexRef{normal, 2, splr, UVSet0},
				Scale:  1,
			},
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 1, splr, UVSet0},
				Strength: 1,
			},
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{0.5, 0.5, 0.5},
			},
			AlphaMode:   AlphaBlend,
			DoubleSided: true,
		})
		check(mat, err)

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 3, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			MetalRough: MetalRough{
				TexRef:    TexRef{},
				Metalness: 0,
				Roughness: 0.5,
			},
			Normal: Normal{
				TexRef: TexRef{normal, 0, splr, UVSet0},
				Scale:  20,
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{1, 0.2, 0.05, 1},
			},
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 2, splr, UVSet1},
				Metalness: 0,
				Roughness: 0.9,
			},
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet0},
				Strength: 0.5,
			},
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{1, 1, 1},
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{1, 1, 1, 1},
			},
			MetalRough: MetalRough{
				TexRef:    TexRef{},
				Metalness: 1,
				Roughness: 1,
			},
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet1},
				Strength: 0.65,
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		})
		check(mat, err)
	})

	// NewUnlit calls that must succeed.
	t.Run("Unlit", func(t *testing.T) {
		mat, err := NewUnlit(&Unlit{})
		check(mat, err)

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 2, splr, UVSet1},
				Factor: [4]float32{1, 1, 1, 1},
			},
			AlphaMode:   AlphaBlend,
			DoubleSided: true,
		})
		check(mat, err)

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 1, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: 0.5,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{0.1, 0.01, 0.125, 1},
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		})
		check(mat, err)

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet1},
				Factor: [4]float32{},
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: 2,
			DoubleSided: false,
		})
		check(mat, err)

		// This has the same effect as AlphaOpaque.
		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{0.6, 0.7, 0.8, 1},
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: -100,
			DoubleSided: false,
		})
		check(mat, err)
	})

	// New calls that must fail.
	t.Run("PBRFail", func(t *testing.T) {
		mat, err := New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, nil, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "nil TexRef.Sampler")

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet1 + 1},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "undefined UV set constant")

		mat, err = New(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1.1},
			},
		})
		checkFail(mat, err, "BaseColor.Factor outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: -0.1,
				Roughness: 0.5,
			},
		})
		checkFail(mat, err, "MetalRough.Metalness outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1.2,
				Roughness: 0.5,
			},
		})
		checkFail(mat, err, "MetalRough.Metalness outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1,
				Roughness: 1000,
			},
		})
		checkFail(mat, err, "MetalRough.Roughness outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1,
				Roughness: -0.01,
			},
		})
		checkFail(mat, err, "MetalRough.Roughness outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			Normal: Normal{
				TexRef: TexRef{normal, -1, splr, UVSet0},
				Scale:  1,
			},
		})
		checkFail(mat, err, "invalid TexRef.View")

		mat, err = New(&PBR{
			Normal: Normal{
				TexRef: TexRef{normal, 0, splr, UVSet0},
				Scale:  -1,
			},
		})
		checkFail(mat, err, "Normal.Scale less than 0.0")

		mat, err = New(&PBR{
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet0},
				Strength: 2,
			},
		})
		checkFail(mat, err, "Occlusion.Strength outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet0},
				Strength: -3,
			},
		})
		checkFail(mat, err, "Occlusion.Strength outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{1, 1, -1},
			},
		})
		checkFail(mat, err, "Emissive.Factor outside [0.0, 1.0] interval")

		mat, err = New(&PBR{
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{2},
			},
		})
		checkFail(mat, err, "Emissive.Factor outside [0.0, 1.0] interval")

		mat, err = New(&PBR{AlphaMode: AlphaMask + 1})
		checkFail(mat, err, "undefined alpha mode constant")
	})

	// NewUnlit calls that must fail.
	t.Run("UnlitFail", func(t *testing.T) {
		mat, err := NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, color.Layers() + 1, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "invalid TexRef.View")

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, nil, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "nil TexRef.Sampler")

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, -1},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "undefined UV set constant")

		mat, err = NewUnlit(&Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet0},
				Factor: [4]float32{1, 1, -0.2, 1},
			},
		})
		checkFail(mat, err, "BaseColor.Factor outside [0.0, 1.0] interval")

		mat, err = NewUnlit(&Unlit{AlphaMode: -1})
		checkFail(mat, err, "undefined alpha mode constant")
	})

	color.Free()
	occMetal.Free()
	normal.Free()
	emissive.Free()
	splr.Free()
}
