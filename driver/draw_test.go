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
	_ "gviegas/neo3/driver/vk"
)

var (
	drv driver.Driver
	gpu driver.GPU
)

func init() {
	// Select a driver to use.
	drivers := driver.Drivers()
drvLoop:
	for i := range drivers {
		switch drivers[i].Name() {
		case "vulkan":
			drv = drivers[i]
			break drvLoop
		}
	}
	if drv == nil {
		log.Fatal("driver.Drivers(): driver not found")
	}
	var err error
	gpu, err = drv.Open()
	if err != nil {
		log.Fatal(err)
	}
	// Ideally, we should call drv.Close somewhere.
}

// Example_draw renders a triangle and outputs the
// result to a file.
func Example_draw() {
	// Create a buffer to store vertex data and constant data for
	// shaders, then copy trianglePos, triangleCol and triangleM
	// to its memory.
	buf, err := gpu.NewBuffer(2<<10, true, driver.UShaderConst|driver.UVertexData)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Destroy()
	p := buf.Bytes()
	npos := unsafe.Sizeof(trianglePos)
	pos := unsafe.Slice((*byte)(unsafe.Pointer(&trianglePos[0])), npos)
	copy(p, pos)
	ncol := unsafe.Sizeof(triangleCol)
	col := unsafe.Slice((*byte)(unsafe.Pointer(&triangleCol[0])), ncol)
	copy(p[npos:], col)
	offm := 1024
	nm := unsafe.Sizeof(triangleM)
	m := unsafe.Slice((*byte)(unsafe.Pointer(&triangleM[0])), nm)
	copy(p[offm:], m)

	// Create an image resource and a 2D image view to use as
	// render target.
	pf := driver.RGBA8un
	psz := 4
	dim := driver.Dim3D{
		Width:  512,
		Height: 512,
		Depth:  1,
	}
	img, err := gpu.NewImage(pf, dim, 1, 1, 1, driver.URenderTarget)
	if err != nil {
		log.Fatal(err)
	}
	defer img.Destroy()
	view, err := img.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer view.Destroy()

	// Define the render target for drawing in a render pass.
	// To draw the triangle, a single color target suffices.
	// The contents are stored at the end of the render pass,
	// so we can copy it to CPU memory later.
	rt := driver.ColorTarget{
		Color:   view,
		Resolve: nil,
		Load:    driver.LClear,
		Store:   driver.SStore,
		Clear:   [4]float32{1, 1, 0, 1},
	}

	// Create vertex and fragment shader binaries.
	// Shaders are platform-specific.
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
	bb := bytes.Buffer{}
	scode := [2]driver.ShaderCode{}
	for i := range scode {
		file, err := os.Open("testdata/" + shd[i].fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		_, err = bb.ReadFrom(file)
		if err != nil {
			log.Fatal(err)
		}
		scode[i], err = gpu.NewShaderCode(bb.Bytes())
		if err != nil {
			log.Fatal(err)
		}
		defer scode[i].Destroy()
		bb.Reset()
	}

	// Define descriptors, create a descriptor heap and
	// a descriptor table.
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
	// Since we are rendering a single instance of the triangle,
	// one copy of the descriptor heap is enough.
	err = dheap.New(1)
	if err != nil {
		log.Fatal(err)
	}
	dheap.SetBuffer(0, 0, 0, []driver.Buffer{buf}, []int64{int64(offm)}, []int64{int64(nm)})

	// Define states and create a graphics pipeline.
	// The bulk of the configuration is done here.
	gs := driver.GraphState{
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
			Color: []driver.ColorBlend{
				{
					Blend:     true,
					WriteMask: driver.CAll,
					Op:        [2]driver.BlendOp{driver.BSubtract, driver.BAdd},
					SrcFac:    [2]driver.BlendFac{driver.BBlendColor, driver.BOne},
					DstFac:    [2]driver.BlendFac{driver.BDstColor, driver.BOne},
				},
			},
		},
		ColorFmt: []driver.PixelFmt{pf},
		DSFmt:    driver.FInvalid,
	}
	pl, err := gpu.NewPipeline(&gs)
	if err != nil {
		log.Fatal(err)
	}
	defer pl.Destroy()

	// Create a second buffer to copy image data into.
	// Image memory is GPU-private, so a staging buffer is required
	// if we are going to access image data from the CPU side.
	cpy, err := gpu.NewBuffer(int64(dim.Width*dim.Height*dim.Depth*psz), true, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer cpy.Destroy()

	// Create a command buffer and record commands.
	// We record a render pass that draws the triangle and
	// a data transfer that copies the results to a buffer
	// accessible from the CPU side.
	// The copy command is set to wait for the render pass
	// to complete before it starts the copy.
	cb, err := gpu.NewCmdBuffer()
	if err != nil {
		log.Fatal(err)
	}
	var (
		vport = driver.Viewport{
			X:      0,
			Y:      0,
			Width:  float32(dim.Width),
			Height: float32(dim.Height),
			Znear:  0,
			Zfar:   1,
		}
		sciss = driver.Scissor{
			X:      0,
			Y:      0,
			Width:  dim.Width,
			Height: dim.Height,
		}
		blit = driver.BufImgCopy{
			Buf:    cpy,
			BufOff: 0,
			// Stride is given in pixels, not bytes.
			Stride: [2]int64{int64(dim.Width), int64(dim.Height)},
			Img:    img,
			ImgOff: driver.Off3D{},
			Layer:  0,
			Level:  0,
			Size:   dim,
		}
		tdraw = [1]driver.Transition{
			{
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
			},
		}
		tcopy = [1]driver.Transition{
			{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SGraphics,
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
			},
		}
	)
	// Begin must be called before recording any commands in
	// the command buffer.
	if err = cb.Begin(); err != nil {
		log.Fatal(err)
	}
	cb.Transition(tdraw[:])
	cb.BeginPass(dim.Width, dim.Height, 1, []driver.ColorTarget{rt}, nil)
	cb.SetPipeline(pl)
	cb.SetViewport([]driver.Viewport{vport})
	cb.SetScissor([]driver.Scissor{sciss})
	cb.SetBlendColor(0, 0, 0.75, 0)
	cb.SetVertexBuf(0, []driver.Buffer{buf, buf}, []int64{0, int64(npos)})
	cb.SetDescTableGraph(dtab, 0, []int{0})
	cb.Draw(3, 1, 0, 0)
	cb.EndPass()
	cb.Transition(tcopy[:])
	cb.CopyImgToBuf(&blit)

	// End must be called before committing the command buffer
	// to the GPU.
	// Recording into a command buffer that was ended and not
	// committed/reset is an error.
	if err = cb.End(); err != nil {
		log.Fatal(err)
	}

	// Commit the command buffer.
	// When Commit completes execution of the commands,
	// it sends to the provided channel. Only then the
	// command buffers can receive new recordings.
	wk := driver.WorkItem{Work: []driver.CmdBuffer{cb}}
	ch := make(chan *driver.WorkItem)
	err = gpu.Commit(&wk, ch)
	if err != nil {
		log.Fatal(err)
	}
	if err := (<-ch).Err; err != nil {
		log.Fatal(err)
	}

	// Write the results to file.
	// Since the image uses a 8-bpc RGBA format and the data in the
	// staging buffer is tightly packed, we can just copy the buffer
	// contents directly.
	nrgba := image.NewNRGBA(image.Rect(0, 0, dim.Width, dim.Height))
	copy(nrgba.Pix, cpy.Bytes())
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

// Vertex positions for the triangle (CCW).
var trianglePos = [9]float32{
	-1.0, +1.0, +0.5,
	+1.0, +1.0, +0.5,
	-0.0, -1.0, +0.5,
}

// Vertex colors for the triangle.
var triangleCol = [12]float32{
	0.0, 0.0, 0.1, 1.0,
	0.0, 0.0, 0.7, 1.0,
	0.0, 0.0, 0.4, 1.0,
}

// Transform for the triangle (column-major).
var triangleM = [16]float32{
	0.8, 0.0, 0.0, 0.0,
	0.0, 0.8, 0.0, 0.0,
	0.0, 0.0, 0.8, 0.0,
	0.0, 0.0, 0.0, 1.0,
}
