// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver

import (
	"unsafe"
)

// GPU is the main interface to an underlying driver
// implementation.
// It is used to create other types and to execute commands.
// A GPU is obtained from a call to Driver.Open.
type GPU interface {
	// Driver returns the Driver that owns the GPU.
	Driver() Driver

	// Commit commits a work item to the GPU for execution.
	// If successful, this method arranges for commands to
	// execute in the background, and ch is written to when
	// all command buffers complete execution.
	// In case that execution itself fails, wk.Err will
	// be set to indicate the cause.
	// It is invalid to use any of the command buffers in
	// wk.Work until ch is notified.
	Commit(wk *WorkItem, ch chan<- *WorkItem) error

	// NewCmdBuffer creates a new command buffer.
	NewCmdBuffer() (CmdBuffer, error)

	// NewShaderCode creates a new shader code.
	NewShaderCode(data []byte) (ShaderCode, error)

	// NewDescHeap creates a new descriptor heap.
	NewDescHeap(ds []Descriptor) (DescHeap, error)

	// NewDescTable creates a new descriptor table.
	NewDescTable(dh []DescHeap) (DescTable, error)

	// NewPipeline creates a new pipeline.
	// The state parameter must be a pointer to a GraphState or
	// a pointer to a CompState.
	NewPipeline(state any) (Pipeline, error)

	// NewBuffer creates a new buffer.
	NewBuffer(size int64, visible bool, usg Usage) (Buffer, error)

	// NewImage creates a new image.
	NewImage(pf PixelFmt, size Dim3D, layers, levels, samples int, usg Usage) (Image, error)

	// NewSampler creates a new Sampler.
	NewSampler(spln *Sampling) (Sampler, error)

	// Limits returns the implementation limits.
	// They are immutable for the lifetime of the GPU.
	Limits() Limits

	// Features returns the supported features.
	// They are immutable for the lifetime of the GPU.
	Features() Features
}

// Destroyer is the interface that wraps the Destroy method.
// Types that implement this interface may allocate external
// memory that is not managed by GC, so Destroy must be
// called explicitly to ensure such memory is deallocated.
type Destroyer interface {
	Destroy()
}

// Dim3D is a three-dimensional size.
type Dim3D struct {
	Width, Height, Depth int
}

// Off3D is a three-dimensional offset.
type Off3D struct {
	X, Y, Z int
}

// WorkItem defines a batch of command buffers for execution.
// Synchronization operations defined in a command buffer
// apply to the batch as a whole, so the order of elements
// in Work is meaningful.
// Err is set by GPU.Commit to indicate the result of the
// call, while Custom is ignored.
type WorkItem struct {
	Work   []CmdBuffer
	Err    error
	Custom any
}

// CmdBuffer is the interface that defines a command buffer.
// Commands are recorded into command buffers and later
// committed to the GPU for execution.
// The usage is as follows:
// First, call Begin to prepare the command buffer for
// recording. Then, if it succeeds:
//
// To record rendering commands:
//  1. call BeginPass
//  2. call Set* methods to configure rendering state
//  3. call Draw* commands
//  4. repeat 2-3 as needed
//  5. call EndPass
//  6. repeat 1-5 as needed
//
// To record compute commands:
//  1. call Set* methods to configure compute state
//  2. call Dispatch commands
//  3. repeat 1-2 as needed
//
// To record copy commands:
//  1. call Copy*/Fill commands
//
// To record synchronization commands:
//  1. call Barrier/Transition commands
//
// Finally, call End and, if it succeeds, GPU.Commit.
// Note that BeginPass commands must not be nested,
// and that they must always be paired with EndPass.
type CmdBuffer interface {
	Destroyer

	// Begin prepares the command buffer for recording.
	// This method must be called before any command
	// is recorded in the command buffer. It needs to
	// be called again if the command buffer is
	// executed or reset.
	Begin() error

	// BeginPass begins a render pass.
	BeginPass(width, height, layers int, color []ColorTarget, ds *DSTarget)

	// EndPass ends the current render pass.
	EndPass()

	// SetPipeline sets the pipeline.
	// There is a separate binding point for each
	// type of pipeline.
	SetPipeline(pl Pipeline)

	// SetViewport sets the bounds of the viewport.
	SetViewport(vp Viewport)

	// SetScissor sets the scissor rectangle.
	SetScissor(sciss Scissor)

	// SetBlendColor sets the constant blend color.
	SetBlendColor(r, g, b, a float32)

	// SetStencilRef sets the stencil reference value.
	SetStencilRef(value uint32)

	// SetVertexBuf sets one or more vertex buffers.
	// off must be aligned to the size of the data
	// format as specified in the vertex input of
	// the bound graphics pipeline.
	SetVertexBuf(start int, buf []Buffer, off []int64)

	// SetIndexBuf sets the index buffer.
	// off must be aligned to 4 bytes.
	SetIndexBuf(format IndexFmt, buf Buffer, off int64)

	// SetDescTableGraph sets a descriptor table
	// range for graphics pipelines.
	SetDescTableGraph(table DescTable, start int, heapCopy []int)

	// SetDescTableComp sets a descriptor table
	// range for compute pipelines.
	SetDescTableComp(table DescTable, start int, heapCopy []int)

	// Draw draws primitives.
	// It must only be called during a render pass.
	Draw(vertCnt, instCnt, baseVert, baseInst int)

	// DrawIndexed draws indexed primitives.
	// It must only be called during a render pass.
	DrawIndexed(idxCnt, instCnt, baseIdx, vertOff, baseInst int)

	// Dispatch dispatches compute thread groups.
	// It must not be called during a render pass.
	Dispatch(grpCntX, grpCntY, grpCntZ int)

	// CopyBuffer copies data between buffers.
	// It must not be called during a render pass.
	CopyBuffer(param *BufferCopy)

	// CopyImage copies data between images.
	// It must not be called during a render pass.
	CopyImage(param *ImageCopy)

	// CopyBufToImg copies data from a buffer to
	// an image.
	// It must not be called during a render pass.
	CopyBufToImg(param *BufImgCopy)

	// CopyImgToBuf copies data from an image to
	// a buffer.
	// It must not be called during a render pass.
	CopyImgToBuf(param *BufImgCopy)

	// Fill fills a buffer range with copies of
	// a byte value.
	// off and size must be aligned to 4 bytes.
	// It must not be called during a render pass.
	Fill(buf Buffer, off int64, value byte, size int64)

	// Barrier inserts a number of global barriers
	// in the command buffer.
	// It must not be called during a render pass.
	Barrier(b []Barrier)

	// Transition inserts a number of image layout
	// transitions in the command buffer.
	// It must not be called during a render pass.
	Transition(t []Transition)

	// End ends command recording and prepares the
	// command buffer for execution.
	// New recordings are not allowed until the
	// command buffer is executed or reset.
	// Upon failure, the command buffer is reset.
	End() error

	// Reset discards all recorded commands from the
	// command buffer.
	Reset() error

	// IsRecording returns whether the command buffer
	// has begun recording commands.
	// It returns true immediately after a successful
	// call to Begin and will keep doing so until the
	// next call to End or Reset.
	IsRecording() bool
}

// LoadOp is the type of a render target's load operation.
type LoadOp int

// Load operations.
const (
	LDontCare LoadOp = iota
	LClear
	LLoad
)

// StoreOp is the type of a render target's store operation.
type StoreOp int

// Store operations.
const (
	SDontCare StoreOp = iota
	SStore
)

// ClearFmt controls how clear color values are interpreted.
type ClearFmt int

// Clear color formats.
const (
	CFloat ClearFmt = iota
	CUint
	CInt
)

// ClearColor defines the color to use when clearing a
// color render target.
// The zero value is only valid when the ColorTarget's
// LoadOp is not LClear. The ClearFloat32, ClearUint32
// and ClearInt32 functions create valid clear values.
type ClearColor struct {
	ClearFmt
	Value [4]int32
}

// ClearColor for PixelFmt constants whose name ends
// in 'un', 'n', 'sRGB' and 'f'.
func ClearFloat32(r, g, b, a float32) ClearColor {
	return ClearColor{
		ClearFmt: CFloat,
		Value: [4]int32{
			*(*int32)(unsafe.Pointer(&r)),
			*(*int32)(unsafe.Pointer(&g)),
			*(*int32)(unsafe.Pointer(&b)),
			*(*int32)(unsafe.Pointer(&a)),
		},
	}
}

// ClearColor for PixelFmt constants whose name ends
// in 'ui'.
func ClearUint32(r, g, b, a uint32) ClearColor {
	return ClearColor{
		ClearFmt: CUint,
		Value:    [4]int32{int32(r), int32(g), int32(b), int32(a)},
	}
}

// ClearColor for PixelFmt constants whose name ends
// in 'i'.
func ClearInt32(r, g, b, a int32) ClearColor {
	return ClearColor{
		ClearFmt: CInt,
		Value:    [4]int32{r, g, b, a},
	}
}

// ColorTarget describes a single color attachment to use
// as render target in a render pass.
type ColorTarget struct {
	Color   ImageView
	Resolve ImageView
	Load    LoadOp
	Store   StoreOp
	Clear   ClearColor
}

// DSTarget describes a depth/stencil attachment to use
// as render target in a render pass.
type DSTarget struct {
	DS      ImageView
	Resolve ImageView
	LoadD   LoadOp
	StoreD  StoreOp
	LoadS   LoadOp
	StoreS  StoreOp
	ClearD  float32
	ClearS  uint32
	DSRead  bool
}

// BufferCopy describes the parameters of a copy command
// that copies data from one buffer to another.
type BufferCopy struct {
	From    Buffer
	FromOff int64
	To      Buffer
	ToOff   int64
	Size    int64
}

// ImageCopy describes the parameters of a copy command
// that copies data from one image to another.
type ImageCopy struct {
	From      Image
	FromOff   Off3D
	FromLayer int
	FromLevel int
	To        Image
	ToOff     Off3D
	ToLayer   int
	ToLevel   int
	Size      Dim3D
	Layers    int
}

// BufImgCopy describes the parameters of a copy command
// that copies data between a buffer and an image.
// BufOff must be aligned to 512 bytes.
// Stride[0] must be aligned to 256 bytes.
type BufImgCopy struct {
	Buf    Buffer
	BufOff int64
	// RowStrd and SlcStrd specify the addressing
	// of image data in the buffer.
	// RowStrd is the row length, in pixels.
	// SlcStrd is the number of rows between
	// adjacent slices.
	RowStrd int
	SlcStrd int
	Img     Image
	ImgOff  Off3D
	Layer   int
	Level   int
	// Plane selects the plane/aspect of the image.
	// The first plane/aspect is 0. For combined
	// depth/stencil formats, depth is Plane 0 and
	// stencil is Plane 1.
	Plane  int
	Size   Dim3D
	Layers int
}

// Sync is the type of a synchronization scope.
type Sync int

// Synchronization scopes.
const (
	// Inputs of vertex stage.
	SVertexInput Sync = 1 << iota
	// Vertex stage.
	SVertexShading
	// Fragment stage.
	SFragmentShading
	// Depth/stencil output.
	SDSOutput
	// Color output.
	SColorOutput
	// Multisample resolve.
	SResolve
	// All graphics stages.
	SGraphics
	// Compute stage.
	SComputeShading
	// Copy commands.
	SCopy
	// Everything.
	SAll
	// Nothing.
	SNone Sync = 0
)

// Access is the type of a memory access scope.
type Access int

// Memory access scopes.
const (
	// Read from vertex buffer.
	AVertexBufRead Access = 1 << iota
	// Read from index buffer.
	AIndexBufRead
	// Read/sample in a shader.
	AShaderRead
	// Write in a shader.
	AShaderWrite
	// Read from color render target.
	AColorRead
	// Write to color render target.
	AColorWrite
	// Read from depth/stencil render target.
	ADSRead
	// Write to depth/stencil render target.
	ADSWrite
	// Read from resolve source or destination.
	AResolveRead
	// Write to resolve source or destination.
	AResolveWrite
	// Read in a copy command.
	ACopyRead
	// Write in a copy command.
	ACopyWrite
	// Any kind of read.
	ARead
	// Any kind of write.
	AWrite
	// No access.
	ANone Access = 0
)

// Layout is the type of an image layout.
type Layout int

// Image layouts.
const (
	// The initial layout of all image views.
	// It can be used as the layout transitioned from
	// when contents need not be preserved.
	LUndefined Layout = iota
	// Shader storage.
	// Every image view in a descriptor heap that will
	// be written in shaders must be transitioned to
	// this layout.
	LShaderStore
	// Shader read/sample.
	// Every image view in a descriptor heap that will
	// be read/sampled in shaders must be transitioned
	// to this layout.
	LShaderRead
	// Color render target.
	// Every ColorTarget view that is used in a
	// render pass must be transitioned to this
	// layout.
	LColorTarget
	// Depth/stencil render target.
	// Every DSTarget view that is used in a render
	// pass must be transitioned to this layout,
	// unless DSTarget.DSRead is true, in which case
	// DSTarget.DS must be transitioned to LDSRead
	// instead.
	LDSTarget
	// Read-only depth/stencil.
	// This layout must be used instead of LDSTarget
	// when DSTarget.DSRead is set to true. Note that
	// this only pertains to the DSTarget.DS view -
	// DSTarget.Resolve, if present, must be in the
	// LDSTarget layout.
	LDSRead
	// Source of a copy command.
	// The source image range defined in ImageCopy and
	// BufImgCopy must be in this layout when used in
	// CopyImage and CopyImgToBuf commands.
	LCopySrc
	// Destination of a copy command.
	// The destination image range defined in ImageCopy
	// and BufImgCopy must be in this layout when used
	// in CopyImage and CopyBufToImg commands.
	LCopyDst
	// Presentation.
	// A swapchain's view must be transitioned to this
	// layout prior to presentation.
	LPresent
)

// Barrier represents a synchronization barrier.
type Barrier struct {
	SyncBefore   Sync
	SyncAfter    Sync
	AccessBefore Access
	AccessAfter  Access
}

// Transition represents a layout transition on a
// specific image subresource.
type Transition struct {
	Barrier
	LayoutBefore Layout
	LayoutAfter  Layout
	Img          Image
	Layer        int
	Layers       int
	Level        int
	Levels       int
}

// ShaderCode is the interface that defines a shader binary
// for execution in a programmable pipeline stage.
type ShaderCode interface {
	Destroyer
}

// ShaderFunc specifies a function within a shader binary.
type ShaderFunc struct {
	Code ShaderCode
	Name string
}

// Stage is a mask of programmable stages.
type Stage int

// Stages.
const (
	SVertex Stage = 1 << iota
	SFragment
	SCompute
)

// DescType is the type of a descriptor.
type DescType int

// Descriptor types.
const (
	// Read/write buffer.
	DBuffer DescType = iota
	// Read/write image.
	DImage
	// Constant buffer.
	DConstant
	// Sampled texture.
	DTexture
	// Texture sampler.
	DSampler
)

// Descriptor describes data for use in shaders.
type Descriptor struct {
	Type   DescType
	Stages Stage
	Nr     int
	Len    int
}

// DescHeap is the interface that defines a set of descriptors
// for use in programmable pipeline stages.
type DescHeap interface {
	Destroyer

	// New creates enough storage for n copies of each
	// descriptor.
	// All copies from a previous call to New are invalidated,
	// unless n is the same as the current Len value, in
	// which case it is a no-op.
	// Calling New(0) frees all storage.
	New(n int) error

	// SetBuffer updates the buffer ranges referred by the
	// given descriptor of the given heap copy.
	// The descriptor must be of type DBuffer or DConstant.
	// Buffer ranges must be aligned to 256 bytes.
	SetBuffer(cpy, nr, start int, buf []Buffer, off, size []int64)

	// SetImage updates the image views referred by the
	// given descriptor of the given heap copy.
	// The descriptor must be of type DImage or DTexture.
	// plane is used to select the plane/aspect for each
	// view in iv. It is allowed to be nil if all views
	// have a single plane/aspect.
	SetImage(cpy, nr, start int, iv []ImageView, plane []int)

	// SetSampler updates the samplers referred by the
	// given descriptor of the given heap copy.
	// The descriptor must be of type DSampler.
	SetSampler(cpy, nr, start int, splr []Sampler)

	// Len returns the number of heap copies created
	// by New.
	Len() int
}

// DescTable is the interface that defines the bindings
// between a number of descriptor heaps and the shaders
// in a pipeline.
type DescTable interface {
	Destroyer

	// Heap returns the descriptor heap at index idx.
	// Heap indices match those of the slice that is
	// used to create the table.
	// It panics if idx is out of bounds.
	Heap(idx int) DescHeap

	// Len returns the number of descriptor heaps in
	// the table.
	Len() int
}

// VertexFmt describes the format of a vertex input.
type VertexFmt int

// Vertex formats.
const (
	// Signed 8-bit integer, 1-4 components.
	Int8   VertexFmt = iota | 1<<16
	Int8x2 VertexFmt = iota | 2<<16
	Int8x3 VertexFmt = iota | 3<<16
	Int8x4 VertexFmt = iota | 4<<16
	// Signed 16-bit integer, 1-4 components.
	Int16   VertexFmt = iota | 2<<16
	Int16x2 VertexFmt = iota | 4<<16
	Int16x3 VertexFmt = iota | 6<<16
	Int16x4 VertexFmt = iota | 8<<16
	// Signed 32-bit integer, 1-4 components.
	Int32   VertexFmt = iota | 4<<16
	Int32x2 VertexFmt = iota | 8<<16
	Int32x3 VertexFmt = iota | 12<<16
	Int32x4 VertexFmt = iota | 16<<16
	// Unsigned 8-bit integer, 1-4 components.
	Uint8   VertexFmt = iota | 1<<16
	Uint8x2 VertexFmt = iota | 2<<16
	Uint8x3 VertexFmt = iota | 3<<16
	Uint8x4 VertexFmt = iota | 4<<16
	// Unsigned 16-bit integer, 1-4 components.
	Uint16   VertexFmt = iota | 2<<16
	Uint16x2 VertexFmt = iota | 4<<16
	Uint16x3 VertexFmt = iota | 6<<16
	Uint16x4 VertexFmt = iota | 8<<16
	// Unsigned 32-bit integer, 1-4 components.
	Uint32   VertexFmt = iota | 4<<16
	Uint32x2 VertexFmt = iota | 8<<16
	Uint32x3 VertexFmt = iota | 12<<16
	Uint32x4 VertexFmt = iota | 16<<16
	// Single precision floating-point, 1-4 components.
	Float32   VertexFmt = iota | 4<<16
	Float32x2 VertexFmt = iota | 8<<16
	Float32x3 VertexFmt = iota | 12<<16
	Float32x4 VertexFmt = iota | 16<<16
)

// Size returns the VertexFmt's size in bytes.
func (f VertexFmt) Size() int { return int(f >> 16) }

// VertexIn describes a vertex input.
// Consecutive vertices are fetched Stride bytes apart.
// Each vertex input represents a separate buffer binding;
// interleaved inputs are not supported.
// The meaning of the Nr field is shader-specific.
type VertexIn struct {
	Format VertexFmt
	Stride int
	Nr     int
}

// Topology is the type of primitive topologies,
// which determines how vertex data is assembled.
type Topology int

// Primitive topologies.
const (
	TPoint Topology = iota
	TLine
	TLnStrip
	TTriangle
	TTriStrip
)

// IndexFmt describes the format of index buffer data.
type IndexFmt int

// Index formats.
const (
	Index16 IndexFmt = 2
	Index32 IndexFmt = 4
)

// Viewport defines the bounds of a viewport.
type Viewport struct {
	X, Y, Width, Height, Znear, Zfar float32
}

// Scissor defines a scissor rectangle.
type Scissor struct {
	X, Y, Width, Height int
}

// Cullmode is the type of cull modes, which
// determines primitive culling based on triangle
// facing direction.
type CullMode int

// Cull modes.
const (
	CNone CullMode = iota
	CFront
	CBack
)

// FillMode is the type of triangle fill modes, which
// determines the final rasterization of triangles.
type FillMode int

// Triangle fill modes.
const (
	FFill FillMode = iota
	FLines
)

// RasterState defines the rasterization state of a
// graphics pipeline.
type RasterState struct {
	// Discard disables rasterization.
	Discard bool
	// Winding order is either clockwise or counter-clockwise.
	Clockwise bool
	Cull      CullMode
	Fill      FillMode
	// DepthBias enables depth bias computation.
	DepthBias bool
	BiasValue float32
	BiasSlope float32
	BiasClamp float32
}

// CmpFunc is the type of comparison functions.
type CmpFunc int

// Comparison functions.
const (
	CNever CmpFunc = iota
	CLess
	CEqual
	CLessEqual
	CGreater
	CNotEqual
	CGreaterEqual
	CAlways
)

// StencilOp is the type of stencil operations.
type StencilOp int

// Stencil operations.
const (
	SKeep StencilOp = iota
	SZero
	SReplace
	SIncClamp
	SDecClamp
	SInvert
	SIncWrap
	SDecWrap
)

// StencilT defines stencil test parameters for the
// depth/stencil state of a graphics pipeline.
type StencilT struct {
	FailS     StencilOp
	FailD     StencilOp
	Pass      StencilOp
	ReadMask  uint32
	WriteMask uint32
	Cmp       CmpFunc
}

// DSState defines the depth/stencil state of a
// graphics pipeline.
type DSState struct {
	// DepthTest enables the depth test.
	DepthTest bool
	// DepthWrite enables depth writes.
	DepthWrite bool
	DepthCmp   CmpFunc
	// StencilTest enables the stencil test.
	StencilTest bool
	Front       StencilT
	Back        StencilT
}

// ColorMask is the type of a color write mask.
type ColorMask int

// Color write masks.
const (
	CRed ColorMask = 1 << iota
	CGreen
	CBlue
	CAlpha
	// Write to all channels.
	CAll ColorMask = 1<<iota - 1
)

// BlendFac is the type of blend factors.
type BlendFac int

// Blend factors.
const (
	BZero BlendFac = iota
	BOne
	BSrcColor
	BInvSrcColor
	BSrcAlpha
	BInvSrcAlpha
	BDstColor
	BInvDstColor
	BDstAlpha
	BInvDstAlpha
	BSrcAlphaSaturated
	BBlendColor
	BInvBlendColor
)

// BlendOp is the type of blend operations.
type BlendOp int

// Blend operations.
const (
	BAdd BlendOp = iota
	BSubtract
	BRevSubtract
	BMin
	BMax
)

// ColorBlend defines a render target's blend parameters
// for the color blend state of a graphics pipeline.
type ColorBlend struct {
	// Blend enables blending.
	Blend bool
	// WriteMask specifies which color channels to write.
	// If blending is not enabled, the incoming samples
	// are written unmodified to the specified channels.
	WriteMask ColorMask
	SrcFacRGB BlendFac
	DstFacRGB BlendFac
	OpRGB     BlendOp
	SrcFacA   BlendFac
	DstFacA   BlendFac
	OpA       BlendOp
}

// BlendState defines the color blend state of a
// graphics pipeline.
type BlendState struct {
	// IndependentBlend enables each render target to use
	// different blend parameters.
	IndependentBlend bool
	// Color contains color blend parameters for each
	// render target. If IndependentBlend is false,
	// only Color[0] is used. Otherwise, it uses one
	// element per color render target.
	Color []ColorBlend
}

// GraphState defines the combination of programmable and
// fixed stages of a graphics pipeline.
// Graphics pipelines are created from graphics states.
// The ColorFmt and DSFmt fields define the valid usage
// for a graphics pipeline - it must only be used with
// render passes whose views match these formats.
type GraphState struct {
	VertFunc ShaderFunc
	FragFunc ShaderFunc
	Desc     DescTable
	Input    []VertexIn
	Topology Topology
	Raster   RasterState
	Samples  int
	DS       DSState
	Blend    BlendState
	ColorFmt []PixelFmt
	DSFmt    PixelFmt
}

// CompState defines the state of a compute pipeline.
// Compute pipelines are created from compute states.
// The state is comprised of a single compute shader and a
// descriptor table describing the resources accessible to
// this shader.
type CompState struct {
	Func ShaderFunc
	Desc DescTable
}

// Pipeline is the interface that defines a GPU pipeline.
type Pipeline interface {
	Destroyer
}

// Usage is a mask indicating valid uses for a resource.
type Usage int

// Usage flags for Buffer and Image.
const (
	// The resource can be copied from.
	UCopySrc Usage = 1 << iota
	// The resource can be copied to.
	UCopyDst
	// The resource can be read in shaders.
	UShaderRead
	// The resource can be written in shaders.
	UShaderWrite
	// The resource can provide constant data for shaders.
	// Valid only for Buffer.
	UShaderConst
	// The resource can be sampled in shaders.
	// Valid only for Image.
	UShaderSample
	// The resource can provide vertex data for draw calls.
	// Valid only for Buffer.
	UVertexData
	// The resource can provide index data for draw calls.
	// Valid only for Buffer.
	UIndexData
	// The resource can be used as render target.
	// Valid only for Image.
	URenderTarget
	// The resource can be used for any purpose.
	UGeneric Usage = 1<<iota - 1
)

// Buffer is the interface that defines a GPU buffer.
// The size of the buffer is fixed. When a larger buffer
// is necessary, a new one must be created and the data
// must be copied explicitly.
type Buffer interface {
	Destroyer

	// Visible returns whether the buffer is host visible.
	// Non-visible memory cannot be accessed by the CPU.
	Visible() bool

	// Bytes returns a slice of length Cap referring to the
	// underlying data. If the buffer is not host visible,
	// it returns nil instead.
	// The slice is valid for the lifetime of the buffer.
	// TODO: Consider replacing this with Map/Unmap.
	Bytes() []byte

	// Cap returns the capacity of the buffer in bytes,
	// which may be greater than the size requested during
	// buffer creation.
	// This value is immutable.
	Cap() int64
}

// PixelFmt describes the format of a pixel.
type PixelFmt int

// IsInternal returns whether f is an internal format.
// Internal formats are represented by negative values.
// Clients must not create images using such formats.
func (f PixelFmt) IsInternal() bool { return f < 0 }

// Pixel formats.
const (
	FInvalid PixelFmt = iota
	// Color, 8-bit channels.
	RGBA8un   PixelFmt = iota | 4<<12 | 4<<20 | fColorf
	RGBA8n    PixelFmt = iota | 4<<12 | 4<<20 | fColorf
	RGBA8ui   PixelFmt = iota | 4<<12 | 4<<20 | fColori
	RGBA8i    PixelFmt = iota | 4<<12 | 4<<20 | fColori
	RGBA8sRGB PixelFmt = iota | 4<<12 | 4<<20 | fColorf
	BGRA8un   PixelFmt = iota | 4<<12 | 4<<20 | fColorf
	BGRA8sRGB PixelFmt = iota | 4<<12 | 4<<20 | fColorf
	RG8un     PixelFmt = iota | 2<<12 | 2<<20 | fColorf
	RG8n      PixelFmt = iota | 2<<12 | 2<<20 | fColorf
	RG8ui     PixelFmt = iota | 2<<12 | 2<<20 | fColori
	RG8i      PixelFmt = iota | 2<<12 | 2<<20 | fColori
	R8un      PixelFmt = iota | 1<<12 | 1<<20 | fColorf
	R8n       PixelFmt = iota | 1<<12 | 1<<20 | fColorf
	R8ui      PixelFmt = iota | 1<<12 | 1<<20 | fColori
	R8i       PixelFmt = iota | 1<<12 | 1<<20 | fColori
	// Color, 16-bit channels.
	RGBA16f  PixelFmt = iota | 8<<12 | 4<<20 | fColorf
	RGBA16ui PixelFmt = iota | 8<<12 | 4<<20 | fColori
	RGBA16i  PixelFmt = iota | 8<<12 | 4<<20 | fColori
	RG16f    PixelFmt = iota | 4<<12 | 2<<20 | fColorf
	RG16ui   PixelFmt = iota | 4<<12 | 2<<20 | fColori
	RG16i    PixelFmt = iota | 4<<12 | 2<<20 | fColori
	R16f     PixelFmt = iota | 2<<12 | 1<<20 | fColorf
	R16ui    PixelFmt = iota | 2<<12 | 1<<20 | fColori
	R16i     PixelFmt = iota | 2<<12 | 1<<20 | fColori
	// Color, 32-bit channels.
	RGBA32f  PixelFmt = iota | 16<<12 | 4<<20 | fColorf
	RGBA32ui PixelFmt = iota | 16<<12 | 4<<20 | fColori
	RGBA32i  PixelFmt = iota | 16<<12 | 4<<20 | fColori
	RG32f    PixelFmt = iota | 8<<12 | 2<<20 | fColorf
	RG32ui   PixelFmt = iota | 8<<12 | 2<<20 | fColori
	RG32i    PixelFmt = iota | 8<<12 | 2<<20 | fColori
	R32f     PixelFmt = iota | 4<<12 | 1<<20 | fColorf
	R32ui    PixelFmt = iota | 4<<12 | 1<<20 | fColori
	R32i     PixelFmt = iota | 4<<12 | 1<<20 | fColori
	// Depth/Stencil.
	D16un     PixelFmt = iota | 2<<12 | 1<<20 | fDepth
	D32f      PixelFmt = iota | 4<<12 | 1<<20 | fDepth
	S8ui      PixelFmt = iota | 1<<12 | 1<<20 | fStencil
	D24unS8ui PixelFmt = iota | 4<<12 | 2<<20 | fDS
	D32fS8ui  PixelFmt = iota | 5<<12 | 2<<20 | fDS

	fColorf  = 1 << 24
	fColori  = 2 << 24
	fDepth   = 4 << 24
	fStencil = 8 << 24
	fColor   = fColorf | fColori
	fDS      = fDepth | fStencil
)

// Size returns the PixelFmt's size in bytes.
// f must not be an internal format.
func (f PixelFmt) Size() int { return int(f >> 12 & 0xff) }

// Channels returns the number of channels in f.
// f must not be an internal format.
func (f PixelFmt) Channels() int { return int(f >> 20 & 0xf) }

// IsColor returns whether f is a color format.
// f must not be an internal format.
func (f PixelFmt) IsColor() bool { return f&fColor != 0 }

// IsNonfloatColor returns whether f is an integer
// color format.
// f must not be an internal format.
func (f PixelFmt) IsNonfloatColor() bool { return f&fColori != 0 }

// IsDS returns whether f is a depth/stencil format.
// f must not be an internal format.
func (f PixelFmt) IsDS() (depth bool, stencil bool) {
	switch f & fDS {
	case fDS:
		return true, true
	case fDepth:
		return true, false
	case fStencil:
		return false, true
	default:
		return false, false
	}
}

// Image is the interface that defines a GPU image.
// The dimensionality of the image is derived from the size
// it was created with:
//
//	Dim3D{Width: >= 1, Height: 0, Depth: 0} is 1D.
//	Dim3D{Width: >= 1, Height: >= 1, Depth: 0} is 2D.
//	Dim3D{Width: >= 1, Height: >= 1, Depth: >= 1} is 3D.
//
// When creating an image view, the dimensionality must
// match that of the view's type.
//
// Direct access to image memory is not provided, so copying
// data from the CPU to an image resource requires the use
// of a staging buffer.
type Image interface {
	Destroyer

	// NewView creates a new image view.
	// Image views represent a typed view of image storage.
	// Its type must be valid according to the image from
	// which it is created and the parameters given when
	// calling this method (e.g., creating a view of type
	// IView3D from a 2D image is not allowed, and neither
	// is IView2DArray if the view will contain a single
	// layer of the image).
	// All views created from a given image must be
	// destroyed before the image itself is destroyed.
	NewView(typ ViewType, layer, layers, level, levels int) (ImageView, error)
}

// ViewType is the type of a resource view.
type ViewType int

// View types.
const (
	IView1D ViewType = iota
	IView2D
	IView3D
	IViewCube
	IView1DArray
	IView2DArray
	IViewCubeArray
	IView2DMS
	IView2DMSArray
)

// ImageView is the interface that defines a typed view of
// an Image resource.
type ImageView interface {
	Destroyer

	// Image returns the image from which the view
	// was created.
	// This value is immutable for the lifetime of
	// the ImageView.
	Image() Image
}

// Filter is the type of sampler filters.
type Filter int

// Filters.
const (
	FNearest Filter = iota
	FLinear
	// FNoMipmap forces mip level 0 to be used.
	// It is only valid as the mip filter of a sampler.
	FNoMipmap
)

// AddrMode is the type of sampler address modes.
type AddrMode int

// Address modes.
const (
	AWrap AddrMode = iota
	AMirror
	AClamp
)

// Sampler is the interface that defines an image sampler.
type Sampler interface {
	Destroyer
}

// Sampling describes image sampler state.
type Sampling struct {
	Min      Filter
	Mag      Filter
	Mipmap   Filter
	AddrU    AddrMode
	AddrV    AddrMode
	AddrW    AddrMode
	MaxAniso int
	DoCmp    bool
	Cmp      CmpFunc
	MinLOD   float32
	MaxLOD   float32
}

// Limits describes implementation limits.
// These may vary across drivers and devices.
type Limits struct {
	// Maximum width of 1D images.
	MaxImage1D int
	// Maximum width and height of 2D images.
	MaxImage2D int
	// Maximum width and height of cube images.
	MaxImageCube int
	// Maximum width, height and depth of 3D images.
	MaxImage3D int
	// Maximum number of layers in an image.
	MaxLayers int

	// Maximum number of descriptor heaps in a
	// descriptor table.
	MaxDescHeaps int
	// Maximum number of buffer descriptors in a
	// descriptor table.
	MaxDescBuffer int
	// Maximum number of image descriptors in a
	// descriptor table.
	MaxDescImage int
	// Maximum number of constant descriptors in a
	// descriptor table.
	MaxDescConstant int
	// Maximum number of texture descriptors in a
	// descriptor table.
	MaxDescTexture int
	// Maximum number of sampler descriptors in a
	// descriptor table.
	MaxDescSampler int
	// Maximum range of buffer descriptors.
	MaxDescBufferRange int64
	// Maximum range of constant descriptors.
	MaxDescConstantRange int64

	// Maximum number of color render targets in
	// a render pass.
	MaxColorTargets int
	// Maximum width/height in a render pass.
	MaxRenderSize [2]int
	// Maximum number of layers in a render pass.
	MaxRenderLayers int
	// Maximum size of a point primitive.
	MaxPointSize float32

	// Maximum number of vertex inputs in a
	// vertex shader.
	MaxVertexIn int
	// Maximum number of fragment inputs in a
	// fragment shader.
	MaxFragmentIn int

	// Maximum dispatch count.
	MaxDispatch [3]int
	// Maximum size of a work group.
	MaxWorkGroupSize [3]int
	// Maximum number of invocations in a
	// work group.
	MaxInvocations int
}

// Features describes available features.
// These may vary across drivers and devices.
type Features struct {
	// Whether BlendState.IndependentBlend
	// is supported.
	IndependentBlend bool
	// Whether RasterState.BiasClamp is supported.
	DepthBiasClamp bool
	// Whether the FLines FillMode is supported.
	FLines bool
	// Whether ImageView of type IViewCubeArray
	// is supported.
	CubeArray bool
}
