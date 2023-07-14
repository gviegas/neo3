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
	"gviegas/neo3/wsi"
)

const NFrame = 3

const DepthFmt = driver.D16un

var dim = driver.Dim3D{
	Width:  480,
	Height: 300,
}

var brokenSC bool

type T struct {
	cb       [NFrame]driver.CmdBuffer
	ch       chan *driver.WorkItem
	win      wsi.Window
	sc       driver.Swapchain
	rt       []driver.ColorTarget
	ds       driver.DSTarget
	dsImg    driver.Image
	dsView   driver.ImageView
	vertFunc driver.ShaderFunc
	fragFunc driver.ShaderFunc
	vertBuf  driver.Buffer
	constBuf driver.Buffer
	splImg   driver.Image
	splView  driver.ImageView
	splr     driver.Sampler
	dheap    driver.DescHeap
	dtab     driver.DescTable
	pipeln   driver.Pipeline
	vport    driver.Viewport
	sciss    driver.Scissor
	xform    M
	angle    float32
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
		Width:  float32(dim.Width),
		Height: float32(dim.Height),
		Znear:  0,
		Zfar:   1,
	}
	t.sciss = driver.Scissor{
		X:      0,
		Y:      0,
		Width:  dim.Width,
		Height: dim.Height,
	}
	wsi.SetWindowCloseHandler(&t)
	wsi.SetWindowResizeHandler(&t)
	wsi.SetKeyboardKeyHandler(&t)
	wsi.SetAppName("driver.example")
	t.renderLoop()
	t.destroy()

	// Output:
}

// swapchainSetup creates the window and swapchain.
func (t *T) swapchainSetup() {
	if wsi.PlatformInUse() == wsi.None {
		log.Fatal("WSI unavailable")
	}
	win, err := wsi.NewWindow(dim.Width, dim.Height, "Present Example")
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
}

// passSetup creates the depth image/view and sets the
// render targets to be used during render passes.
func (t *T) passSetup() {
	scViews := t.sc.Views()
	rt := make([]driver.ColorTarget, len(scViews))
	for i := range rt {
		rt[i] = driver.ColorTarget{
			Color: scViews[i],
			Load:  driver.LClear,
			Store: driver.SStore,
			Clear: [4]float32{0.075, 0.075, 0.075, 1},
		}
	}

	dsImg, err := gpu.NewImage(DepthFmt, dim, 1, 1, 1, driver.URenderTarget)
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
	t.ds = ds
	t.dsImg = dsImg
	t.dsView = dsView
}

// shaderSetup creates the vertex and fragment shaders.
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

	buf := bytes.Buffer{}
	code := [2]driver.ShaderCode{}
	for i := range code {
		file, err := os.Open("testdata/" + shd[i].fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		_, err = buf.ReadFrom(file)
		if err != nil {
			log.Fatal(err)
		}
		code[i], err = gpu.NewShaderCode(buf.Bytes())
		if err != nil {
			log.Fatal(err)
		}
		buf.Reset()
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

// bufferSetup creates the vertex buffer to store vertex data
// and the constant buffer to store shader constants (uniforms).
func (t *T) bufferSetup() {
	// Since vertex data is not going to change, we could have
	// created the buffer as GPU private instead and used a
	// staging buffer to do the copying.
	vertBuf, err := gpu.NewBuffer(1024, true, driver.UVertexData)
	if err != nil {
		log.Fatal(err)
	}
	copy(vertBuf.Bytes(), unsafe.Slice((*byte)(unsafe.Pointer(&cubePos[0])), len(cubePos)*4))
	copy(vertBuf.Bytes()[512:], unsafe.Slice((*byte)(unsafe.Pointer(&cubeTexCoord[0])), len(cubeTexCoord)*4))

	// Shader data is going to change every frame, so it makes
	// more sense to have it as CPU visible.
	constBuf, err := gpu.NewBuffer(512*NFrame, true, driver.UShaderConst)
	if err != nil {
		log.Fatal(err)
	}
	t.xform.identity()
	copy(constBuf.Bytes(), unsafe.Slice((*byte)(unsafe.Pointer(&t.xform[0])), len(t.xform)*4))

	t.vertBuf = vertBuf
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
	buf, err := gpu.NewBuffer(int64(len(pix)), true, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Destroy()
	copy(buf.Bytes(), pix)

	size := driver.Dim3D{
		Width:  decImg.Bounds().Max.X,
		Height: decImg.Bounds().Max.Y,
	}
	img, err := gpu.NewImage(driver.RGBA8sRGB, size, 1, 1, 1, driver.UShaderSample)
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
	t.cb[0].Transition([]driver.Transition{
		{
			Barrier: driver.Barrier{
				SyncAfter:   driver.SCopy,
				AccessAfter: driver.ACopyWrite,
			},
			LayoutBefore: driver.LUndefined,
			LayoutAfter:  driver.LCopyDst,
			Img:          img,
			Layers:       1,
			Levels:       1,
		},
	})
	t.cb[0].CopyBufToImg(&driver.BufImgCopy{
		Buf:    buf,
		Stride: [2]int{size.Width, size.Height},
		Img:    img,
		Size:   size,
		Layers: 1,
	})
	t.cb[0].Transition([]driver.Transition{
		{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SCopy,
				AccessBefore: driver.ACopyWrite,
			},
			LayoutBefore: driver.LCopyDst,
			LayoutAfter:  driver.LShaderRead,
			Img:          img,
			Layers:       1,
			Levels:       1,
		},
	})
	if err := t.cb[0].End(); err != nil {
		log.Fatal(err)
	}
	wk := driver.WorkItem{Work: []driver.CmdBuffer{t.cb[0]}}
	ch := make(chan *driver.WorkItem)
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
	for i := 0; i < NFrame; i++ {
		dheap.SetBuffer(i, 0, 0, []driver.Buffer{t.constBuf}, []int64{int64(512 * i)}, []int64{64})
		dheap.SetImage(i, 1, 0, []driver.ImageView{t.splView}, nil)
		dheap.SetSampler(i, 2, 0, []driver.Sampler{t.splr})
	}

	t.dtab = dtab
	t.dheap = dheap
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
			Color: []driver.ColorBlend{
				{
					Blend:     false,
					WriteMask: driver.CAll,
				},
			},
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
	var frame int
	for i := 0; i < cap(t.ch); i++ {
		wk := &driver.WorkItem{Work: []driver.CmdBuffer{t.cb[i]}}
		t.ch <- wk
	}
	for !t.quit {
		wk := <-t.ch
		if err = wk.Err; err != nil {
			switch err {
			case driver.ErrFatal:
				log.Fatal(err)
			default:
				log.Printf("GPU.Commit <send>: %v\n", err)
			}
		}
		cb := wk.Work[0]

		wsi.Dispatch()
		if brokenSC {
			t.recreateSwapchain()
			brokenSC = false
		}

		// Note that, as long as we use the same buffer range,
		// we need not set the descriptor heap again.
		t.updateTransform(time.Second / 60)
		copy(t.constBuf.Bytes()[512*frame:], unsafe.Slice((*byte)(unsafe.Pointer(&t.xform[0])), 64))

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

		// After acquiring a backbuffer that we can use as
		// render target, we transition it to a valid layout.
		cb.Transition([]driver.Transition{
			{
				Barrier: driver.Barrier{
					SyncBefore:  driver.SColorOutput,
					SyncAfter:   driver.SColorOutput,
					AccessAfter: driver.AColorWrite,
				},
				LayoutBefore: driver.LUndefined,
				LayoutAfter:  driver.LColorTarget,
				Img:          t.rt[next].Color.Image(),
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

		// We now record a render pass that draws the cube
		// in the image view we acquired previously.
		cb.BeginPass(dim.Width, dim.Height, 1, []driver.ColorTarget{t.rt[next]}, &t.ds)
		cb.SetPipeline(t.pipeln)
		cb.SetViewport([]driver.Viewport{t.vport})
		cb.SetScissor([]driver.Scissor{t.sciss})
		cb.SetVertexBuf(0, []driver.Buffer{t.vertBuf, t.vertBuf}, []int64{0, 512})
		cb.SetDescTableGraph(t.dtab, 0, []int{frame})
		cb.Draw(36, 1, 0, 0)
		cb.EndPass()

		// When done writing to the image, we transition it
		// to driver.LPresent so we can present the result.
		cb.Transition([]driver.Transition{
			{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SColorOutput,
					SyncAfter:    driver.SColorOutput,
					AccessBefore: driver.AColorWrite,
					AccessAfter:  driver.AColorRead,
				},
				LayoutBefore: driver.LColorTarget,
				LayoutAfter:  driver.LPresent,
				Img:          t.rt[next].Color.Image(),
				Layers:       1,
				Levels:       1,
			},
		})

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
		frame = (frame + 1) % NFrame
	}
	for len(t.ch) != cap(t.ch) {
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
	t.vertBuf.Destroy()
	t.constBuf.Destroy()
	t.vertFunc.Code.Destroy()
	t.fragFunc.Code.Destroy()
	t.dsView.Destroy()
	t.dsImg.Destroy()
	t.sc.Destroy()
	t.win.Close()
}

// Cube positions.
var cubePos = [36 * 3]float32{
	-1, +1, -1,
	+1, +1, +1,
	+1, +1, -1,
	+1, +1, +1,
	-1, -1, +1,
	+1, -1, +1,

	-1, +1, +1,
	-1, -1, -1,
	-1, -1, +1,
	+1, -1, -1,
	-1, -1, +1,
	-1, -1, -1,

	+1, +1, -1,
	+1, -1, +1,
	+1, -1, -1,
	-1, +1, -1,
	+1, -1, -1,
	-1, -1, -1,

	-1, +1, -1,
	-1, +1, +1,
	+1, +1, +1,
	+1, +1, +1,
	-1, +1, +1,
	-1, -1, +1,

	-1, +1, +1,
	-1, +1, -1,
	-1, -1, -1,
	+1, -1, -1,
	+1, -1, +1,
	-1, -1, +1,

	+1, +1, -1,
	+1, +1, +1,
	+1, -1, +1,
	-1, +1, -1,
	+1, +1, -1,
	+1, -1, -1,
}

// Cube texture coordinates.
var cubeTexCoord = [36 * 2]float32{
	0, 1,
	1, 0,
	1, 1,
	1, 1,
	0, 0,
	1, 0,

	0, 1,
	1, 0,
	0, 0,
	1, 1,
	0, 0,
	0, 1,

	1, 1,
	0, 0,
	1, 0,
	0, 1,
	1, 0,
	0, 0,

	0, 1,
	0, 0,
	1, 0,
	1, 1,
	0, 1,
	0, 0,

	0, 1,
	1, 1,
	1, 0,
	1, 1,
	1, 0,
	0, 0,

	1, 1,
	0, 1,
	0, 0,
	0, 1,
	1, 1,
	1, 0,
}

// Vector.
type V [3]float32

func (v *V) normalize() {
	length := float32(math.Sqrt(float64(v.dot(v))))
	for i := range v {
		v[i] /= length
	}
}

func (v *V) dot(u *V) float32 {
	d := float32(0)
	for i := range v {
		d += v[i] * u[i]
	}
	return d
}

func (v *V) cross(v1, v2 *V) {
	v[0] = v1[1]*v2[2] - v1[2]*v2[1]
	v[1] = v1[2]*v2[0] - v1[0]*v2[2]
	v[2] = v1[0]*v2[1] - v1[1]*v2[0]
}

func (v *V) subtract(v1, v2 *V) {
	for i := range v {
		v[i] = v1[i] - v2[i]
	}
}

// Matrix.
type M [16]float32

func (m *M) identity() {
	*m = M{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func (m *M) multiply(m1, m2 *M) {
	m.identity()
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			m[4*i+j] = 0
			for k := 0; k < 4; k++ {
				m[4*i+j] += m1[4*k+j] * m2[4*i+k]
			}
		}
	}
}

func (m *M) infPerspective(yfov, aspectRatio, znear float32) {
	*m = M{}
	ct := float32(1 / math.Tan(float64(yfov)*0.5))
	m[0] = ct / aspectRatio
	m[5] = ct
	m[10] = -1
	m[11] = -1
	m[14] = -2 * znear
}

func (m *M) lookAt(eye, center, up *V) {
	var f, s, u V
	f.subtract(center, eye)
	f.normalize()
	s.cross(&f, up)
	s.normalize()
	u.cross(&f, &s)
	m[0] = s[0]
	m[1] = u[0]
	m[2] = -f[0]
	m[3] = 0
	m[4] = s[1]
	m[5] = u[1]
	m[6] = -f[1]
	m[7] = 0
	m[8] = s[2]
	m[9] = u[2]
	m[10] = -f[2]
	m[11] = 0
	m[12] = -s.dot(eye)
	m[13] = -u.dot(eye)
	m[14] = f.dot(eye)
	m[15] = 1
}

func (m *M) rotate(axis *V, angle float32) {
	m.identity()
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))
	v := *axis
	v.normalize()
	xx := v[0] * v[0]
	xy := v[0] * v[1]
	xz := v[0] * v[2]
	yy := v[1] * v[1]
	yz := v[1] * v[2]
	zz := v[2] * v[2]
	icos := 1 - cos
	sinx := sin * v[0]
	siny := sin * v[1]
	sinz := sin * v[2]
	m[0] = cos + icos*xx
	m[1] = icos*xy + sinz
	m[2] = icos*xz - siny
	m[4] = icos*xy - sinz
	m[5] = cos + icos*yy
	m[6] = icos*yz + sinx
	m[8] = icos*xz + siny
	m[9] = icos*yz - sinx
	m[10] = cos + icos*zz
}

// updateTransform is called every frame to update the
// transform matrix used by the cube.
func (t *T) updateTransform(dt time.Duration) {
	var proj, view, model, vp M
	proj.infPerspective(math.Pi/4, float32(t.win.Width())/float32(t.win.Height()), 0.01)

	eye := V{3, -3, -4}
	center := V{0}
	up := V{0, -1, 0}
	view.lookAt(&eye, &center, &up)

	axis := &up
	model.rotate(axis, t.angle)
	t.angle += float32(dt.Seconds()) * 5
	if t.angle > 2*math.Pi {
		t.angle = t.angle - 2*math.Pi
	}

	vp.multiply(&proj, &view)
	t.xform.multiply(&vp, &model)
}

func (t *T) WindowClose(win wsi.Window) {
	if win == t.win {
		t.quit = true
	}
}

func (*T) WindowResize(wsi.Window, int, int) { brokenSC = true }

func (t *T) KeyboardKey(key wsi.Key, pressed bool) {
	if pressed && key == wsi.KeyEsc {
		t.quit = true
	}
}

// recreateSwapchain recreates the swapchain and all
// framebuffers.
func (t *T) recreateSwapchain() {
	// Ensure that any calls to Commit have completed.
	for len(t.ch) < NFrame-1 {
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
	if dim.Width != width || dim.Height != height {
		dim.Width = width
		dim.Height = height
		t.dsView.Destroy()
		t.dsImg.Destroy()
		t.dsImg, err = gpu.NewImage(DepthFmt, dim, 1, 1, 1, driver.URenderTarget)
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
		t.rt[i].Color = scViews[i]
	}
}
