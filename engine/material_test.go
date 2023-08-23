// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"strings"
	"testing"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/texture"
)

func TestMaterial(t *testing.T) {
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

	oneChTex, err := texture.New2D(&texture.TexParam{
		PixelFmt: driver.R8un,
		Dim3D:    driver.Dim3D{Width: 1024, Height: 1024},
		Layers:   1,
		Levels:   1,
		Samples:  1,
	})
	if err != nil {
		t.Fatalf("texture.New2D failed:\n%#v", err)
	}

	twoChTex, err := texture.New2D(&texture.TexParam{
		PixelFmt: driver.RG8un,
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

	check := func(mat *Material, err error, prop any) {
		if err != nil || mat == nil {
			t.Fatalf("New*:\nhave %v, %#v\nwant non-nil, nil", mat, err)
		}
		var want Material
		switch prop := prop.(type) {
		case *PBR:
			want = Material{
				baseColor:  prop.BaseColor.TexRef,
				metalRough: prop.MetalRough.TexRef,
				normal:     prop.Normal.TexRef,
				occlusion:  prop.Occlusion.TexRef,
				emissive:   prop.Emissive.TexRef,
				layout:     prop.shaderLayout(),
			}
		case *Unlit:
			want = Material{
				baseColor: prop.BaseColor.TexRef,
				layout:    prop.shaderLayout(),
			}
		default:
			t.Fatalf("unexpected Material property")
		}
		if mat.baseColor != want.baseColor {
			t.Fatalf("New*: Material.baseColor\nhave %v\nwant %v", mat.baseColor, want.baseColor)
		}
		if mat.metalRough != want.metalRough {
			t.Fatalf("New*: Material.metalRough\nhave %v\nwant %v", mat.metalRough, want.metalRough)
		}
		if mat.normal != want.normal {
			t.Fatalf("New*: Material.normal\nhave %v\nwant %v", mat.normal, want.normal)
		}
		if mat.occlusion != want.occlusion {
			t.Fatalf("New*: Material.occlusion\nhave %v\nwant %v", mat.occlusion, want.occlusion)
		}
		if mat.emissive != want.emissive {
			t.Fatalf("New*: Material.emissive\nhave %v\nwant %v", mat.emissive, want.emissive)
		}
		if mat.layout != want.layout {
			// TODO: Should validate layout contents.
			t.Fatalf("New*: Material.layout\nhave %v\nwant %v", mat.layout, want.layout)
		}
	}

	checkFail := func(mat *Material, err error, reason string) {
		if err == nil || mat != nil {
			t.Fatalf("New*:\nhave %v, %#v\nwant nil, non-nil", mat, err)
		}
		if !strings.HasSuffix(err.Error(), reason) {
			t.Fatalf("New*: error.Error\nhave \"%s\"\nwant \"%s\"", err.Error(), "material: "+reason)
		}
	}

	// NewPBR calls that must succeed.
	t.Run("PBR", func(t *testing.T) {
		var pbr PBR
		mat, err := NewPBR(&pbr)
		check(mat, err, &pbr)

		pbr = PBR{
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
		}
		mat, err = NewPBR(&pbr)
		check(mat, err, &pbr)

		pbr = PBR{
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
		}
		mat, err = NewPBR(&pbr)
		check(mat, err, &pbr)

		pbr = PBR{
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
		}
		mat, err = NewPBR(&pbr)
		check(mat, err, &pbr)

		pbr = PBR{
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
		}
		mat, err = NewPBR(&pbr)
		check(mat, err, &pbr)

		pbr = PBR{
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
		}
		mat, err = NewPBR(&pbr)
		check(mat, err, &pbr)

		pbr = PBR{
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
		}
		mat, err = NewPBR(&pbr)
		check(mat, err, &pbr)
	})

	// NewUnlit calls that must succeed.
	t.Run("Unlit", func(t *testing.T) {
		var unlit Unlit
		mat, err := NewUnlit(&unlit)
		check(mat, err, &unlit)

		unlit = Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		}
		mat, err = NewUnlit(&unlit)
		check(mat, err, &unlit)

		unlit = Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 2, splr, UVSet1},
				Factor: [4]float32{1, 1, 1, 1},
			},
			AlphaMode:   AlphaBlend,
			DoubleSided: true,
		}
		mat, err = NewUnlit(&unlit)
		check(mat, err, &unlit)

		unlit = Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 1, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: 0.5,
			DoubleSided: false,
		}
		mat, err = NewUnlit(&unlit)
		check(mat, err, &unlit)

		unlit = Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{0.1, 0.01, 0.125, 1},
			},
			AlphaMode:   AlphaOpaque,
			DoubleSided: false,
		}
		mat, err = NewUnlit(&unlit)
		check(mat, err, &unlit)

		unlit = Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet1},
				Factor: [4]float32{},
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: 2,
			DoubleSided: false,
		}
		mat, err = NewUnlit(&unlit)
		check(mat, err, &unlit)

		// This has the same effect as AlphaOpaque.
		unlit = Unlit{
			BaseColor: BaseColor{
				TexRef: TexRef{},
				Factor: [4]float32{0.6, 0.7, 0.8, 1},
			},
			AlphaMode:   AlphaMask,
			AlphaCutoff: -100,
			DoubleSided: false,
		}
		mat, err = NewUnlit(&unlit)
		check(mat, err, &unlit)
	})

	// NewPBR calls that must fail.
	t.Run("PBRFail", func(t *testing.T) {
		mat, err := NewPBR(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, nil, UVSet0},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "nil TexRef.Sampler")

		mat, err = NewPBR(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet1 + 1},
				Factor: [4]float32{1, 1, 1, 1},
			},
		})
		checkFail(mat, err, "undefined UV set constant")

		mat, err = NewPBR(&PBR{
			BaseColor: BaseColor{
				TexRef: TexRef{color, 0, splr, UVSet0},
				Factor: [4]float32{1, 1, 1, 1.1},
			},
		})
		checkFail(mat, err, "BaseColor.Factor outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{oneChTex, 0, splr, UVSet0},
				Metalness: 1,
				Roughness: 0.5,
			},
		})
		checkFail(mat, err, "MetalRough.Texture has insufficient channels")

		mat, err = NewPBR(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: -0.1,
				Roughness: 0.5,
			},
		})
		checkFail(mat, err, "MetalRough.Metalness outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1.2,
				Roughness: 0.5,
			},
		})
		checkFail(mat, err, "MetalRough.Metalness outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1,
				Roughness: 1000,
			},
		})
		checkFail(mat, err, "MetalRough.Roughness outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			MetalRough: MetalRough{
				TexRef:    TexRef{occMetal, 0, splr, UVSet0},
				Metalness: 1,
				Roughness: -0.01,
			},
		})
		checkFail(mat, err, "MetalRough.Roughness outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			Normal: Normal{
				TexRef: TexRef{normal, -1, splr, UVSet0},
				Scale:  1,
			},
		})
		checkFail(mat, err, "invalid TexRef.View")

		mat, err = NewPBR(&PBR{
			Normal: Normal{
				TexRef: TexRef{twoChTex, 0, splr, UVSet0},
				Scale:  1,
			},
		})
		checkFail(mat, err, "Normal.Texture has insufficient channels")

		mat, err = NewPBR(&PBR{
			Normal: Normal{
				TexRef: TexRef{normal, 0, splr, UVSet0},
				Scale:  -1,
			},
		})
		checkFail(mat, err, "Normal.Scale less than 0.0")

		mat, err = NewPBR(&PBR{
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet0},
				Strength: 2,
			},
		})
		checkFail(mat, err, "Occlusion.Strength outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			Occlusion: Occlusion{
				TexRef:   TexRef{occMetal, 0, splr, UVSet0},
				Strength: -3,
			},
		})
		checkFail(mat, err, "Occlusion.Strength outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			Emissive: Emissive{
				TexRef: TexRef{twoChTex, 0, splr, UVSet0},
				Factor: [3]float32{1, 1, 1},
			},
		})
		checkFail(mat, err, "Emissive.Texture has insufficient channels")

		mat, err = NewPBR(&PBR{
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{1, 1, -1},
			},
		})
		checkFail(mat, err, "Emissive.Factor outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{
			Emissive: Emissive{
				TexRef: TexRef{emissive, 0, splr, UVSet0},
				Factor: [3]float32{2},
			},
		})
		checkFail(mat, err, "Emissive.Factor outside [0.0, 1.0] interval")

		mat, err = NewPBR(&PBR{AlphaMode: AlphaMask + 1})
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
