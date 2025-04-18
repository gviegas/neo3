// Copyright 2024 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"testing"

	"gviegas/neo3/driver"
)

func TestGPUDriver(t *testing.T) {
	g, _ := drv.Open()
	if gpu.Driver() != drv || gpu.Driver() != g.Driver() {
		t.Error("GPU.Driver: unexpected Driver value")
	}
}

func TestCmdBuffer(t *testing.T) {
	cb, err := gpu.NewCmdBuffer()
	if err != nil {
		t.Errorf("GPU.NewCmdBuffer failed: %v", err)
		return
	}
	defer cb.Destroy()
	for range 2 {
		if cb.IsRecording() {
			t.Error("CmdBuffer.IsRecording:\nhave true\nwant false")
		}
		err := cb.Begin()
		if err != nil {
			t.Errorf("CmdBuffer.Begin failed: %v", err)
			return
		}
		if !cb.IsRecording() {
			t.Error("CmdBuffer.IsRecording:\nhave false\nwant true")
		}
		err = cb.End()
		if err != nil {
			t.Errorf("CmdBuffer.End failed: %v", err)
			return
		}
		err = cb.Reset()
		if err != nil {
			t.Errorf("CmdBuffer.Reset failed: %v", err)
			return
		}
	}
}

// tDesc contains lists of descriptors for testing.
var tDesc = [...][]driver.Descriptor{
	{
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SVertex, Nr: 1, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 2, Len: 8},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex, Nr: 3, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 4, Len: 3},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DSampler, Stages: driver.SVertex | driver.SFragment, Nr: 3, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SFragment, Nr: 0, Len: 4},
		{Type: driver.DConstant, Stages: driver.SVertex, Nr: 1, Len: 1},
	},
	{
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DSampler, Stages: driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 3, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex, Nr: 0, Len: 1},
		{Type: driver.DSampler, Stages: driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DTexture, Stages: driver.SFragment, Nr: 1, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
		{Type: driver.DBuffer, Stages: driver.SVertex, Nr: 3, Len: 1},
		{Type: driver.DImage, Stages: driver.SFragment, Nr: 4, Len: 1},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DSampler, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 1},
	},
	{
		{Type: driver.DConstant, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 1},
		{Type: driver.DBuffer, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DImage, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 1},
		{Type: driver.DImage, Stages: driver.SFragment, Nr: 3, Len: 1},
		{Type: driver.DImage, Stages: driver.SVertex, Nr: 4, Len: 1},
		{Type: driver.DConstant, Stages: driver.SFragment, Nr: 5, Len: 1},
	},
	{
		{Type: driver.DSampler, Stages: driver.SVertex | driver.SFragment, Nr: 0, Len: 12},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 1, Len: 1},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 2, Len: 4},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 3, Len: 1},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 4, Len: 4},
		{Type: driver.DTexture, Stages: driver.SVertex | driver.SFragment, Nr: 5, Len: 2},
	},
}

func TestDescHeap(t *testing.T) {
	for _, ds := range tDesc {
		dh, err := gpu.NewDescHeap(ds)
		if err != nil {
			t.Errorf("GPU.NewDescHeap failed: %v\nds: %v\n", err, ds)
			continue
		}
		defer dh.Destroy()
		if n := dh.Len(); n != 0 {
			t.Errorf("DescHeap.Len:\nhave %d\nwant 0", n)
		}
		for _, n := range [...]int{1, 0, 4, 10, 2, 16, 64, 300, 13} {
			err := dh.New(n)
			if err != nil {
				t.Errorf("DescHeap.New failed: %v", err)
				continue
			}
			if m := dh.Len(); m != n {
				t.Errorf("DescHeap.Len:\nhave %d\nwant %d", m, n)
			}
		}
	}
}

func TestDescTable(t *testing.T) {
	dh := make([]driver.DescHeap, len(tDesc))
	defer func() {
		for _, h := range dh {
			if h != nil {
				h.Destroy()
			}
		}
	}()
	hs := make([][]driver.DescHeap, len(dh))
	for i, ds := range tDesc {
		h, err := gpu.NewDescHeap(ds)
		if err != nil {
			t.Errorf("GPU.NewDescHeap failed - cannot test GPU.NewDescTable")
			return
		}
		dh[i] = h
		hs[i] = []driver.DescHeap{h}
	}
	hs = append(hs,
		[]driver.DescHeap{dh[0], dh[2]},
		[]driver.DescHeap{dh[0], dh[3]},
		[]driver.DescHeap{dh[3], dh[4]},
		[]driver.DescHeap{dh[0], dh[1], dh[2]},
		[]driver.DescHeap{dh[1], dh[2], dh[3], dh[4]},
		[]driver.DescHeap{dh[5], dh[0]},
		[]driver.DescHeap{dh[5], dh[3]},
		[]driver.DescHeap{dh[6], dh[1]},
		[]driver.DescHeap{dh[6], dh[4]},
		[]driver.DescHeap{dh[6], dh[0], dh[1]},
		[]driver.DescHeap{dh[7], dh[6]},
		[]driver.DescHeap{dh[8], dh[0], dh[4]},
		[]driver.DescHeap{dh[9], dh[3], dh[4]},
		// Sets have separate namespaces, so these
		// should not clash.
		[]driver.DescHeap{dh[10], dh[1]},
		[]driver.DescHeap{dh[10], dh[1], dh[2], dh[3]},
		[]driver.DescHeap{dh[11], dh[10]},
		[]driver.DescHeap{dh[12], dh[4], dh[1], dh[0]},
	)
	for _, dh := range hs {
		dt, err := gpu.NewDescTable(dh)
		if err != nil {
			t.Errorf("GPU.NewDescTable failed: %v\ndh: %v", err, dh)
			continue
		}
		defer dt.Destroy()
		if n, m := len(dh), dt.Len(); n != m {
			t.Errorf("DescTable.Len:\nhave %d\nwant %d", m, n)
		}
		for i := range dh {
			if h, g := dh[i], dt.Heap(i); h != g {
				t.Errorf("DescTable.Heap(%d):\nhave %v\nwant %v", i, g, h)
			}
		}
	}
}

func TestBuffer(t *testing.T) {
	cases := [...]struct {
		size    int64
		visible bool
		usage   driver.Usage
	}{
		{2048, true, driver.UCopySrc},
		{4096, true, driver.UCopyDst},
		{8192, true, driver.UShaderRead | driver.UShaderWrite | driver.UShaderConst | driver.UVertexData | driver.UIndexData},
		//{512, true, 0},
		{16, true, driver.UShaderRead | driver.UShaderWrite},
		{1 << 20, false, driver.UGeneric},
		{1 << 20, false, driver.UShaderConst | driver.UVertexData | driver.UIndexData},
		{1 << 20, true, driver.UVertexData | driver.UIndexData},
		{100 << 20, true, driver.UGeneric},
		{1, true, driver.UGeneric},
		//{0, true, 0},
	}
	for _, c := range cases {
		buf, err := gpu.NewBuffer(c.size, c.visible, c.usage)
		if err != nil {
			t.Errorf("GPU.NewBuffer failed: %v", err)
			continue
		}
		defer buf.Destroy()
		if c.visible && !buf.Visible() {
			// Private memory is optional, shared memory is not.
			t.Errorf("Buffer.Visible:\nhave false\nwant true")
		}
		switch b := buf.Bytes(); {
		case buf.Visible():
			if len := len(b); int64(len) < c.size {
				t.Errorf("len(Buffer.Bytes):\nhave %d\nwant >= %d", len, c.size)
			}
		default:
			if b != nil {
				t.Error("Buffer.Bytes:\nhave non-nil\nwant nil")
			}
		}
		if cap := buf.Cap(); cap < c.size {
			t.Errorf("Buffer.Cap:\nhave %d\nwant >= %d", cap, c.size)
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
		{driver.RGBA8Unorm, driver.Dim3D{Width: 1024, Height: 1024}, 1, 11, 1, driver.UShaderSample, []iview{
			{driver.IView2D, 0, 1, 0, 1},
			{driver.IView2D, 0, 1, 0, 11},
			{driver.IView2D, 0, 1, 4, 5},
		}},
		{driver.RGBA16Float, driver.Dim3D{Width: 1024, Height: 1024}, 1, 6, 1, driver.UGeneric, []iview{
			{driver.IView2D, 0, 1, 0, 1},
			{driver.IView2D, 0, 1, 2, 1},
			{driver.IView2D, 0, 1, 3, 3},
		}},
		{driver.BGRA8SRGB, driver.Dim3D{Width: 1280, Height: 768}, 1, 1, 8, driver.URenderTarget, []iview{
			{driver.IView2DMS, 0, 1, 0, 1},
			{driver.IView2D, 0, 1, 0, 1},
		}},
		{driver.D24UnormS8Uint, driver.Dim3D{Width: 1280, Height: 768}, 2, 1, 1, driver.URenderTarget, []iview{
			{driver.IView2D, 0, 1, 0, 1},
			{driver.IView2DArray, 0, 2, 0, 1},
		}},
		{driver.D16Unorm, driver.Dim3D{Width: 1280, Height: 768}, 2, 1, 1, driver.URenderTarget, []iview{
			{driver.IView2D, 0, 1, 0, 1},
			{driver.IView2D, 1, 1, 0, 1},
		}},
		{driver.S8Uint, driver.Dim3D{Width: 1280, Height: 768}, 3, 1, 1, driver.URenderTarget, []iview{
			{driver.IView2D, 2, 1, 0, 1},
			{driver.IView2DArray, 0, 3, 0, 1},
		}},
		{driver.R8Unorm, driver.Dim3D{Width: 4096}, 4, 1, 1, driver.UGeneric, []iview{
			{driver.IView1D, 0, 1, 0, 1},
			{driver.IView1D, 3, 1, 0, 1},
			{driver.IView1DArray, 0, 4, 0, 1},
		}},
		{driver.RG16Float, driver.Dim3D{Width: 480, Height: 720, Depth: 5}, 1, 1, 1, driver.UGeneric, []iview{
			{driver.IView3D, 0, 1, 0, 1},
		}},
		{driver.RGBA8Unorm, driver.Dim3D{Width: 512, Height: 512}, 16, 10, 1, driver.UShaderSample, []iview{
			{driver.IViewCube, 0, 6, 0, 1},
			{driver.IViewCube, 4, 6, 0, 10},
			// TODO: Check cube array feature.
			//{driver.IViewCubeArray, 0, 12, 0, 10},
		}},
	}
	for _, c := range cases {
		img, err := gpu.NewImage(c.pf, c.size, c.layers, c.levels, c.samples, c.usage)
		if err != nil {
			t.Errorf("GPU.NewImage failed: %v", err)
			continue
		}
		defer img.Destroy()
		for _, c := range c.iv {
			iv, err := img.NewView(c.typ, c.layer, c.layers, c.level, c.levels)
			if err != nil {
				t.Errorf("Image.NewView failed: %s", err)
				continue
			}
			defer iv.Destroy()
			if im := iv.Image(); im != img {
				t.Errorf("ImageView.Image:\nhave %v\nwant %v", im, img)
			}
		}
	}
}
