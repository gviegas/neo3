// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"os"
	"strings"

	"gviegas/neo3/driver"
)

// Example_dispatch creates a checker pattern using
// compute and writes the result to a file.
func Example_dispatch() {
	// Each 2D group defines a cell where all pixels
	// have the same color.
	grpCntX := 8
	grpCntY := 9
	invCntX := 10 // From shader code.
	invCntY := 10 // From shader code.
	dim := driver.Dim3D{
		Width:  grpCntX * invCntX,
		Height: grpCntY * invCntY,
	}
	pf := driver.RGBA8Unorm // From shader code.
	sz := int64(dim.Width * dim.Height * pf.Size())

	// Create the storage image/view (write-only).
	stor, err := gpu.NewImage(pf, dim, 1, 1, 1, driver.UCopySrc|driver.UShaderWrite)
	if err != nil {
		log.Fatal(err)
	}
	defer stor.Destroy()
	view, err := stor.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer view.Destroy()

	// Create the descriptor heap/table that will
	// contain the storage view.
	dheap, err := gpu.NewDescHeap([]driver.Descriptor{{
		Type:   driver.DImage,
		Stages: driver.SCompute,
		Nr:     0,
		Len:    1,
	}})
	if err != nil {
		log.Fatal(err)
	}
	defer dheap.Destroy()
	dtab, err := gpu.NewDescTable([]driver.DescHeap{dheap})
	if err != nil {
		log.Fatal(err)
	}
	defer dtab.Destroy()

	// Set the storage descriptor.
	if err := dheap.New(1); err != nil {
		log.Fatal(err)
	}
	dheap.SetImage(0, 0, 0, []driver.ImageView{view}, nil)

	// Get the compute shader.
	// This shader has (invCntX, invCntY, 1) invocations
	// per work group, which determines the cell size.
	// Each invocation stores either white or black to
	// the image, based on its group ID.
	var shd struct {
		fileName, funcName string
	}
	switch name := drv.Name(); {
	case strings.Contains(strings.ToLower(name), "vulkan"):
		shd.fileName = "checker_cs.spv"
		shd.funcName = "main"
	default:
		log.Fatalf("no shader for %s driver", name)
	}
	file, err := os.Open("testdata/" + shd.fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var b bytes.Buffer
	if _, err := b.ReadFrom(file); err != nil {
		log.Fatal(err)
	}
	cs := b.Bytes()

	// Create the compute pipeline.
	pl, err := gpu.NewPipeline(&driver.CompState{
		Func: driver.ShaderFunc{cs, shd.funcName},
		Desc: dtab,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pl.Destroy()

	// Create a staging buffer to copy the result
	// into, so it can be accessed by the CPU.
	stg, err := gpu.NewBuffer(sz, true, driver.UCopyDst)
	if err != nil {
		log.Fatal(err)
	}
	defer stg.Destroy()

	// We will record the dispatch and the copy in
	// separate command buffers.
	cbDisp, err := gpu.NewCmdBuffer()
	if err != nil {
		log.Fatal(err)
	}
	defer cbDisp.Destroy()
	cbCopy, err := gpu.NewCmdBuffer()
	if err != nil {
		log.Fatal(err)
	}
	defer cbCopy.Destroy()

	ch := make(chan error, 2)

	// Dispatch.
	go func() {
		if err := cbDisp.Begin(); err != nil {
			ch <- err
			return
		}
		cbDisp.SetDescTableComp(dtab, 0, []int{0})
		cbDisp.SetPipeline(pl)
		cbDisp.Transition([]driver.Transition{{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SNone,
				SyncAfter:    driver.SComputeShading,
				AccessBefore: driver.ANone,
				AccessAfter:  driver.AShaderWrite,
			},
			LayoutBefore: driver.LUndefined,
			LayoutAfter:  driver.LShaderStore,
			Img:          stor,
			Layers:       1,
			Levels:       1,
		}})
		cbDisp.Dispatch(grpCntX, grpCntY, 1)
		if err := cbDisp.End(); err != nil {
			ch <- err
			return
		}
		ch <- nil
	}()

	// Copy.
	go func() {
		if err := cbCopy.Begin(); err != nil {
			log.Fatal(err)
		}
		cbCopy.Transition([]driver.Transition{{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SComputeShading,
				SyncAfter:    driver.SCopy,
				AccessBefore: driver.AShaderWrite,
				AccessAfter:  driver.ACopyRead,
			},
			LayoutBefore: driver.LShaderStore,
			LayoutAfter:  driver.LCopySrc,
			Img:          stor,
			Layers:       1,
			Levels:       1,
		}})
		cbCopy.CopyImgToBuf(&driver.BufImgCopy{
			Buf:     stg,
			RowStrd: dim.Width,
			SlcStrd: dim.Height,
			Img:     stor,
			Size:    dim,
			Layers:  1,
		})
		if err := cbCopy.End(); err != nil {
			ch <- err
			return
		}
		ch <- nil
	}()

	// Commit the command buffers.
	for range cap(ch) {
		if err := <-ch; err != nil {
			log.Fatal(err)
		}
	}
	// The order here matters.
	wk := &driver.WorkItem{Work: []driver.CmdBuffer{cbDisp, cbCopy}}
	wch := make(chan *driver.WorkItem)
	if err := gpu.Commit(wk, wch); err != nil {
		log.Fatal(err)
	}
	wk = <-wch
	if wk.Err != nil {
		log.Fatal(wk.Err)
	}

	// Write the result to file.
	nrgba := image.NewNRGBA(image.Rect(0, 0, dim.Width, dim.Height))
	copy(nrgba.Pix, stg.Bytes())
	file, err = os.Create("testdata/checker.png")
	if err != nil {
		log.Fatal(err)
	}
	err = png.Encode(file, nrgba)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	// Output:
}
