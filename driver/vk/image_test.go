// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/driver"
)

func TestImage(t *testing.T) {
	cases := [...]struct {
		pf      driver.PixelFmt
		size    driver.Dim3D
		layers  int
		levels  int
		samples int
		usage   driver.Usage
	}{
		{driver.RGBA8un, driver.Dim3D{Width: 1024, Height: 1024, Depth: 0}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 1000, Height: 1000, Depth: 0}, 1, 1, 1, driver.UShaderRead | driver.UShaderWrite},
		{driver.RGBA8un, driver.Dim3D{Width: 1600, Height: 1200, Depth: 0}, 1, 1, 1, driver.UGeneric},
		{driver.RGBA8n, driver.Dim3D{Width: 2048, Height: 1600, Depth: 0}, 8, 12, 1, driver.UShaderSample},
		{driver.RGBA8n, driver.Dim3D{Width: 1600, Height: 2048, Depth: 0}, 8, 12, 1, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 1, driver.URenderTarget},
		{driver.BGRA8un, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 1, driver.URenderTarget},
		{driver.D16un, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 1, driver.URenderTarget},
		{driver.D32f, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 1, driver.URenderTarget},
		{driver.S8ui, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 1, driver.URenderTarget},
		{driver.D24unS8ui, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 8, driver.URenderTarget},
		{driver.D32fS8ui, driver.Dim3D{Width: 1280, Height: 768, Depth: 0}, 2, 1, 1, driver.URenderTarget},
		{driver.RGBA8un, driver.Dim3D{Width: 1024, Height: 512, Depth: 0}, 1, 1, 4, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 512, Height: 1024, Depth: 0}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 1024, Height: 0, Depth: 0}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 1, Height: 1024, Depth: 0}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 1, Height: 1, Depth: 1024}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA16f, driver.Dim3D{Width: 1024, Height: 1024, Depth: 0}, 4, 11, 1, driver.UGeneric},
		{driver.RG16f, driver.Dim3D{Width: 1024, Height: 1024, Depth: 0}, 6, 11, 1, driver.UGeneric},
		{driver.R16f, driver.Dim3D{Width: 1024, Height: 1024, Depth: 0}, 6, 11, 1, driver.UGeneric},
		{driver.RG8un, driver.Dim3D{Width: 2000, Height: 1000, Depth: 0}, 1, 11, 1, driver.UGeneric},
		{driver.R8un, driver.Dim3D{Width: 2000, Height: 1000, Depth: 0}, 1, 11, 1, driver.UGeneric},
		{driver.RGBA8un, driver.Dim3D{Width: 1, Height: 0, Depth: 0}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA8un, driver.Dim3D{Width: 1, Height: 1, Depth: 0}, 1, 1, 1, driver.UGeneric},
		{driver.RGBA8un, driver.Dim3D{Width: 1, Height: 1, Depth: 1}, 1, 1, 1, driver.UGeneric},
		{driver.RGBA8sRGB, driver.Dim3D{Width: 64, Height: 64, Depth: 0}, 16, 1, 1, driver.UShaderSample},
		{driver.BGRA8sRGB, driver.Dim3D{Width: 720, Height: 480, Depth: 0}, 10, 1, 1, driver.UShaderSample},
		{driver.RGBA8sRGB, driver.Dim3D{Width: 1024, Height: 1024, Depth: 0}, 1, 1, 1, driver.UShaderSample},
		{driver.RGBA16f, driver.Dim3D{Width: 128, Height: 128, Depth: 128}, 1, 8, 1, driver.UGeneric},
	}
	zi := image{}
	zm := memory{}
	for _, c := range cases {
		call := fmt.Sprintf("tDrv.NewImage(%v, %v, %v, %v, %v, %v)", c.pf, c.size, c.layers, c.levels, c.samples, c.usage)
		// NewImage.
		if img, err := tDrv.NewImage(c.pf, c.size, c.layers, c.levels, c.samples, c.usage); err == nil {
			if img == nil {
				t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
				continue
			}
			img := img.(*image)
			if img.m != nil {
				if img.m.d != &tDrv {
					t.Errorf("%s: img.m.d\nhave %p\nwant %p", call, img.m.d, &tDrv)
				}
				// The size can be greater than what was requested.
				// TODO: Multiply by pixel size.
				size := int64(c.size.Width * c.size.Height * c.size.Depth)
				if img.m.size < size {
					t.Errorf("%s: img.m.size\nhave %d\nwant at least %d", call, img.m.size, size)
				}
				if img.m.vis && int64(len(img.m.p)) != img.m.size {
					t.Errorf("%s: len(img.m.p)\nhave %d\nwant %d", call, len(img.m.p), img.m.size)
				}
				// NewImage should bind the memory and set this to true.
				if !img.m.bound {
					t.Errorf("%s: img.m.bound\nhave false\nwant true", call)
				}
				if img.m.mem == zm.mem {
					t.Errorf("%s: img.m.mem\nhave %v\nwant valid handle", call, img.m.mem)
				}
				if img.m.typ < 0 || img.m.typ >= int(tDrv.mprop.memoryTypeCount) {
					t.Errorf("%s: img.m.typ\nhave %d\nwant valid index", call, img.m.typ)
				} else {
					heap := int(tDrv.mprop.memoryTypes[img.m.typ].heapIndex)
					if img.m.heap != heap {
						t.Errorf("%s: img.m.heap\nhave %d\nwant %d", call, img.m.heap, heap)
					}
				}
			} else {
				t.Errorf("%s: img.m\nhave nil\nwant non-nil", call)
			}
			if img.img == zi.img {
				t.Errorf("%s: img.img\nhave %v\nwant valid handle", call, img.img)
			}
			// Destroy.
			img.Destroy()
			if *img != zi {
				t.Errorf("img.Destroy(): img\nhave %v\nwant %v", img, zi)
			}
		} else if img != nil {
			t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, img, err, err)
		} else {
			t.Logf("(error) %s: %v", call, err)
		}
	}
}

func TestImageView(t *testing.T) {
	type iview struct {
		typ    driver.ViewType
		layer  int
		layers int
		level  int
		levels int
	}
	cases := [...]struct {
		pf      driver.PixelFmt
		size    driver.Dim3D
		layers  int
		levels  int
		samples int
		usage   driver.Usage
		iv      []iview
	}{
		{driver.RGBA8un, driver.Dim3D{Width: 1024, Height: 1024}, 1, 11, 1, driver.UShaderSample, []iview{
			{driver.IView2D, 0, 1, 0, 1}, {driver.IView2D, 0, 1, 0, 11}, {driver.IView2D, 0, 1, 4, 5},
		}},
		{driver.RGBA16f, driver.Dim3D{Width: 1024, Height: 1024}, 1, 6, 1, driver.UGeneric, []iview{
			{driver.IView2D, 0, 1, 0, 1}, {driver.IView2D, 0, 1, 2, 1}, {driver.IView2D, 0, 1, 3, 3},
		}},
		{driver.BGRA8sRGB, driver.Dim3D{Width: 1280, Height: 768}, 1, 1, 8, driver.URenderTarget, []iview{
			{driver.IView2DMS, 0, 1, 0, 1}, {driver.IView2D, 0, 1, 0, 1},
		}},
		{driver.D24unS8ui, driver.Dim3D{Width: 1280, Height: 768}, 2, 1, 1, driver.URenderTarget, []iview{
			{driver.IView2D, 0, 1, 0, 1}, {driver.IView2DArray, 0, 2, 0, 1},
		}},
		{driver.D16un, driver.Dim3D{Width: 1280, Height: 768}, 2, 1, 1, driver.URenderTarget, []iview{
			{driver.IView2D, 0, 1, 0, 1}, {driver.IView2D, 1, 1, 0, 1},
		}},
		{driver.S8ui, driver.Dim3D{Width: 1280, Height: 768}, 3, 1, 1, driver.URenderTarget, []iview{
			{driver.IView2D, 2, 1, 0, 1}, {driver.IView2DArray, 0, 3, 0, 1},
		}},
		{driver.R8un, driver.Dim3D{Width: 4096}, 4, 1, 1, driver.UGeneric, []iview{
			{driver.IView1D, 0, 1, 0, 1}, {driver.IView1D, 3, 1, 0, 1}, {driver.IView1DArray, 0, 4, 0, 1},
		}},
		{driver.RG16f, driver.Dim3D{Width: 480, Height: 720, Depth: 5}, 1, 1, 1, driver.UGeneric, []iview{
			{driver.IView3D, 0, 1, 0, 1}, /*{driver.IView2DArray, 0, 5, 0, 1}, {driver.IView2D, 1, 1, 0, 1},*/
		}},
		{driver.RGBA8un, driver.Dim3D{Width: 512, Height: 512}, 16, 10, 1, driver.UShaderSample, []iview{
			// TODO: Enable cube array feature.
			{driver.IViewCube, 0, 6, 0, 1}, {driver.IViewCube, 4, 6, 0, 10}, /*{driver.IViewCubeArray, 0, 12, 0, 10},*/
		}},
	}
	zv := imageView{}
	for _, c := range cases {
		img, err := tDrv.NewImage(c.pf, c.size, c.layers, c.levels, c.samples, c.usage)
		if err != nil {
			call := fmt.Sprintf("tDrv.NewImage(%v, %v, %v, %v, %v, %v)", c.pf, c.size, c.layers, c.levels, c.samples, c.usage)
			t.Errorf("%s failed, cannot test NewView method", call)
			continue
		}
		for _, c := range c.iv {
			call := fmt.Sprintf("img.NewView(%v, %v, %v, %v, %v)", c.typ, c.layer, c.layers, c.level, c.levels)
			// NewView.
			if iv, err := img.NewView(c.typ, c.layer, c.layers, c.level, c.levels); err == nil {
				if iv == nil {
					t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
					continue
				}
				iv := iv.(*imageView)
				if iv.i != img {
					t.Errorf("%s: iv.i\nhave %p\nwant %p", call, iv.i, img)
				}
				if iv.view == zv.view {
					t.Errorf("%s: iv.view\nhave %v\nwant valid handle", call, iv.view)
				}
				// Destroy.
				iv.Destroy()
				if *iv != zv {
					t.Errorf("iv.Destroy()\nhave %v\nwant %v", *iv, zv)
				}
			} else if iv != nil {
				t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, iv, err, err)
			} else {
				t.Errorf("(error) %s: %v", call, err)
			}
		}
		img.Destroy()
	}
}

func TestPixelFmt(t *testing.T) {
	pfs := [...]driver.PixelFmt{
		driver.FInvalid,
		driver.RGBA8un,
		driver.RGBA8n,
		driver.RGBA8sRGB,
		driver.BGRA8un,
		driver.BGRA8sRGB,
		driver.RG8un,
		driver.RG8n,
		driver.R8un,
		driver.R8n,
		driver.RGBA16f,
		driver.RG16f,
		driver.R16f,
		driver.RGBA32f,
		driver.RG32f,
		driver.R32f,
		driver.D16un,
		driver.D32f,
		driver.S8ui,
		driver.D24unS8ui,
		driver.D32fS8ui,
	}
	for _, f := range pfs {
		if x := convPixelFmt(f); x < 0 || f.IsInternal() {
			t.Fatalf("convPixelFmt(%v)\nhave %v\nwant >= 0", f, x)
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
		if x := internalFmt(f); x > 0 || !x.IsInternal() {
			t.Fatalf("internalFmt(%v)\nhave %v\nwant < 0", f, x)
		} else if y := convPixelFmt(x); y != f {
			t.Fatalf("convPixelFmt(%v)\nhave %v\nwant %v", x, y, f)
		}
	}
}
