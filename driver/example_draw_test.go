// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
	"unsafe"

	"gviegas/neo3/driver"
)

// Example_draw renders a triangle and writes the
// result to a file.
func Example_draw() {
	// Create an image resource and a 2D image view
	// to use as render target.
	pf := driver.RGBA8un
	dim := driver.Dim3D{
		Width:  256,
		Height: 256,
	}
	img, err := gpu.NewImage(pf, dim, 1, 1, 1, driver.UCopySrc|driver.URenderTarget)
	if err != nil {
		log.Fatal(err)
	}
	defer img.Destroy()
	view, err := img.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer view.Destroy()

	// Define the render target for drawing into.
	// To draw the triangle, a single color target
	// suffices. The contents are stored at the
	// end of the render pass, as we want to copy
	// them to CPU memory afterwards.
	rt := driver.ColorTarget{
		Color:   view,
		Resolve: nil,
		Load:    driver.LClear,
		Store:   driver.SStore,
		Clear:   driver.ClearFloat32(1, 1, 1, 1),
	}

	// Create a GPU private buffer to store vertex
	// and constant data.
	const bsz = (triPosSize+triColSize+255)&^255 + 256
	buf, err := gpu.NewBuffer(bsz, false, driver.UCopyDst|driver.UVertexData|driver.UShaderConst)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Destroy()

	// We will use staging buffers to copy data
	// from/to GPU private memory. This is required
	// for images and generally more performant for
	// buffers.
	rdbk, err := gpu.NewBuffer(int64(dim.Width*dim.Height*pf.Size()), true, driver.UCopyDst)
	if err != nil {
		log.Fatal(err)
	}
	defer rdbk.Destroy()
	upld, err := gpu.NewBuffer(bsz, true, driver.UCopySrc)
	if err != nil {
		log.Fatal(err)
	}
	defer upld.Destroy()

	p := upld.Bytes()
	copy(p, unsafe.Slice((*byte)(unsafe.Pointer(&triPos[0])), triPosSize))
	copy(p[triPosSize:], unsafe.Slice((*byte)(unsafe.Pointer(&triCol[0])), triColSize))
	copy(p[bsz-256:], unsafe.Slice((*byte)(unsafe.Pointer(&triM[0])), triMSize))

	// Get the shaders.
	var shd [2]struct {
		fileName, funcName string
	}
	switch name := drv.Name(); {
	case strings.Contains(strings.ToLower(name), "vulkan"):
		shd[0].fileName = "triangle_vs.spv"
		shd[0].funcName = "main"
		shd[1].fileName = "triangle_fs.spv"
		shd[1].funcName = "main"
	default:
		log.Fatalf("no shaders for %s driver", name)
	}
	var bb bytes.Buffer
	var offc [2]int
	for i := range shd {
		file, err := os.Open("testdata/" + shd[i].fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		offc[i] = bb.Len()
		if _, err = bb.ReadFrom(file); err != nil {
			log.Fatal(err)
		}
	}
	scode := [2][]byte{
		bb.Bytes()[offc[0]:offc[1]],
		bb.Bytes()[offc[1]:],
	}

	// Define descriptors, create a descriptor heap
	// and a descriptor table.
	dconst := driver.Descriptor{
		Type:   driver.DConstant,
		Stages: driver.SVertex,
		Nr:     0,
		Len:    1,
	}
	dheap, err := gpu.NewDescHeap([]driver.Descriptor{dconst})
	if err != nil {
		log.Fatal(err)
	}
	defer dheap.Destroy()
	dtab, err := gpu.NewDescTable([]driver.DescHeap{dheap})
	if err != nil {
		log.Fatal(err)
	}
	defer dtab.Destroy()

	// One copy of the descriptor heap is enough,
	// as we will draw just once.
	if err := dheap.New(1); err != nil {
		log.Fatal(err)
	}
	dheap.SetBuffer(0, 0, 0, []driver.Buffer{buf}, []int64{bsz - 256}, []int64{triMSize})

	// Create a graphics pipeline.
	pl, err := gpu.NewPipeline(&driver.GraphState{
		VertFunc: driver.ShaderFunc{
			Code: scode[0],
			Name: shd[0].funcName,
		},
		FragFunc: driver.ShaderFunc{
			Code: scode[1],
			Name: shd[1].funcName,
		},
		Desc: dtab,
		Input: []driver.VertexIn{
			{
				Format: driver.Float32x3,
				Stride: 4 * 3,
				Nr:     0,
			},
			{
				Format: driver.Float32x4,
				Stride: 4 * 4,
				Nr:     1,
			},
		},
		Topology: driver.TTriangle,
		Raster: driver.RasterState{
			Discard:   false,
			Clockwise: false,
			Cull:      driver.CBack,
			Fill:      driver.FFill,
			DepthBias: false,
		},
		Samples: 1,
		DS: driver.DSState{
			DepthTest:   false,
			DepthWrite:  false,
			StencilTest: false,
		},
		Blend: driver.BlendState{
			IndependentBlend: false,
			Color: []driver.ColorBlend{{
				Blend:     true,
				WriteMask: driver.CAll,
				SrcFacRGB: driver.BBlendColor,
				DstFacRGB: driver.BDstColor,
				OpRGB:     driver.BRevSubtract,
				SrcFacA:   driver.BOne,
				DstFacA:   driver.BZero,
				OpA:       driver.BAdd,
			}},
		},
		ColorFmt: []driver.PixelFmt{pf},
		DSFmt:    driver.FInvalid,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pl.Destroy()

	// Create a command buffer and record commands.
	// First we upload the vertices and constants
	// from CPU visible to GPU private memory.
	// We then record a render pass that draws the
	// triangle. Finally, we copy the rendered image
	// into the readback buffer.
	cb, err := gpu.NewCmdBuffer()
	if err != nil {
		log.Fatal(err)
	}
	defer cb.Destroy()

	// Begin must be called before recording any
	// commands in the command buffer.
	if err = cb.Begin(); err != nil {
		log.Fatal(err)
	}

	cb.CopyBuffer(&driver.BufferCopy{
		From:    upld,
		FromOff: 0,
		To:      buf,
		ToOff:   0,
		Size:    bsz,
	})

	// This barrier ensures that the following
	// render pass waits the above copy's completion
	// before accessing vertex/constant data.
	cb.Barrier([]driver.Barrier{{
		SyncBefore:   driver.SCopy,
		SyncAfter:    driver.SVertexInput | driver.SVertexShading,
		AccessBefore: driver.ACopyWrite,
		AccessAfter:  driver.AVertexBufRead | driver.AShaderRead,
	}})

	// Put the render target in the correct layout.
	cb.Transition([]driver.Transition{{
		Barrier: driver.Barrier{
			SyncBefore:   driver.SNone,
			SyncAfter:    driver.SColorOutput,
			AccessBefore: driver.ANone,
			AccessAfter:  driver.AColorWrite,
		},
		LayoutBefore: driver.LUndefined,
		LayoutAfter:  driver.LColorTarget,
		Img:          img,
		Layer:        0,
		Layers:       1,
		Level:        0,
		Levels:       1,
	}})

	// Record the render pass.
	cb.BeginPass(dim.Width, dim.Height, 1, []driver.ColorTarget{rt}, nil)
	cb.SetPipeline(pl)
	cb.SetViewport(driver.Viewport{
		X:      0,
		Y:      0,
		Width:  float32(dim.Width),
		Height: float32(dim.Height),
		Znear:  0,
		Zfar:   1,
	})
	cb.SetScissor(driver.Scissor{
		X:      0,
		Y:      0,
		Width:  dim.Width,
		Height: dim.Height,
	})
	cb.SetBlendColor(0.25, 0.5, 0.75, 0)
	cb.SetVertexBuf(0, []driver.Buffer{buf, buf}, []int64{0, triPosSize})
	cb.SetDescTableGraph(dtab, 0, []int{0})
	cb.Draw(3, 1, 0, 0)
	cb.EndPass()

	// Stall the copy below (and the layout change)
	// until the previous render pass finishes
	// writting to the render target.
	cb.Transition([]driver.Transition{{
		Barrier: driver.Barrier{
			SyncBefore:   driver.SColorOutput,
			SyncAfter:    driver.SCopy,
			AccessBefore: driver.AColorWrite,
			AccessAfter:  driver.ACopyRead | driver.ACopyWrite,
		},
		LayoutBefore: driver.LColorTarget,
		LayoutAfter:  driver.LCopySrc,
		Img:          img,
		Layer:        0,
		Layers:       1,
		Level:        0,
		Levels:       1,
	}})

	cb.CopyImgToBuf(&driver.BufImgCopy{
		Buf:     rdbk,
		BufOff:  0,
		RowStrd: dim.Width,
		SlcStrd: dim.Height,
		Img:     img,
		ImgOff:  driver.Off3D{},
		Layer:   0,
		Level:   0,
		Size:    dim,
		Layers:  1,
	})

	// End must be called before committing the
	// command buffer to the GPU.
	// Recording into a command buffer that was
	// ended and not committed/reset is an error.
	if err = cb.End(); err != nil {
		log.Fatal(err)
	}

	// Commit the command buffer.
	// When commit finishes executing the commands,
	// it sends the driver.WorkItem back on the
	// provided channel - only then the command
	// buffers can receive new recordings.
	wk := driver.WorkItem{Work: []driver.CmdBuffer{cb}}
	ch := make(chan *driver.WorkItem)
	if err = gpu.Commit(&wk, ch); err != nil {
		log.Fatal(err)
	}
	if err = (<-ch).Err; err != nil {
		log.Fatal(err)
	}

	// Write the result to a file.
	// Since the image uses a 8 bits per channel
	// RGBA format and the data in the staging
	// buffer is tightly packed, we can just copy
	// the buffer contents directly.
	nrgba := image.NewNRGBA(image.Rect(0, 0, dim.Width, dim.Height))
	copy(nrgba.Pix, rdbk.Bytes())
	file, err := os.Create("testdata/triangle.png")
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

var (
	// Vertex positions (CCW).
	triPos = [9]float32{
		-1, 1, 0,
		1, 1, 0,
		0, -1, 0,
	}
	// Vertex colors.
	triCol = [12]float32{
		0, 1, 1, 1,
		1, 0, 1, 1,
		1, 1, 0, 1,
	}
	// Transform.
	triM = [16]float32{
		0.7, 0, 0, 0,
		0, 0.7, 0, 0,
		0, 0, 0.7, 0,
		0, 0, 0, 1,
	}
)

const (
	triPosSize = int64(unsafe.Sizeof(triPos))
	triColSize = int64(unsafe.Sizeof(triCol))
	triMSize   = int64(unsafe.Sizeof(triM))
)
