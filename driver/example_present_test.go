// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"bytes"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"strings"
	"time"
	"unsafe"

	"gviegas/neo3/driver"
	"gviegas/neo3/linear"
	"gviegas/neo3/wsi"
)

const (
	NFrame   = 3
	Samples  = 4
	DepthFmt = driver.D16Unorm
)

type T struct {
	cb       [NFrame]driver.CmdBuffer
	ch       chan *driver.WorkItem
	win      wsi.Window
	sc       driver.Swapchain
	dim      driver.Dim3D
	rt       []driver.ColorTarget
	rtImg    driver.Image
	rtView   driver.ImageView
	ds       driver.DSTarget
	dsImg    driver.Image
	dsView   driver.ImageView
	vertFunc driver.ShaderFunc
	fragFunc driver.ShaderFunc
	stgBuf   driver.Buffer
	vertBuf  driver.Buffer
	idxBuf   driver.Buffer
	constBuf driver.Buffer
	splImg   driver.Image
	splView  driver.ImageView
	splr     driver.Sampler
	dheap    driver.DescHeap
	dtab     driver.DescTable
	pipeln   driver.Pipeline
	vport    driver.Viewport
	sciss    driver.Scissor
	xform    linear.M4
	angleX   float32
	angleY   float32
	turnX    float32
	turnY    float32
	auto     bool
	broken   bool
	quit     bool
}

// Example_present renders a spinning cube and presents
// the result in a window.
func Example_present() {
	var t T
	var err error
	for i := range t.cb {
		t.cb[i], err = gpu.NewCmdBuffer()
		if err != nil {
			log.Fatal(err)
		}
	}
	t.ch = make(chan *driver.WorkItem, NFrame)
	t.swapchainSetup()
	t.passSetup()
	t.shaderSetup()
	t.bufferSetup()
	t.samplingSetup()
	t.descriptorSetup()
	t.pipelineSetup()
	t.vport = driver.Viewport{
		X:      0,
		Y:      0,
		Width:  float32(t.dim.Width),
		Height: float32(t.dim.Height),
		Znear:  0,
		Zfar:   1,
	}
	t.sciss = driver.Scissor{
		X:      0,
		Y:      0,
		Width:  t.dim.Width,
		Height: t.dim.Height,
	}
	wsi.SetWindowHandler(&t)
	wsi.SetKeyboardKeyHandler(&t)
	wsi.SetAppName("driver.example")
	t.renderLoop()
	t.destroy()

	// Output:
}

// swapchainSetup creates the window and swapchain.
func (t *T) swapchainSetup() {
	if wsi.PlatformInUse() == wsi.None {
		log.Fatal("WSI not available")
	}
	win, err := wsi.NewWindow(400, 300, "Present Example")
	if err != nil {
		log.Fatal(err)
	}
	win.Map()

	gpu, ok := gpu.(driver.Presenter)
	if !ok {
		log.Fatal("GPU cannot present")
	}
	sc, err := gpu.NewSwapchain(win, NFrame+1)
	if err != nil {
		log.Fatal(err)
	}

	t.win = win
	t.sc = sc
	t.dim.Width = win.Width()
	t.dim.Height = win.Height()
}

// passSetup creates the color and depth images/views and sets
// the render targets to be used during render passes.
func (t *T) passSetup() {
	rtImg, err := gpu.NewImage(t.sc.Format(), t.dim, 1, 1, Samples, driver.URenderTarget)
	if err != nil {
		log.Fatal(err)
	}
	rtView, err := rtImg.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}
	scViews := t.sc.Views()
	rt := make([]driver.ColorTarget, len(scViews))
	for i := range rt {
		rt[i] = driver.ColorTarget{
			Color:   rtView,
			Resolve: scViews[i],
			Load:    driver.LClear,
			Store:   driver.SDontCare,
			Clear:   driver.ClearFloat32(0.075, 0.075, 0.075, 1),
		}
	}

	dsImg, err := gpu.NewImage(DepthFmt, t.dim, 1, 1, Samples, driver.URenderTarget)
	if err != nil {
		log.Fatal(err)
	}
	dsView, err := dsImg.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}
	ds := driver.DSTarget{
		DS:     dsView,
		LoadD:  driver.LClear,
		StoreD: driver.SDontCare,
		ClearD: 1,
	}

	t.rt = rt
	t.rtImg = rtImg
	t.rtView = rtView
	t.ds = ds
	t.dsImg = dsImg
	t.dsView = dsView
}

// shaderSetup sets the vertex and fragment functions.
func (t *T) shaderSetup() {
	var shd [2]struct {
		fileName, funcName string
	}
	switch name := drv.Name(); {
	case strings.Contains(strings.ToLower(name), "vulkan"):
		shd[0].fileName = "cube_vs.spv"
		shd[0].funcName = "main"
		shd[1].fileName = "cube_fs.spv"
		shd[1].funcName = "main"
	default:
		log.Fatalf("no shaders for %s driver", name)
	}

	var buf bytes.Buffer
	var off [2]int
	for i := range shd {
		file, err := os.Open("testdata/" + shd[i].fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		off[i] = buf.Len()
		if _, err = buf.ReadFrom(file); err != nil {
			log.Fatal(err)
		}
	}
	code := [2][]byte{
		buf.Bytes()[off[0]:off[1]],
		buf.Bytes()[off[1]:],
	}

	t.vertFunc = driver.ShaderFunc{
		Code: code[0],
		Name: shd[0].funcName,
	}
	t.fragFunc = driver.ShaderFunc{
		Code: code[1],
		Name: shd[1].funcName,
	}
}

// bufferSetup creates GPU buffers to store vertex, index and
// constant data.
func (t *T) bufferSetup() {
	const (
		vbSize = cubePosSize + cubeUVSize
		ibSize = cubeIdxSize
		cbSize = int64(256 * NFrame)
		sbSize = max(vbSize+ibSize, cbSize)
	)
	stgBuf, err := gpu.NewBuffer(sbSize, true, driver.UCopySrc)
	if err != nil {
		log.Fatal(err)
	}
	vertBuf, err := gpu.NewBuffer(vbSize, false, driver.UCopyDst|driver.UVertexData)
	if err != nil {
		log.Fatal(err)
	}
	idxBuf, err := gpu.NewBuffer(ibSize, false, driver.UCopyDst|driver.UIndexData)
	if err != nil {
		log.Fatal(err)
	}
	constBuf, err := gpu.NewBuffer(cbSize, false, driver.UCopyDst|driver.UShaderConst)
	if err != nil {
		log.Fatal(err)
	}

	// Since vertex/index data is not going to change,
	// we can copy it upfront.
	stg := stgBuf.Bytes()
	pos := unsafe.Slice((*byte)(unsafe.Pointer(&cubePos[0])), cubePosSize)
	uv := unsafe.Slice((*byte)(unsafe.Pointer(&cubeUV[0])), cubeUVSize)
	idx := unsafe.Slice((*byte)(unsafe.Pointer(&cubeIdx[0])), cubeIdxSize)
	copy(stg, pos)
	copy(stg[cubePosSize:], uv)
	copy(stg[vbSize:], idx)
	if err := t.cb[0].Begin(); err != nil {
		log.Fatal(err)
	}
	t.cb[0].CopyBuffer(&driver.BufferCopy{
		From:    stgBuf,
		FromOff: 0,
		To:      vertBuf,
		ToOff:   0,
		Size:    vbSize,
	})
	t.cb[0].CopyBuffer(&driver.BufferCopy{
		From:    stgBuf,
		FromOff: vbSize,
		To:      idxBuf,
		ToOff:   0,
		Size:    ibSize,
	})
	if err := t.cb[0].End(); err != nil {
		log.Fatal(err)
	}
	wk := driver.WorkItem{Work: []driver.CmdBuffer{t.cb[0]}}
	ch := make(chan *driver.WorkItem, 1)
	if err := gpu.Commit(&wk, ch); err != nil {
		log.Fatal(err)
	}
	if err := (<-ch).Err; err != nil {
		log.Fatal(err)
	}

	t.stgBuf = stgBuf
	t.vertBuf = vertBuf
	t.idxBuf = idxBuf
	t.constBuf = constBuf
}

// samplingSetup creates the sampler and the texture to
// sample from.
func (t *T) samplingSetup() {
	reader, err := os.Open("testdata/feral.png")
	if err != nil {
		log.Fatal(err)
	}
	decImg, err := png.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	var pix []uint8
	switch m := decImg.(type) {
	case *image.NRGBA:
		pix = m.Pix[:]
	case *image.RGBA:
		pix = m.Pix[:]
	default:
		log.Fatal("decoded image is neither NRGBA nor RGBA")
	}
	buf, err := gpu.NewBuffer(int64(len(pix)), true, driver.UCopySrc)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Destroy()
	copy(buf.Bytes(), pix)

	size := driver.Dim3D{
		Width:  decImg.Bounds().Max.X,
		Height: decImg.Bounds().Max.Y,
	}
	img, err := gpu.NewImage(driver.RGBA8SRGB, size, 1, 1, 1, driver.UCopyDst|driver.UShaderSample)
	if err != nil {
		log.Fatal(err)
	}
	view, err := img.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}

	// Images are always GPU private. We need to use a
	// staging buffer to copy data to an image.
	if err = t.cb[0].Begin(); err != nil {
		log.Fatal(err)
	}
	t.cb[0].Transition([]driver.Transition{{
		Barrier: driver.Barrier{
			SyncAfter:   driver.SCopy,
			AccessAfter: driver.ACopyWrite,
		},
		LayoutBefore: driver.LUndefined,
		LayoutAfter:  driver.LCopyDst,
		Img:          img,
		Layers:       1,
		Levels:       1,
	}})
	t.cb[0].CopyBufToImg(&driver.BufImgCopy{
		Buf:     buf,
		RowStrd: size.Width,
		SlcStrd: size.Height,
		Img:     img,
		Size:    size,
		Layers:  1,
	})
	t.cb[0].Transition([]driver.Transition{{
		Barrier: driver.Barrier{
			SyncBefore:   driver.SCopy,
			AccessBefore: driver.ACopyWrite,
		},
		LayoutBefore: driver.LCopyDst,
		LayoutAfter:  driver.LShaderRead,
		Img:          img,
		Layers:       1,
		Levels:       1,
	}})
	if err := t.cb[0].End(); err != nil {
		log.Fatal(err)
	}
	wk := driver.WorkItem{Work: []driver.CmdBuffer{t.cb[0]}}
	ch := make(chan *driver.WorkItem, 1)
	if err := gpu.Commit(&wk, ch); err != nil {
		log.Fatal(err)
	}
	if err := (<-ch).Err; err != nil {
		log.Fatal(err)
	}

	splr, err := gpu.NewSampler(&driver.Sampling{
		Min:      driver.FLinear,
		Mag:      driver.FLinear,
		Mipmap:   driver.FNoMipmap,
		AddrU:    driver.AWrap,
		AddrV:    driver.AWrap,
		AddrW:    driver.AWrap,
		MaxAniso: 1,
		DoCmp:    false,
		Cmp:      driver.CNever,
		MinLOD:   0,
		MaxLOD:   0,
	})
	if err != nil {
		log.Fatal(err)
	}

	t.splImg = img
	t.splView = view
	t.splr = splr
}

// descriptorSetup creates the descriptor heap and
// descriptor table.
func (t *T) descriptorSetup() {
	desc := []driver.Descriptor{
		{
			Type:   driver.DConstant,
			Stages: driver.SVertex,
			Nr:     0,
			Len:    1,
		},
		{
			Type:   driver.DTexture,
			Stages: driver.SFragment,
			Nr:     1,
			Len:    1,
		},
		{
			Type:   driver.DSampler,
			Stages: driver.SFragment,
			Nr:     2,
			Len:    1,
		},
	}
	dheap, err := gpu.NewDescHeap(desc)
	if err != nil {
		log.Fatal(err)
	}
	dtab, err := gpu.NewDescTable([]driver.DescHeap{dheap})
	if err != nil {
		log.Fatal(err)
	}

	// Descriptors are in effect references to resources.
	// This means that the data they refer must not change
	// until execution completes. When there are multiple
	// instances that use different resources, additional
	// heap copies need to be created.
	if err := dheap.New(NFrame); err != nil {
		log.Fatal(err)
	}
	for i := range NFrame {
		dheap.SetBuffer(i, 0, 0, []driver.Buffer{t.constBuf}, []int64{int64(256 * i)}, []int64{64})
		dheap.SetImage(i, 1, 0, []driver.ImageView{t.splView}, nil)
		dheap.SetSampler(i, 2, 0, []driver.Sampler{t.splr})
	}

	t.dheap = dheap
	t.dtab = dtab
}

// pipelineSetup creates the graphics pipeline.
func (t *T) pipelineSetup() {
	gs := driver.GraphState{
		VertFunc: t.vertFunc,
		FragFunc: t.fragFunc,
		Desc:     t.dtab,
		Input: []driver.VertexIn{
			{
				Format: driver.Float32x3,
				Stride: 4 * 3,
				Nr:     0,
			},
			{
				Format: driver.Float32x2,
				Stride: 4 * 2,
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
		Samples: Samples,
		DS: driver.DSState{
			DepthTest:   true,
			DepthWrite:  true,
			DepthCmp:    driver.CLessEqual,
			StencilTest: false,
		},
		Blend: driver.BlendState{
			IndependentBlend: false,
			Color: []driver.ColorBlend{{
				Blend:     false,
				WriteMask: driver.CAll,
			}},
		},
		ColorFmt: []driver.PixelFmt{t.sc.Format()},
		DSFmt:    DepthFmt,
	}
	pipeln, err := gpu.NewPipeline(&gs)
	if err != nil {
		log.Fatal(err)
	}

	t.pipeln = pipeln
}

// renderLoop renders the cube in a loop.
func (t *T) renderLoop() {
	var err error
	for i := range cap(t.ch) {
		wk := &driver.WorkItem{Work: []driver.CmdBuffer{t.cb[i]}, Custom: i}
		t.ch <- wk
	}
	t0 := time.Now()
	t1 := t0
	t.auto = true
	for !t.quit {
		wk := <-t.ch
		if err = wk.Err; err != nil {
			switch err {
			case driver.ErrFatal:
				log.Fatal(err)
			default:
				log.Printf("GPU.Commit (WorkItem.Err): %v\n", err)
			}
		}
		cb := wk.Work[0]
		frame := wk.Custom.(int)

		wsi.Dispatch()
		if t.broken {
			t.recreateSwapchain()
			t.broken = false
		}

		dt := t1.Sub(t0)
		t0, t1 = t1, time.Now()

		// Begin must come before anything else.
		if err = cb.Begin(); err != nil {
			log.Fatal(err)
		}

		next := -1
	nextLoop:
		for {
			next, err = t.sc.Next()
			switch err {
			case nil:
				// Got a backbuffer to use as render target.
				break nextLoop
			case driver.ErrNoBackbuffer:
				// No backbuffer available, try again.
				time.Sleep(time.Millisecond * 10)
				continue
			case driver.ErrSwapchain:
				// The swapchain is broken, we need to
				// recreate it.
				t.recreateSwapchain()
				continue
			default:
				log.Fatal(err)
			}
		}

		// Update per-frame constant data and copy it into the
		// GPU private buffer.
		// Note that, as long as we use the same buffer range,
		// we need not set the descriptor heap again.
		t.updateTransform(dt)
		copy(t.stgBuf.Bytes()[256*frame:], unsafe.Slice((*byte)(unsafe.Pointer(&t.xform[0])), 64))
		cb.CopyBuffer(&driver.BufferCopy{
			From:    t.stgBuf,
			FromOff: int64(256 * frame),
			To:      t.constBuf,
			ToOff:   int64(256 * frame),
			Size:    64,
		})

		// Make sure that the above copy happens before the
		// vertex shader executes.
		cb.Barrier([]driver.Barrier{{
			SyncBefore:   driver.SCopy,
			SyncAfter:    driver.SVertexShading,
			AccessBefore: driver.ACopyWrite,
			AccessAfter:  driver.AShaderRead,
		}})

		// The render targets must be in a valid layout when
		// they are accessed by the GPU.
		cb.Transition([]driver.Transition{
			{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SColorOutput,
					SyncAfter:    driver.SColorOutput,
					AccessBefore: driver.AColorWrite,
					AccessAfter:  driver.AColorWrite,
				},
				LayoutBefore: driver.LUndefined,
				LayoutAfter:  driver.LColorTarget,
				Img:          t.rt[next].Color.Image(),
				Layers:       1,
				Levels:       1,
			},
			{
				Barrier: driver.Barrier{
					SyncBefore:  driver.SColorOutput,
					SyncAfter:   driver.SColorOutput,
					AccessAfter: driver.AColorWrite,
				},
				LayoutBefore: driver.LUndefined,
				LayoutAfter:  driver.LColorTarget,
				Img:          t.rt[next].Resolve.Image(),
				Layers:       1,
				Levels:       1,
			},
			{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SDSOutput,
					SyncAfter:    driver.SDSOutput,
					AccessBefore: driver.ADSWrite,
					AccessAfter:  driver.ADSRead | driver.ADSWrite,
				},
				LayoutBefore: driver.LUndefined,
				LayoutAfter:  driver.LDSTarget,
				Img:          t.dsImg,
				Layers:       1,
				Levels:       1,
			},
		})

		// Now we can draw the cube.
		cb.BeginPass(t.dim.Width, t.dim.Height, 1, []driver.ColorTarget{t.rt[next]}, &t.ds)
		cb.SetPipeline(t.pipeln)
		cb.SetViewport(t.vport)
		cb.SetScissor(t.sciss)
		cb.SetVertexBuf(0, []driver.Buffer{t.vertBuf, t.vertBuf}, []int64{0, cubePosSize})
		cb.SetIndexBuf(driver.Index32, t.idxBuf, 0)
		cb.SetDescTableGraph(t.dtab, 0, []int{frame})
		cb.DrawIndexed(len(cubeIdx), 1, 0, 0, 0)
		cb.EndPass()

		// The backbuffer must be in the driver.LPresent layout
		// to be presented.
		cb.Transition([]driver.Transition{{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SColorOutput,
				SyncAfter:    driver.SColorOutput,
				AccessBefore: driver.AColorWrite,
			},
			LayoutBefore: driver.LColorTarget,
			LayoutAfter:  driver.LPresent,
			Img:          t.rt[next].Resolve.Image(),
			Layers:       1,
			Levels:       1,
		}})

		// End must be called when done recording commands.
		if err := cb.End(); err != nil {
			log.Fatal(err)
		}

		// Commit the commands for this frame.
		// Notice that we do not wait for the work to complete.
		if err := gpu.Commit(wk, t.ch); err != nil {
			log.Fatal(err)
		}

		// Now we can present the swapchain's view.
		if err := t.sc.Present(next); err != nil {
			switch err {
			case driver.ErrSwapchain:
				log.Printf("Swapchain.Present: %v\n", err)
			default:
				log.Fatal(err)
			}
		}

		// We are done with this frame, so start working on
		// the next one.
	}
	for range cap(t.ch) {
		<-t.ch
	}
}

// destroy frees all data.
func (t *T) destroy() {
	for _, cb := range t.cb {
		cb.Destroy()
	}
	t.pipeln.Destroy()
	t.dtab.Destroy()
	t.dheap.Destroy()
	t.splView.Destroy()
	t.splImg.Destroy()
	t.splr.Destroy()
	t.stgBuf.Destroy()
	t.vertBuf.Destroy()
	t.idxBuf.Destroy()
	t.constBuf.Destroy()
	t.dsView.Destroy()
	t.dsImg.Destroy()
	t.rtView.Destroy()
	t.rtImg.Destroy()
	t.sc.Destroy()
	t.win.Close()
}

var (
	// Vertex positions (CCW).
	cubePos = [24 * 3]float32{
		-1, -1, +1,
		-1, +1, +1,
		-1, +1, -1,
		-1, -1, -1,

		+1, -1, -1,
		+1, +1, -1,
		+1, +1, +1,
		+1, -1, +1,

		+1, -1, -1,
		+1, -1, +1,
		-1, -1, +1,
		-1, -1, -1,

		-1, +1, -1,
		-1, +1, +1,
		+1, +1, +1,
		+1, +1, -1,

		-1, -1, -1,
		-1, +1, -1,
		+1, +1, -1,
		+1, -1, -1,

		+1, -1, +1,
		+1, +1, +1,
		-1, +1, +1,
		-1, -1, +1,
	}

	// Vertex UVs.
	cubeUV = [24 * 2]float32{
		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,

		0, 0,
		0, 1,
		1, 1,
		1, 0,
	}

	// Input assembly indices.
	cubeIdx = [36]uint32{
		0, 1, 2,
		0, 2, 3,
		4, 5, 6,
		4, 6, 7,
		8, 9, 10,
		8, 10, 11,
		12, 13, 14,
		12, 14, 15,
		16, 17, 18,
		16, 18, 19,
		20, 21, 22,
		20, 22, 23,
	}
)

const (
	cubePosSize = int64(unsafe.Sizeof(cubePos))
	cubeUVSize  = int64(unsafe.Sizeof(cubeUV))
	cubeIdxSize = int64(unsafe.Sizeof(cubeIdx))
)

// updateTransform is called every frame to update the
// transform matrix used by the cube.
func (t *T) updateTransform(dt time.Duration) {
	var proj, view, model, vp linear.M4

	w := float32(t.win.Width())
	h := float32(t.win.Height())
	if w < h {
		w, h = w/h, 1
	} else {
		w, h = 1, h/w
	}
	proj.Frustum(-w, w, -h, h, 1, 100)

	var center linear.V3
	eye := linear.V3{2, -3, -4}
	up := linear.V3{0, -1, 0}
	view.LookAt(&center, &eye, &up)

	if t.auto {
		model.Rotate(t.angleY, &up)
		t.angleY += float32(dt.Seconds()) * 5
		if t.angleY > 2*math.Pi {
			t.angleY = t.angleY - 2*math.Pi
		}
	} else {
		x := float32(math.Cos(float64(t.angleY)))
		z := float32(math.Sin(float64(t.angleY)))
		model.Rotate(t.angleX, &linear.V3{x, 0, z})
		var yaw linear.M4
		yaw.Rotate(t.angleY, &up)
		model.Mul(&model, &yaw)
		t.angleX += float32(dt.Seconds()) * t.turnX
		t.angleY += float32(dt.Seconds()) * t.turnY
		for _, angle := range [2]*float32{&t.angleX, &t.angleY} {
			if *angle > 2*math.Pi {
				*angle = *angle - 2*math.Pi
			} else if *angle < -2*math.Pi {
				*angle = *angle + 2*math.Pi
			}
		}
	}

	vp.Mul(&proj, &view)
	t.xform.Mul(&vp, &model)
}

// recreateSwapchain recreates the swapchain and all
// framebuffers.
func (t *T) recreateSwapchain() {
	// Wait for ongoing Commit calls to complete.
	var wk [NFrame - 1]*driver.WorkItem
	for i := range NFrame - 1 {
		wk[i] = <-t.ch
	}
	for _, wk := range wk {
		t.ch <- wk
	}

	var err error
	pf := t.sc.Format()
	if err = t.sc.Recreate(); err != nil {
		log.Fatal(err)
	}
	scViews := t.sc.Views()
	if pf != t.sc.Format() || len(scViews) != len(t.rt) {
		// The solution would be to recreate the pipeline,
		// which is expensive.
		log.Fatal("recreate swapchain mismatch")
	}

	width := t.win.Width()
	height := t.win.Height()
	if t.dim.Width != width || t.dim.Height != height {
		t.dim.Width = width
		t.dim.Height = height

		t.rtView.Destroy()
		t.rtImg.Destroy()
		t.rtImg, err = gpu.NewImage(pf, t.dim, 1, 1, Samples, driver.URenderTarget)
		if err != nil {
			log.Fatal(err)
		}
		t.rtView, err = t.rtImg.NewView(driver.IView2D, 0, 1, 0, 1)
		if err != nil {
			log.Fatal(err)
		}
		for i := range t.rt {
			t.rt[i].Color = t.rtView
		}

		t.dsView.Destroy()
		t.dsImg.Destroy()
		t.dsImg, err = gpu.NewImage(DepthFmt, t.dim, 1, 1, Samples, driver.URenderTarget)
		if err != nil {
			log.Fatal(err)
		}
		t.dsView, err = t.dsImg.NewView(driver.IView2D, 0, 1, 0, 1)
		if err != nil {
			log.Fatal(err)
		}
		t.ds.DS = t.dsView

		t.vport.Width = float32(width)
		t.vport.Height = float32(height)
		t.sciss.Width = width
		t.sciss.Height = height
	}

	for i := range t.rt {
		t.rt[i].Resolve = scViews[i]
	}
}

func (t *T) WindowClose(win wsi.Window) {
	if win == t.win {
		t.quit = true
	}
}

func (t *T) WindowResize(wsi.Window, int, int) { t.broken = true }

func (t *T) KeyboardKey(key wsi.Key, pressed bool) {
	switch key {
	case wsi.KeyEsc:
		t.quit = t.quit || pressed
	case wsi.KeyUp:
		t.auto = false
		if pressed {
			t.turnX = -1
		} else {
			t.turnX = 0
		}
	case wsi.KeyDown:
		t.auto = false
		if pressed {
			t.turnX = 1
		} else {
			t.turnX = 0
		}
	case wsi.KeyLeft:
		t.auto = false
		if pressed {
			t.turnY = -1
		} else {
			t.turnY = 0
		}
	case wsi.KeyRight:
		t.auto = false
		if pressed {
			t.turnY = 1
		} else {
			t.turnY = 0
		}
	default:
		t.turnX = 0
		t.turnY = 0
	}
}
