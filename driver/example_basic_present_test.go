// Copyright 2025 Gustavo C. Viegas. All rights reserved.

package driver_test

import (
	"bytes"
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

type U struct {
	cb       [NFrame]driver.CmdBuffer
	ch       chan *driver.WorkItem
	win      wsi.Window
	sc       driver.Swapchain
	dim      driver.Dim3D
	rt       []driver.ColorTarget
	ds       driver.DSTarget
	dsImg    driver.Image
	dsView   driver.ImageView
	vertFunc driver.ShaderFunc
	fragFunc driver.ShaderFunc
	stgBuf   driver.Buffer
	vertBuf  driver.Buffer
	idxBuf   driver.Buffer
	constBuf driver.Buffer
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

// Example_basicPresent is a stripped-down version
// of Example_present (no texture sampling; no MS).
func Example_basicPresent() {
	var u U
	var err error
	for i := range u.cb {
		u.cb[i], err = gpu.NewCmdBuffer()
		if err != nil {
			log.Fatal(err)
		}
	}
	u.ch = make(chan *driver.WorkItem, NFrame)
	u.swapchainSetup()
	u.passSetup()
	u.shaderSetup()
	u.bufferSetup()
	u.descriptorSetup()
	u.pipelineSetup()
	u.vport = driver.Viewport{
		X:      0,
		Y:      0,
		Width:  float32(u.dim.Width),
		Height: float32(u.dim.Height),
		Znear:  0,
		Zfar:   1,
	}
	u.sciss = driver.Scissor{
		X:      0,
		Y:      0,
		Width:  u.dim.Width,
		Height: u.dim.Height,
	}
	wsi.SetWindowHandler(&u)
	wsi.SetKeyboardKeyHandler(&u)
	wsi.SetAppName("driver.example 2")
	u.renderLoop()
	u.destroy()

	// Output:
}

func (u *U) swapchainSetup() {
	if wsi.PlatformInUse() == wsi.None {
		log.Fatal("WSI not available")
	}
	win, err := wsi.NewWindow(400, 300, "Basic Present Example")
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

	u.win = win
	u.sc = sc
	u.dim.Width = win.Width()
	u.dim.Height = win.Height()
}

func (u *U) passSetup() {
	scViews := u.sc.Views()
	rt := make([]driver.ColorTarget, len(scViews))
	for i := range rt {
		rt[i] = driver.ColorTarget{
			Color: scViews[i],
			Load:  driver.LClear,
			Store: driver.SStore,
			Clear: driver.ClearFloat32(0.05, 0.05, 0.05, 1),
		}
	}

	dsImg, err := gpu.NewImage(DepthFmt, u.dim, 1, 1, 1, driver.URenderTarget)
	if err != nil {
		log.Fatal(err)
	}
	dsView, err := dsImg.NewView(driver.IView2D, 0, 1, 0, 1)
	if err != nil {
		log.Fatal(err)
	}

	u.rt = rt
	u.ds = driver.DSTarget{
		DS:     dsView,
		LoadD:  driver.LClear,
		StoreD: driver.SDontCare,
		ClearD: 1,
	}
	u.dsImg = dsImg
	u.dsView = dsView
}

func (u *U) shaderSetup() {
	var shd [2]struct {
		fileName, funcName string
	}
	switch name := drv.Name(); {
	case strings.Contains(strings.ToLower(name), "vulkan"):
		shd[0].fileName = "basic_cube_vs.spv"
		shd[0].funcName = "main"
		shd[1].fileName = "basic_cube_fs.spv"
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

	u.vertFunc = driver.ShaderFunc{
		Code: code[0],
		Name: shd[0].funcName,
	}
	u.fragFunc = driver.ShaderFunc{
		Code: code[1],
		Name: shd[1].funcName,
	}
}

func (u *U) bufferSetup() {
	const (
		vbSize = cubePosSize
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

	stg := stgBuf.Bytes()
	pos := unsafe.Slice((*byte)(unsafe.Pointer(&cubePos[0])), cubePosSize)
	idx := unsafe.Slice((*byte)(unsafe.Pointer(&cubeIdx[0])), cubeIdxSize)
	copy(stg, pos)
	copy(stg[vbSize:], idx)
	if err := u.cb[0].Begin(); err != nil {
		log.Fatal(err)
	}
	u.cb[0].CopyBuffer(&driver.BufferCopy{
		From:    stgBuf,
		FromOff: 0,
		To:      vertBuf,
		ToOff:   0,
		Size:    vbSize,
	})
	u.cb[0].CopyBuffer(&driver.BufferCopy{
		From:    stgBuf,
		FromOff: vbSize,
		To:      idxBuf,
		ToOff:   0,
		Size:    ibSize,
	})
	if err := u.cb[0].End(); err != nil {
		log.Fatal(err)
	}
	wk := driver.WorkItem{Work: []driver.CmdBuffer{u.cb[0]}}
	ch := make(chan *driver.WorkItem, 1)
	if err := gpu.Commit(&wk, ch); err != nil {
		log.Fatal(err)
	}
	if err := (<-ch).Err; err != nil {
		log.Fatal(err)
	}

	u.stgBuf = stgBuf
	u.vertBuf = vertBuf
	u.idxBuf = idxBuf
	u.constBuf = constBuf
}

func (u *U) descriptorSetup() {
	desc := []driver.Descriptor{{
		Type:   driver.DConstant,
		Stages: driver.SVertex,
		Nr:     0,
		Len:    1,
	}}
	dheap, err := gpu.NewDescHeap(desc)
	if err != nil {
		log.Fatal(err)
	}
	dtab, err := gpu.NewDescTable([]driver.DescHeap{dheap})
	if err != nil {
		log.Fatal(err)
	}

	if err := dheap.New(NFrame); err != nil {
		log.Fatal(err)
	}
	for i := range NFrame {
		dheap.SetBuffer(i, 0, 0, []driver.Buffer{u.constBuf},
			[]int64{int64(256 * i)}, []int64{int64(unsafe.Sizeof(u.xform))})
	}

	u.dheap = dheap
	u.dtab = dtab
}

func (u *U) pipelineSetup() {
	gs := driver.GraphState{
		VertFunc: u.vertFunc,
		FragFunc: u.fragFunc,
		Desc:     u.dtab,
		Input: []driver.VertexIn{{
			Format: driver.Float32x3,
			Stride: 4 * 3,
			Nr:     0,
		}},
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
		ColorFmt: []driver.PixelFmt{u.sc.Format()},
		DSFmt:    DepthFmt,
	}
	pipeln, err := gpu.NewPipeline(&gs)
	if err != nil {
		log.Fatal(err)
	}

	u.pipeln = pipeln
}

func (u *U) renderLoop() {
	var err error
	for i := range cap(u.ch) {
		wk := &driver.WorkItem{Work: []driver.CmdBuffer{u.cb[i]}, Custom: i}
		u.ch <- wk
	}
	t0 := time.Now()
	t1 := t0
	u.auto = true
	for !u.quit {
		wk := <-u.ch
		if err := wk.Err; err != nil {
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
		if u.broken {
			// TODO
			log.Fatal("u.broken")
		}

		dt := t1.Sub(t0)
		t0, t1 = t1, time.Now()

		if err := cb.Begin(); err != nil {
			log.Fatal(err)
		}

		next := -1
	nextLoop:
		for {
			next, err = u.sc.Next()
			switch err {
			case nil:
				break nextLoop
			case driver.ErrNoBackbuffer:
				time.Sleep((time.Millisecond * 10))
				continue
			case driver.ErrSwapchain:
				// TODO
				log.Fatal("U.recreateSwapchain")
				continue
			default:
				log.Fatal(err)
			}
		}

		u.updateTransform(dt)
		copy(u.stgBuf.Bytes()[256*frame:],
			unsafe.Slice((*byte)(unsafe.Pointer(&u.xform[0])), 64))
		cb.CopyBuffer(&driver.BufferCopy{
			From:    u.stgBuf,
			FromOff: int64(256 * frame),
			To:      u.constBuf,
			ToOff:   int64(256 * frame),
			Size:    64,
		})

		cb.Barrier([]driver.Barrier{{
			SyncBefore:   driver.SCopy,
			SyncAfter:    driver.SVertexShading,
			AccessBefore: driver.ACopyWrite,
			AccessAfter:  driver.AShaderRead,
		}})

		cb.Transition([]driver.Transition{{
			Barrier: driver.Barrier{
				SyncBefore:  driver.SColorOutput,
				SyncAfter:   driver.SColorOutput,
				AccessAfter: driver.AColorWrite,
			},
			LayoutBefore: driver.LUndefined,
			LayoutAfter:  driver.LColorTarget,
			Img:          u.rt[next].Color.Image(),
			Layers:       1,
			Levels:       1,
		}})

		cb.BeginPass(u.dim.Width, u.dim.Height, 1, []driver.ColorTarget{u.rt[next]}, &u.ds)
		cb.SetPipeline(u.pipeln)
		cb.SetViewport(u.vport)
		cb.SetScissor(u.sciss)
		cb.SetVertexBuf(0, []driver.Buffer{u.vertBuf}, []int64{0})
		cb.SetIndexBuf(driver.Index32, u.idxBuf, 0)
		cb.SetDescTableGraph(u.dtab, 0, []int{frame})
		cb.DrawIndexed(len(cubeIdx), 1, 0, 0, 0)
		cb.EndPass()

		cb.Transition([]driver.Transition{{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SColorOutput,
				SyncAfter:    driver.SColorOutput,
				AccessBefore: driver.AColorWrite,
			},
			LayoutBefore: driver.LColorTarget,
			LayoutAfter:  driver.LPresent,
			Img:          u.rt[next].Color.Image(),
			Layers:       1,
			Levels:       1,
		}})

		if err := cb.End(); err != nil {
			log.Fatal(err)
		}

		if err := gpu.Commit(wk, u.ch); err != nil {
			log.Fatal(err)
		}

		if err := u.sc.Present(next); err != nil {
			switch err {
			case driver.ErrSwapchain:
				log.Printf("Swapchain.present: %v\n", err)
			default:
				log.Fatal(err)
			}
		}
	}
	for range cap(u.ch) {
		<-u.ch
	}
}

func (u *U) destroy() {
	for _, cb := range u.cb {
		cb.Destroy()
	}
	u.pipeln.Destroy()
	u.dtab.Destroy()
	u.dheap.Destroy()
	u.stgBuf.Destroy()
	u.vertBuf.Destroy()
	u.idxBuf.Destroy()
	u.constBuf.Destroy()
	u.dsView.Destroy()
	u.dsImg.Destroy()
	u.sc.Destroy()
	u.win.Close()
}

func (u *U) updateTransform(dt time.Duration) {
	var proj, view, model, vp linear.M4

	w := float32(u.dim.Width)
	h := float32(u.dim.Height)
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

	if u.auto {
		model.Rotate(u.angleY, &up)
		u.angleY += float32(dt.Seconds()) * 5
		if u.angleY > 2*math.Pi {
			u.angleY = u.angleY - 2*math.Pi
		}
	} else {
		x := float32(math.Cos(float64(u.angleY)))
		z := float32(math.Sin(float64(u.angleY)))
		model.Rotate(u.angleX, &linear.V3{x, 0, z})
		var yaw linear.M4
		yaw.Rotate(u.angleY, &up)
		model.Mul(&model, &yaw)
		u.angleX += float32(dt.Seconds()) * u.turnX
		u.angleY += float32(dt.Seconds()) * u.turnY
		for _, angle := range [2]*float32{&u.angleX, &u.angleY} {
			if *angle > 2*math.Pi {
				*angle = *angle - 2*math.Pi
			} else if *angle < -2*math.Pi {
				*angle = *angle + 2*math.Pi
			}
		}
	}

	vp.Mul(&proj, &view)
	u.xform.Mul(&vp, &model)
}

func (u *U) WindowClose(win wsi.Window) {
	if win == u.win {
		u.quit = true
	}
}

func (u *U) WindowResize(wsi.Window, int, int) { u.broken = true }

func (u *U) KeyboardKey(key wsi.Key, pressed bool) {
	switch key {
	case wsi.KeyEsc:
		u.quit = u.quit || pressed
	case wsi.KeyUp:
		u.auto = false
		if pressed {
			u.turnX = -1
		} else {
			u.turnX = 0
		}
	case wsi.KeyDown:
		u.auto = false
		if pressed {
			u.turnX = 1
		} else {
			u.turnX = 0
		}
	case wsi.KeyLeft:
		u.auto = false
		if pressed {
			u.turnY = -1
		} else {
			u.turnY = 0
		}
	case wsi.KeyRight:
		u.auto = false
		if pressed {
			u.turnY = 1
		} else {
			u.turnY = 0
		}
	default:
		u.turnX = 0
		u.turnY = 0
	}
}
