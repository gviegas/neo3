// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/driver"
)

// tRP contains attachments and subpasses for testing.
var tRP = [...]struct {
	att []driver.Attachment
	sub []driver.Subpass
}{
	{
		tAttach[:2],
		tSubp[:1],
	},
	{
		[]driver.Attachment{tAttach[0], tAttach[6], tAttach[2], tAttach[3]},
		tSubp[1:2],
	},
	{
		tAttach[:3],
		tSubp[2:3],
	},
	{
		tAttach[:5],
		tSubp[3:4],
	},
	{
		nil,
		tSubp[4:5],
	},
	{
		tAttach[:],
		tSubp[5:6],
	},
	{
		tAttach[:],
		tSubp[6:7],
	},
	{
		tAttach[:],
		[]driver.Subpass{tSubp[0], tSubp[2]},
	},
	{
		tAttach[:],
		[]driver.Subpass{tSubp[0], tSubp[4], tSubp[1]},
	},
	{
		tAttach[:],
		[]driver.Subpass{tSubp[3], tSubp[4], tSubp[2], tSubp[6]},
	},
	{
		tAttach[:],
		tSubp[1:5],
	},
	{
		tAttach[:],
		tSubp[2:6],
	},
	{
		tAttach[:],
		tSubp[:],
	},
}

// tSubp contains subpasses for testing.
var tSubp = [7]driver.Subpass{
	{
		Color: []int{0},
		DS:    1,
		MSR:   nil,
		Wait:  true,
	},
	{
		Color: []int{0, 2},
		DS:    3,
		MSR:   nil,
		Wait:  false,
	},
	{
		Color: []int{2},
		DS:    -1,
		MSR:   []int{},
		Wait:  true,
	},
	{
		Color: []int{0},
		DS:    4,
		MSR:   nil,
		Wait:  false,
	},
	{
		Color: nil,
		DS:    -1,
		MSR:   nil,
		Wait:  false,
	},
	{
		Color: []int{5},
		DS:    1,
		MSR:   []int{6},
		Wait:  true,
	},
	{
		Color: []int{5},
		DS:    -1,
		MSR:   []int{6},
		Wait:  false,
	},
}

// tAttach contains attachments for testing.
var tAttach = [7]driver.Attachment{
	{
		Format:  driver.RGBA8un,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LClear},
		Store:   [2]driver.StoreOp{driver.SStore},
	},
	{
		Format:  driver.D32fS8ui,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LClear, driver.LClear},
		Store:   [2]driver.StoreOp{driver.SDontCare, driver.SDontCare},
	},
	{
		Format:  driver.BGRA8sRGB,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LLoad},
		Store:   [2]driver.StoreOp{driver.SStore},
	},
	{
		Format:  driver.D16un,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LClear},
		Store:   [2]driver.StoreOp{driver.SStore},
	},
	{
		Format:  driver.S8ui,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LClear},
		Store:   [2]driver.StoreOp{driver.SDontCare},
	},
	{
		Format:  driver.RGBA16f,
		Samples: 8,
		Load:    [2]driver.LoadOp{driver.LClear},
		Store:   [2]driver.StoreOp{driver.SDontCare},
	},
	{
		Format:  driver.RGBA16f,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LDontCare},
		Store:   [2]driver.StoreOp{driver.SStore},
	},
}

func TestRenderPass(t *testing.T) {
	zp := renderPass{}
	for _, c := range tRP {
		call := fmt.Sprintf("tDrv.NewRenderPass(%v, %v)", c.att, c.sub)
		// NewRenderPass.
		if p, err := tDrv.NewRenderPass(c.att, c.sub); err == nil {
			if p == nil {
				t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
				continue
			}
			p := p.(*renderPass)
			if p.d != &tDrv {
				t.Errorf("%s: p.d\nhave %p\nwant %p", call, p.d, &tDrv)
			}
			if p.pass == zp.pass {
				t.Errorf("%s: p.pass\nhave %v\nwant valid handle", call, p.pass)
			}
			// Destroy.
			p.Destroy()
			if p.d != nil {
				t.Errorf("p.Destroy(): p.d\nhave %p\nwant nil", p.d)
			}
			if p.pass != zp.pass {
				t.Errorf("p.Destroy(): p.pass\nhave %v\nwant null handle", p.pass)
			}
		} else if p != nil {
			t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, p, err, err)
		} else {
			t.Logf("(error) %s: %v", call, err)
		}
	}
}

// fbTestViews creates image views for framebuffer testing.
func fbTestViews(width, height, layers int) (iv []driver.ImageView, free func(), err error) {
	size := driver.Dim3D{Width: width, Height: height, Depth: 1}
	im := make([]driver.Image, len(tAttach))
	iv = make([]driver.ImageView, len(tAttach))
	for i, a := range tAttach {
		im[i], err = tDrv.NewImage(a.Format, size, layers, 1, a.Samples, driver.URenderTarget)
		if err != nil {
			break
		}
		var typ driver.ViewType
		switch layers {
		case 1:
			switch a.Samples {
			case 1:
				typ = driver.IView2D
			default:
				typ = driver.IView2DMS
			}
		default:
			switch a.Samples {
			case 1:
				typ = driver.IView2DArray
			default:
				typ = driver.IView2DMSArray
			}
		}
		iv[i], err = im[i].NewView(typ, 0, layers, 0, 1)
		if err != nil {
			break
		}
	}
	free = func() {
		for i := range iv {
			if iv[i] != nil {
				iv[i].Destroy()
			}
			if im[i] != nil {
				im[i].Destroy()
			} else {
				break
			}
		}
	}
	return
}

func TestFramebuf(t *testing.T) {
	ps := make([]driver.RenderPass, len(tRP))
	for i, p := range tRP {
		ps[i], _ = tDrv.NewRenderPass(p.att, p.sub)
	}
	cases := [...]struct {
		width  int
		height int
		layers int
	}{
		{768, 480, 1},
		{600, 800, 1},
		{512, 512, 2},
		{1920, 1080, 1},
	}
	zf := framebuf{}
	for _, c := range cases {
		iv, free, err := fbTestViews(c.width, c.height, c.layers)
		if err == nil {
			for i, p := range ps {
				if p == nil {
					t.Errorf("skipping nil render pass (%v)...", p)
					continue
				}
				call := fmt.Sprintf("ps[%d].NewFB(%v, %d, %d, %d)", i, iv, c.width, c.height, c.layers)
				// NewFB.
				if f, err := p.NewFB(iv, c.width, c.height, c.layers); err == nil {
					if f == nil {
						t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
						continue
					}
					f := f.(*framebuf)
					if f.p != p {
						t.Errorf("%s: f.p\nhave %p\nwant %p", call, f.p, p)
					}
					if f.fb == zf.fb {
						t.Errorf("%s: f.fb\nhave %v\nwant valid handle", call, f.fb)
					}
					// Destroy.
					f.Destroy()
					if *f != zf {
						t.Errorf("f.Destroy(): f\nhave %v\nwant %v", *f, zf)
					}
				} else if f != nil {
					t.Errorf("%s\nhave %v, %v\nwant nil, %v", call, f, err, err)
				} else {
					t.Logf("(error) %s: %v", call, err)
				}
			}
		} else {
			t.Errorf("fbTestViews(%d, %d, %d) failed, cannot call NewFB", c.width, c.height, c.layers)
		}
		free()
	}
	for _, p := range ps {
		if p != nil {
			p.Destroy()
		}
	}
}
