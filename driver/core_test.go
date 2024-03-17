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
		t.Errorf("GPU.NewCmdBuffer failed: %#v", err)
		return
	}
	defer cb.Destroy()
	for range 2 {
		if cb.IsRecording() {
			t.Error("CmdBuffer.isRecording:\nhave true\nwant false")
		}
		err := cb.Begin()
		if err != nil {
			t.Errorf("CmdBuffer.Begin failed: %#v", err)
			return
		}
		if !cb.IsRecording() {
			t.Error("CmdBuffer.isRecording:\nhave false\nwant true")
		}
		err = cb.End()
		if err != nil {
			t.Errorf("CmdBuffer.End failed: %#v", err)
			return
		}
		err = cb.Reset()
		if err != nil {
			t.Errorf("CmdBuffer.Reset failed: %#v", err)
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
			t.Errorf("GPU.NewDescHeap failed: %#v\nds: %v\n", err, ds)
			continue
		}
		defer dh.Destroy()
		if n := dh.Len(); n != 0 {
			t.Errorf("DescHeap.Len:\nhave %d\nwant 0", n)
		}
		for _, n := range [...]int{1, 0, 4, 10, 2, 16, 64, 300, 13} {
			err := dh.New(n)
			if err != nil {
				t.Errorf("DescHeap.New failed: %#v", err)
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
			t.Errorf("GPU.NewDescTable failed: %#v\ndh: %v", err, dh)
			continue
		}
		defer dt.Destroy()
		if n, m := len(dh), dt.Len(); n != m {
			t.Errorf("DescTable.Len:\nhave %d\nwant %d", m, n)
		}
		for i := range len(dh) {
			if h, g := dh[i], dt.Heap(i); h != g {
				t.Errorf("DescTable.Heap(%d):\nhave %v\nwant %v", i, g, h)
			}
		}
	}
}
