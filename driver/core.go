// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package driver

// GPU is the main interface to an underlying driver
// implementation.
// It is used to create other types and to execute commands.
// A GPU is obtained from a call to Driver.Open.
type GPU interface {
	// Driver returns the Driver that owns the GPU.
	Driver() Driver

	// Commit commits a batch of command buffers to the GPU
	// for execution.
	// Wait operations defined in a command buffer apply to
	// the batch as a whole, so the order of command buffers
	// in cb is meaningful.
	// This method sends the result to ch when all commands
	// complete execution. Command buffers in cb cannot be
	// used for recording until then.
	Commit(cb []CmdBuffer, ch chan<- error)

	// NewCmdBuffer creates a new command buffer.
	NewCmdBuffer() (CmdBuffer, error)

	// NewRenderPass creates a new render pass.
	NewRenderPass(att []Attachment, sub []Subpass) (RenderPass, error)

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
}

// Destroyer is the interface that wraps the Destroy method.
// Types that implement this interface may allocate external
// memory that is not managed by GC, so Destroy must be
// called explicitly to ensure such memory is deallocated.
type Destroyer interface {
	Destroy()
}

// CmdBuffer is the interface that defines a command buffer.
// Commands are recorded into command buffers and later
// committed to the GPU for execution. Recording is separate
// into logical blocks containing either rendering, compute
// or copy commands. Multiple logical blocks can be recorded
// into a single command buffer. The usage is as follows:
// First, call Begin to prepare the command buffer for
// recording. Then, if it succeeds:
//
// To record commands for a render pass:
// 	1. call BeginPass
// 	2. call Set* methods to configure rendering state
// 	3. call Draw* commands
// 	4. call NextSubpass (if using multiple subpasses)
// 	5. repeat 2-4 as needed
// 	6. call EndPass
//
// To record compute commands:
//	1. call BeginWork
//	2. call Set* methods to configure compute state
//	3. call Dispatch commands
//	4. repeat 2-3 as needed
//	5. call EndWork
//
// To record copy commands:
//	1. call BeginBlit
//	2. call Copy*/Fill commands
//	3. call EndBlit
//
// Finally, call End and, if it succeeds, GPU.Commit.
// Note that Begin* commands must not be nested, and
// must always be ended before another call to Begin*
// and prior to the final End call.
type CmdBuffer interface {
	Destroyer

	// Begin prepares the command buffer for recording.
	// This method must be called before any command
	// is recorded in the command buffer. It needs to
	// be called again if the command buffer is
	// executed or reset.
	Begin() error

	// BeginPass begins the first subpass of a given
	// render pass.
	// Draw commands within a subpass may run in
	// parallel. The behavior of draw commands across
	// different subpasses is defined on render pass
	// creation.
	BeginPass(pass RenderPass, fb Framebuf, clear []ClearValue)

	// NextSubpass ends the current subpass and begins
	// the next one.
	// It must not be called in the last subpass.
	NextSubpass()

	// EndPass ends the current render pass.
	EndPass()

	// BeginWork begins compute work.
	// If wait is set, compute work only starts when
	// all previous commands recorded in the same
	// command buffer are done executing.
	// Dispatch commands may run in parallel.
	BeginWork(wait bool)

	// EndWork ends the current compute work.
	EndWork()

	// BeginBlit begins data transfer.
	// If wait is set, data transfer only starts when
	// all previous commands recorded in the same
	// command buffer are done executing.
	// Copy/fill commands may run in parallel.
	BeginBlit(wait bool)

	// EndBlit ends the current data transfer.
	EndBlit()

	// SetPipeline sets the pipeline.
	// There is a separate binding point for each
	// type of pipeline.
	SetPipeline(pl Pipeline)

	// SetViewport sets the bounds of one or more
	// viewports.
	SetViewport(vp []Viewport)

	// SetScissor sets the rectangles of one or more
	// viewport scissors.
	SetScissor(sciss []Scissor)

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
	Draw(vertCount, instCount, baseVert, baseInst int)

	// DrawIndexed draws indexed primitives.
	// It must only be called during a render pass.
	DrawIndexed(idxCount, instCount, baseIdx, vertOff, baseInst int)

	// Dispatch dispatches compute thread groups.
	// It must only be called during compute work.
	Dispatch(grpCountX, grpCountY, grpCountZ int)

	// CopyBuffer copies data between buffers.
	// It must only be called during data transfer.
	CopyBuffer(param *BufferCopy)

	// CopyImage copies data between images.
	// It must only be called during data transfer.
	CopyImage(param *ImageCopy)

	// CopyBufToImg copies data from a buffer to
	// an image.
	// It must only be called during data transfer.
	CopyBufToImg(param *BufImgCopy)

	// CopyImgToBuf copies data from an image to
	// a buffer.
	// It must only be called during data transfer.
	CopyImgToBuf(param *BufImgCopy)

	// Fill fills a buffer range with copies of
	// a byte value.
	// It must only be called during data transfer.
	// off and size must be aligned to 4 bytes.
	Fill(buf Buffer, off int64, value byte, size int64)

	// Barrier inserts a number of global barriers
	// in the command buffer.
	Barrier(b []Barrier)

	// Transition inserts a number of image layout
	// transitions in the command buffer.
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
	// Stride specifies the addressing of image data
	// in the buffer. It is given in pixels.
	// Stride[0] refers to the row length and Stride[1]
	// refers to the image height.
	Stride [2]int64
	Img    Image
	ImgOff Off3D
	Layer  int
	Level  int
	Size   Dim3D
	// DepthCopy selects either the depth or stencil
	// aspects to copy. It is only used if Img has a
	// combined depth/stencil format.
	DepthCopy bool
}

// Sync is the type of a synchronization scope.
type Sync int

// Synchronization scopes.
const (
	SVertexInput Sync = 1 << iota
	SVertexShading
	SFragmentShading
	SComputeShading
	SColorOutput
	SDSOutput
	SDraw
	SResolve
	SCopy
	SAll
	SNone Sync = 0
)

// Access is the type of a memory access scope.
type Access int

// Memory access scopes.
const (
	AVertexBufRead Access = 1 << iota
	AIndexBufRead
	AColorRead
	AColorWrite
	ADSRead
	ADSWrite
	AResolveRead
	AResolveWrite
	ACopyRead
	ACopyWrite
	AShaderRead
	AShaderWrite
	AAnyRead
	AAnyWrite
	ANone Access = 0
)

// Layout is the type of an image layout.
type Layout int

// Image layouts.
const (
	LUndefined Layout = iota
	LCommon
	LColorTarget
	LDSTarget
	LDSRead
	LResolveSrc
	LResolveDst
	LCopySrc
	LCopyDst
	LShaderRead
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
	IView        ImageView
}

// LoadOp is the type of an attachment's load operation.
type LoadOp int

// Load operations.
const (
	LDontCare LoadOp = iota
	LClear
	LLoad
)

// StoreOp is the type of an attachment's store operation.
type StoreOp int

// Store operations.
const (
	SDontCare StoreOp = iota
	SStore
)

// Attachment describes the configuration of a single
// render target for use in a render pass.
type Attachment struct {
	Format  PixelFmt
	Samples int
	Load    [2]LoadOp
	Store   [2]StoreOp
}

// Subpass defines a subpass of a render pass.
// Render passes are split into a number of subpasses.
// The Color, DS (depth/stencil) and MSR (multisample resolve)
// fields contain indices in the render pass' attachment list
// indicating a subset of the render targets that the subpass
// will use. The Wait field controls whether or not the
// subpass stalls waiting for previous work to finish.
type Subpass struct {
	Color []int
	DS    int
	MSR   []int
	Wait  bool
}

// RenderPass is the interface that defines a render pass
// into which draw commands operate.
type RenderPass interface {
	Destroyer

	// NewFB creates a new framebuffer.
	// Each image view in iv correspond to the render pass'
	// attachment of same index. A view's pixel format and
	// sample count must match the attachment's. Views whose
	// image was not created with URenderTarget as a valid
	// usage cannot be used in a framebuffer.
	// All framebuffers created from a given render pass
	// must be destroyed before the render pass itself
	// is destroyed.
	NewFB(iv []ImageView, width, height, layers int) (Framebuf, error)
}

// Framebuf is the interface that defines the render targets
// of a render pass.
type Framebuf interface {
	Destroyer
}

// ClearValue defines clear values for color or depth/stencil
// aspects of a render target.
type ClearValue struct {
	Color   [4]float32
	Depth   float32
	Stencil uint32
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
	// unless n is the same as the current Count value, in
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
	SetImage(cpy, nr, start int, iv []ImageView)

	// SetSampler updates the samplers referred by the
	// given descriptor of the given heap copy.
	// The descriptor must be of type DSampler.
	SetSampler(cpy, nr, start int, splr []Sampler)

	// Count returns the number of heap copies created
	// by New.
	Count() int
}

// DescTable is the interface that defines the bindings
// between a number of descriptor heaps and the shaders
// in a pipeline.
type DescTable interface {
	Destroyer
}

// VertexFmt describes the format of a vertex input.
type VertexFmt int

// Vertex formats.
const (
	// Signed 8-bit integer, 1-4 components.
	Int8 VertexFmt = iota
	Int8x2
	Int8x3
	Int8x4
	// Signed 16-bit integer, 1-4 components.
	Int16
	Int16x2
	Int16x3
	Int16x4
	// Signed 32-bit integer, 1-4 components.
	Int32
	Int32x2
	Int32x3
	Int32x4
	// Unsigned 8-bit integer, 1-4 components.
	UInt8
	UInt8x2
	UInt8x3
	UInt8x4
	// Unsigned 16-bit integer, 1-4 components.
	UInt16
	UInt16x2
	UInt16x3
	UInt16x4
	// Unsigned 32-bit integer, 1-4 components.
	UInt32
	UInt32x2
	UInt32x3
	UInt32x4
	// Single precision floating-point, 1-4 components.
	Float32
	Float32x2
	Float32x3
	Float32x4
)

// VertexIn describes a vertex input.
// Consecutive vertices are fetched Stride bytes apart.
// Each vertex input represents a separate buffer binding,
// interleaved inputs are not supported.
// The meaning of the Nr and Name fields is shader-specific.
type VertexIn struct {
	Format VertexFmt
	Stride int
	Nr     int
	Name   string
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
	DSFail    [2]StencilOp
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

// BlenOp is the type of blend operations.
type BlendOp int

// Blend operations.
const (
	BAdd BlendOp = iota
	BSubtract
	BRevSubtract
	BMin
	BMax
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

// ColorBlend defines a render target's blend parameters
// for the color blend state of a graphics pipeline.
type ColorBlend struct {
	// Blend enables blending.
	Blend bool
	// WriteMask specifies which color channels to write.
	// If blending is not enabled, the incoming samples
	// are written unmodified to the specified channels.
	WriteMask ColorMask
	// In the arrays that follows, [0] is for color and
	// [1] is for alpha.
	Op     [2]BlendOp
	SrcFac [2]BlendFac
	DstFac [2]BlendFac
}

// BlendState defines the color blend state of a
// graphics pipeline.
type BlendState struct {
	// IndependentBlend enables each render target to use
	// different blend parameters.
	IndependentBlend bool
	// Color contains color blend parameters for each
	// render target. If IndependentBlend is false,
	// only Color[0] is used.
	Color []ColorBlend
}

// GraphState defines the combination of programmable and
// fixed stages of a graphics pipeline.
// Graphics pipelines are created from graphics states.
// The Pass and Subpass fields in the state define the
// valid use of a graphics pipeline - it must not be used
// outside this subpass.
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
	Pass     RenderPass
	Subpass  int
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
	// The resource can be read in shaders.
	UShaderRead Usage = 1 << iota
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
	Bytes() []byte

	// Cap returns the capacity of the buffer in bytes,
	// which may be greater than the size requested during
	// buffer creation.
	// This value is immutable.
	Cap() int64
}

// PixelFmt describes the format of a pixel.
type PixelFmt int

// Internal format bit.
// All internal formats have this bit set. Client code
// must not create images using internal formats.
const FInternal PixelFmt = 1 << 31

// IsInternal returns whether f is an internal format.
func (f PixelFmt) IsInternal() bool { return f&FInternal == FInternal }

// Pixel formats.
const (
	// Color, 8-bit channels.
	RGBA8un PixelFmt = iota
	RGBA8n
	RGBA8sRGB
	BGRA8un
	BGRA8sRGB
	RG8un
	RG8n
	R8un
	R8n
	// Color, 16-bit channels.
	RGBA16f
	RG16f
	R16f
	// Color, 32-bit channels.
	RGBA32f
	RG32f
	R32f
	// Depth/Stencil.
	D16un
	D32f
	S8ui
	D24unS8ui
	D32fS8ui
)

// Dim3D is a three-dimensional size.
type Dim3D struct {
	Width, Height, Depth int
}

// Off3D is a three-dimensional offset.
type Off3D struct {
	X, Y, Z int
}

// Image is the interface that defines a GPU image.
// Direct access to image memory is not provided, so copying
// data from the CPU to an image resource requires the use
// of a staging buffer.
type Image interface {
	Destroyer

	// NewView creates a new image view.
	// Image views represent a typed view of image storage.
	// Its type must be valid according to the image from
	// which it is created and the parameters given when
	// calling this method (e.g, creating a view of 3D type
	// from a 2D image is not allowed, and neither is a
	// view of array type if the view is created from a
	// single layer).
	// All views created from a given image must be
	// detroyed before the image itself is destroyed.
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
	MaxDBuffer int
	// Maximum number of image descriptors in a
	// descriptor table.
	MaxDImage int
	// Maximum number of constant descriptors in a
	// descriptor table.
	MaxDConstant int
	// Maximum number of texture descriptors in a
	// descriptor table.
	MaxDTexture int
	// Maximum number of sampler descriptors in a
	// descriptor table.
	MaxDSampler int
	// Maximum range of buffer descriptors.
	MaxDBufferRange int64
	// Maximum range of constant descriptors.
	MaxDConstantRange int64

	// Maximum number of color render targets in a
	// subpass of a render pass.
	MaxColorTargets int
	// Maximum width/height for a framebuffer.
	MaxFBSize [2]int
	// Maximum number of layers in a framebuffer.
	MaxFBLayers int
	// Maximum size of a point primitive.
	MaxPointSize float32
	// Maximum number of viewports.
	MaxViewports int

	// Maximum number of vertex inputs in a
	// vertex shader.
	MaxVertexIn int
	// Maximum number of fragment inputs in a
	// fragment shader.
	MaxFragmentIn int

	// Maximum dipatch count.
	MaxDispatch [3]int
}
