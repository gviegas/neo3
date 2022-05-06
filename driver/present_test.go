// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"bytes"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/gviegas/scene/driver"
	_ "github.com/gviegas/scene/driver/vk"
	"github.com/gviegas/scene/wsi"
)

const FPS = 60

// Example_present renders a triangle and presents the
// result in a window.
func Example_present() {
	// Obtain a driver.GPU.
	gpu := GPU()
	defer drv.Close()

	// GPUs that can present implement the Presenter interface.
	presenter, ok := gpu.(driver.Presenter)
	if !ok {
		log.Fatal("GPU cannot present")
	}

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

	// Create a swapchain from a wsi.Window.
	// The swapchain will provide the back buffers to use
	// as attachment in the render pass.
	dim := driver.Dim3D{
		Width:  512,
		Height: 512,
		Depth:  1,
	}
	win, err := wsi.NewWindow(dim.Width, dim.Height, "Presentation Example")
	if err != nil {
		log.Fatal(err)
	}
	defer win.Close()
	win.Map()
	sc, err := presenter.NewSwapchain(win, 2)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Destroy()
	pf := sc.Format()

	// Create a render pass and framebuffer for drawing.
	// To draw the triangle, a single color attachment and
	// subpass suffices.
	// The contents are stored at the end of the subpass so
	// we can present later.
	att := driver.Attachment{
		Format:  pf,
		Samples: 1,
		Load:    [2]driver.LoadOp{driver.LClear},
		Store:   [2]driver.StoreOp{driver.SStore},
	}
	subp := driver.Subpass{
		Color: []int{0},
		DS:    -1,
		MSR:   nil,
		Wait:  false,
	}
	pass, err := gpu.NewRenderPass([]driver.Attachment{att}, []driver.Subpass{subp})
	if err != nil {
		log.Fatal(err)
	}
	defer pass.Destroy()
	// Create all framebuffers in advance.
	var scFB []driver.Framebuf
	for _, iv := range sc.Images() {
		fb, err := pass.NewFB([]driver.ImageView{iv}, dim.Width, dim.Height, 1)
		if err != nil {
			log.Fatal(err)
		}
		defer fb.Destroy()
		scFB = append(scFB, fb)
	}

	// Create vertex and fragment shader binaries.
	// These are simple pass-through shaders.
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
				Name:   "POSITION",
			},
			{
				Format: driver.Float32x4,
				Stride: 4 * 4,
				Nr:     1,
				Name:   "COLOR",
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
					Op:        [2]driver.BlendOp{driver.BAdd, driver.BAdd},
					SrcFac:    [2]driver.BlendFac{driver.BSrcColor, driver.BOne},
					DstFac:    [2]driver.BlendFac{driver.BBlendColor, driver.BOne},
				},
			},
		},
		Pass:    pass,
		Subpass: 0,
	}
	pl, err := gpu.NewPipeline(&gs)
	if err != nil {
		log.Fatal(err)
	}
	defer pl.Destroy()

	// Create a command buffer and record commands.
	//
	// We record a single top-level command:
	// 	1. a render pass command that draws the triangle.
	//
	// Calling the swapchain's Next and Present methods at
	// the right time will ensure correct synchronization.
	cb, err := gpu.NewCmdBuffer()
	if err != nil {
		log.Fatal(err)
	}
	cache := driver.CmdCache{}
	cache.Pass = []driver.PassCmd{
		{
			Pass: pass,
			FB:   nil, // Will be set every frame.
			Clear: []driver.ClearValue{
				{
					Color: [4]float32{1.0, 1.0, 1.0, 1.0},
				},
			},
			SubCmd: []int{0},
		},
	}
	cache.Subpass = []driver.SubpassCmd{
		{
			[]driver.Command{
				{
					Type:  driver.CPipeline,
					Index: 0,
				},
				{
					Type:  driver.CViewport,
					Index: 0,
				},
				{
					Type:  driver.CScissor,
					Index: 0,
				},
				{
					Type:  driver.CBlendColor,
					Index: 0,
				},
				{
					Type:  driver.CVertexBuf,
					Index: 0,
				},
				{
					Type:  driver.CDescTable,
					Index: 0,
				},
				{
					Type:  driver.CDraw,
					Index: 0,
				},
			},
		},
	}
	cache.Pipeline = []driver.PipelineCmd{
		{
			PL: pl,
		},
	}
	cache.Viewport = []driver.ViewportCmd{
		{
			VP: []driver.Viewport{
				{
					X:      0,
					Y:      0,
					Width:  float32(dim.Width),
					Height: float32(dim.Height),
					Znear:  0,
					Zfar:   1,
				},
			},
		},
	}
	cache.Scissor = []driver.ScissorCmd{
		{
			S: []driver.Scissor{
				{
					X:      0,
					Y:      0,
					Width:  dim.Width,
					Height: dim.Height,
				},
			},
		},
	}
	cache.BlendColor = []driver.BlendColorCmd{
		{
			R: 0.75,
			G: 0,
			B: 0,
			A: 0,
		},
	}
	cache.VertexBuf = []driver.VertexBufCmd{
		{
			Start: 0,
			Buf:   []driver.Buffer{buf, buf},
			Off:   []int64{0, int64(npos)},
		},
	}
	cache.DescTable = []driver.DescTableCmd{
		{
			Desc:  dtab,
			Start: 0,
			Copy:  []int{0},
		},
	}
	cache.Draw = []driver.DrawCmd{
		{
			VertCount: 3,
			InstCount: 1,
			BaseVert:  0,
			BaseInst:  0,
		},
	}
	// Reuse the same command data for every frame.
	// We only need to update the framebuffer that
	// will be used as render target.
	cmd := []driver.Command{
		{
			Type:  driver.CPass,
			Index: 0,
		},
	}

	// Start the render loop.
	// We will render for 3 seconds.
	dur := time.Second * 3
	start := time.Now()
	sample := make([]time.Duration, 0, 100)
	ch := make(chan error)
	for tm := start; tm.Sub(start) < dur; tm = time.Now() {
		// Obtain the index of the next writable image view
		// from the swapchain.
		// Because an image might not be available right away,
		// we keep calling Next until it succeeds.
		var fbIdx int
		for {
			fbIdx, err = sc.Next(cb)
			switch err {
			case nil:
			case driver.ErrNotReady:
				log.Printf("sc.Next(): %v", err)
				continue
			case driver.ErrSwapchain:
				// The swapchain has become unusable. It must be recreated
				// if we want to continue presenting on the same window.
				if err = sc.Recreate(); err != nil {
					log.Fatal(err)
				}
				imgs := sc.Images()
				if pf != sc.Format() || len(scFB) != len(imgs) {
					// We could get around this by creating a new
					// render pass and graphics pipeline.
					log.Fatal("sc.Recreate() changed the swapchain in an unexpected way")
				}
				dim.Width = win.Width()
				dim.Height = win.Height()
				for i := range imgs {
					scFB[i], err = pass.NewFB([]driver.ImageView{imgs[i]}, dim.Width, dim.Height, 1)
					if err != nil {
						log.Fatal(err)
					}
					defer scFB[i].Destroy()
				}
				cache.Viewport[0].VP[0].Width = float32(dim.Width)
				cache.Viewport[0].VP[0].Height = float32(dim.Height)
				cache.Scissor[0].S[0].Width = dim.Width
				cache.Scissor[0].S[0].Height = dim.Height
				continue
			default:
				log.Fatal(err)
			}
			cache.Pass[0].FB = scFB[fbIdx]
			break
		}

		// Record the render pass that will target the swapchain.
		// This must only be done after obtaining the image view.
		err = cb.Record(cmd, &cache)
		if err != nil {
			log.Fatal(err)
		}

		// Enqueue the image view for presentation.
		// This must only be done after all render passes that
		// target the image view have been recorded.
		err = sc.Present(fbIdx, cb)
		if err != nil {
			log.Fatal(err)
		}

		// End must be called before commiting the command buffer
		// to the GPU.
		// Recording into a command buffer that was ended and not
		// commited/reset is an error.
		err = cb.End()
		if err != nil {
			log.Fatal(err)
		}

		// Commit the command buffer.
		// When Commit completes execution of the commands, it sends
		// the result to the provided channel. Only then the command
		// buffers can receive new recordings.
		// Note that since the swapchain's Next and Present work as
		// commands, presentation will only happen after commiting.
		go gpu.Commit([]driver.CmdBuffer{cb}, ch)
		err = <-ch
		if err != nil {
			log.Fatal(err)
		}

		ft := time.Second / FPS
		dt := time.Now().Sub(tm)
		if dt < ft {
			time.Sleep(ft - dt)
		}
		if len(sample) < cap(sample) {
			sample = append(sample, time.Now().Sub(tm))
		}
		wsi.Dispatch()
	}

	acc := time.Duration(0)
	for _, s := range sample {
		acc += time.Second / s
	}
	log.Printf("FPS: %d", int(acc)/len(sample))

	// Output:
}
