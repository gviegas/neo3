// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

// #include <stdlib.h>
// #include <proc.h>
import "C"

import (
	"errors"
	"unsafe"

	"github.com/gviegas/scene/driver"
)

// pipeline implements driver.Pipeline.
type pipeline struct {
	d  *Driver
	pl C.VkPipeline
}

// NewPipeline creates a new pipeline.
func (d *Driver) NewPipeline(state any) (driver.Pipeline, error) {
	switch t := state.(type) {
	case *driver.GraphState:
		return d.newGraphics(t)
	case *driver.CompState:
		return d.newCompute(t)
	}
	return nil, errors.New("unknown pipeline state type")
}

// newGraphics creates a new graphics pipeline.
func (d *Driver) newGraphics(gs *driver.GraphState) (driver.Pipeline, error) {
	p := &pipeline{d: d}
	var layout C.VkPipelineLayout
	if gs.Desc == nil {
		// We need a valid pipeline layout, so create a temporary
		// descTable for its layout and destroy it at the end.
		if desc, err := d.NewDescTable(nil); err != nil {
			return nil, err
		} else {
			defer desc.Destroy()
			layout = desc.(*descTable).layout
		}
	} else {
		layout = gs.Desc.(*descTable).layout
	}
	info := C.VkGraphicsPipelineCreateInfo{
		sType:             C.VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		layout:            layout,
		renderPass:        gs.Pass.(*renderPass).pass,
		subpass:           C.uint32_t(gs.Subpass),
		basePipelineIndex: -1,
	}
	free := [...]func(){
		setGraphStages(gs, &info),
		setGraphInput(gs, &info),
		setGraphIA(gs, &info),
		setGraphTess(gs, &info),
		setGraphViewport(gs, &info),
		setGraphRaster(gs, &info),
		setGraphMS(gs, &info),
		setGraphDS(gs, &info),
		setGraphBlend(gs, &info),
		setGraphDynamic(gs, &info),
	}
	// TODO: Pipeline cache.
	var cache C.VkPipelineCache
	err := checkResult(C.vkCreateGraphicsPipelines(d.dev, cache, 1, &info, nil, &p.pl))
	for _, f := range free {
		f()
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// setGraphStages sets the shader stages for graphics pipeline creation.
func setGraphStages(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	nstg := 2
	pstg := (*C.VkPipelineShaderStageCreateInfo)(C.malloc(C.size_t(nstg) * C.sizeof_VkPipelineShaderStageCreateInfo))
	*pstg = C.VkPipelineShaderStageCreateInfo{
		sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
		stage:  C.VK_SHADER_STAGE_VERTEX_BIT,
		module: gs.VertFunc.Code.(*shaderCode).mod,
		pName:  C.CString(gs.VertFunc.Name),
	}
	if gs.FragFunc.Code == nil {
		nstg--
		free = func() {
			C.free(unsafe.Pointer(pstg.pName))
			C.free(unsafe.Pointer(pstg))
		}
	} else {
		fstg := (*C.VkPipelineShaderStageCreateInfo)(unsafe.Add(unsafe.Pointer(pstg), C.sizeof_VkPipelineShaderStageCreateInfo))
		*fstg = C.VkPipelineShaderStageCreateInfo{
			sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
			stage:  C.VK_SHADER_STAGE_FRAGMENT_BIT,
			module: gs.FragFunc.Code.(*shaderCode).mod,
			pName:  C.CString(gs.FragFunc.Name),
		}
		free = func() {
			C.free(unsafe.Pointer(pstg.pName))
			C.free(unsafe.Pointer(fstg.pName))
			C.free(unsafe.Pointer(pstg))
		}
	}
	info.stageCount = C.uint32_t(nstg)
	info.pStages = pstg
	return
}

// setGraphInput sets the vertex input state for graphics pipeline creation.
func setGraphInput(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	pin := (*C.VkPipelineVertexInputStateCreateInfo)(C.malloc(C.sizeof_VkPipelineVertexInputStateCreateInfo))
	info.pVertexInputState = pin
	nin := len(gs.Input)
	if nin > 0 {
		// Because vertex input data is non-interleaved, each attribute
		// maps to a different binding number.
		// The binding corresponds to the input index.
		pbind := (*C.VkVertexInputBindingDescription)(C.malloc(C.size_t(nin) * C.sizeof_VkVertexInputBindingDescription))
		sbind := unsafe.Slice(pbind, nin)
		pattr := (*C.VkVertexInputAttributeDescription)(C.malloc(C.size_t(nin) * C.sizeof_VkVertexInputAttributeDescription))
		sattr := unsafe.Slice(pattr, nin)
		for i := range sbind {
			sbind[i] = C.VkVertexInputBindingDescription{
				binding:   C.uint32_t(i),
				stride:    C.uint32_t(gs.Input[i].Stride),
				inputRate: C.VK_VERTEX_INPUT_RATE_VERTEX,
			}
			sattr[i] = C.VkVertexInputAttributeDescription{
				location: C.uint32_t(gs.Input[i].Nr),
				binding:  C.uint32_t(i),
				format:   convVertexFmt(gs.Input[i].Format),
			}
		}
		*pin = C.VkPipelineVertexInputStateCreateInfo{
			sType:                           C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
			vertexBindingDescriptionCount:   C.uint32_t(nin),
			pVertexBindingDescriptions:      pbind,
			vertexAttributeDescriptionCount: C.uint32_t(nin),
			pVertexAttributeDescriptions:    pattr,
		}
		free = func() {
			C.free(unsafe.Pointer(pbind))
			C.free(unsafe.Pointer(pattr))
			C.free(unsafe.Pointer(pin))
		}
	} else {
		*pin = C.VkPipelineVertexInputStateCreateInfo{
			sType: C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
		}
		free = func() {
			C.free(unsafe.Pointer(pin))
		}
	}
	return
}

// setGraphIA sets the input assembly state for graphics pipeline creation.
func setGraphIA(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	pia := (*C.VkPipelineInputAssemblyStateCreateInfo)(C.malloc(C.sizeof_VkPipelineInputAssemblyStateCreateInfo))
	*pia = C.VkPipelineInputAssemblyStateCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
		topology: convTopology(gs.Topology),
	}
	info.pInputAssemblyState = pia
	return func() {
		C.free(unsafe.Pointer(pia))
	}
}

// setGraphTess sets the tessellation state for graphics pipeline creation.
func setGraphTess(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	// Tessellation is not supported currently.
	info.pTessellationState = nil
	return func() {}
}

// setGraphViewport sets the viewport state for graphics pipeline creation.
func setGraphViewport(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	// TODO: Define a field in driver.GraphState indicating the
	// number of viewports to use.
	pvp := (*C.VkPipelineViewportStateCreateInfo)(C.malloc(C.sizeof_VkPipelineViewportStateCreateInfo))
	*pvp = C.VkPipelineViewportStateCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
		viewportCount: 1,
		scissorCount:  1,
	}
	info.pViewportState = pvp
	return func() {
		C.free(unsafe.Pointer(pvp))
	}
}

// setGraphRaster sets the rasterization state for graphics pipeline creation.
func setGraphRaster(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	var frontFace C.VkFrontFace
	if gs.Raster.Clockwise {
		frontFace = C.VK_FRONT_FACE_CLOCKWISE
	} else {
		frontFace = C.VK_FRONT_FACE_COUNTER_CLOCKWISE
	}
	var depthBias C.VkBool32
	if gs.Raster.DepthBias {
		depthBias = C.VK_TRUE
	}
	prz := (*C.VkPipelineRasterizationStateCreateInfo)(C.malloc(C.sizeof_VkPipelineRasterizationStateCreateInfo))
	*prz = C.VkPipelineRasterizationStateCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
		polygonMode:             convFillMode(gs.Raster.Fill),
		cullMode:                convCullMode(gs.Raster.Cull),
		frontFace:               frontFace,
		depthBiasEnable:         depthBias,
		depthBiasConstantFactor: C.float(gs.Raster.BiasValue),
		depthBiasClamp:          C.float(gs.Raster.BiasClamp),
		depthBiasSlopeFactor:    C.float(gs.Raster.BiasSlope),
		lineWidth:               1.0,
	}
	info.pRasterizationState = prz
	return func() {
		C.free(unsafe.Pointer(prz))
	}
}

// setGraphMS sets the multisample state for graphics pipeline creation.
func setGraphMS(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	pms := (*C.VkPipelineMultisampleStateCreateInfo)(C.malloc(C.sizeof_VkPipelineMultisampleStateCreateInfo))
	*pms = C.VkPipelineMultisampleStateCreateInfo{
		sType:                C.VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
		rasterizationSamples: convSamples(gs.Samples),
	}
	info.pMultisampleState = pms
	return func() {
		C.free(unsafe.Pointer(pms))
	}
}

// setGraphDS sets the depth/stencil state for graphics pipeline creation.
func setGraphDS(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	pds := (*C.VkPipelineDepthStencilStateCreateInfo)(C.malloc(C.sizeof_VkPipelineDepthStencilStateCreateInfo))
	*pds = C.VkPipelineDepthStencilStateCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
	}
	if gs.DS.DepthTest {
		pds.depthTestEnable = C.VK_TRUE
		if gs.DS.DepthWrite {
			pds.depthWriteEnable = C.VK_TRUE
		}
		pds.depthCompareOp = convCmpFunc(gs.DS.DepthCmp)
	}
	if gs.DS.StencilTest {
		pds.stencilTestEnable = C.VK_TRUE
		pds.front = C.VkStencilOpState{
			failOp:      convStencilOp(gs.DS.Front.DSFail[1]),
			passOp:      convStencilOp(gs.DS.Front.Pass),
			depthFailOp: convStencilOp(gs.DS.Front.DSFail[0]),
			compareOp:   convCmpFunc(gs.DS.Front.Cmp),
			compareMask: C.uint32_t(gs.DS.Front.ReadMask),
			writeMask:   C.uint32_t(gs.DS.Front.WriteMask),
		}
		pds.back = C.VkStencilOpState{
			failOp:      convStencilOp(gs.DS.Back.DSFail[1]),
			passOp:      convStencilOp(gs.DS.Back.Pass),
			depthFailOp: convStencilOp(gs.DS.Back.DSFail[0]),
			compareOp:   convCmpFunc(gs.DS.Back.Cmp),
			compareMask: C.uint32_t(gs.DS.Back.ReadMask),
			writeMask:   C.uint32_t(gs.DS.Back.WriteMask),
		}
	}
	info.pDepthStencilState = pds
	return func() {
		C.free(unsafe.Pointer(pds))
	}
}

// setGraphBlend sets the color blend state for graphics pipeline creation.
func setGraphBlend(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	ncolor := gs.Pass.(*renderPass).ncolor[gs.Subpass]
	if ncolor == 0 {
		// No color attachments in the subpass.
		info.pColorBlendState = nil
		return func() {}
	}
	pba := (*C.VkPipelineColorBlendAttachmentState)(C.malloc(C.size_t(ncolor) * C.sizeof_VkPipelineColorBlendAttachmentState))
	sba := unsafe.Slice(pba, ncolor)
	if gs.Blend.IndependentBlend {
		// gs.Blend.Color contains one element for every
		// color attachment in the subpass.
		for i := range sba {
			var blend C.VkBool32
			if gs.Blend.Color[i].Blend {
				blend = C.VK_TRUE
			}
			sba[i] = C.VkPipelineColorBlendAttachmentState{
				blendEnable:         blend,
				srcColorBlendFactor: convBlendFac(gs.Blend.Color[i].SrcFac[0]),
				dstColorBlendFactor: convBlendFac(gs.Blend.Color[i].DstFac[0]),
				colorBlendOp:        convBlendOp(gs.Blend.Color[i].Op[0]),
				srcAlphaBlendFactor: convBlendFac(gs.Blend.Color[i].SrcFac[1]),
				dstAlphaBlendFactor: convBlendFac(gs.Blend.Color[i].DstFac[1]),
				alphaBlendOp:        convBlendOp(gs.Blend.Color[i].Op[1]),
				colorWriteMask:      convColorMask(gs.Blend.Color[i].WriteMask),
			}
		}
	} else {
		// gs.Blend.Color[0] contains the color blend
		// parameters to use for all color attachments
		// in the subpass.
		var blend C.VkBool32
		if gs.Blend.Color[0].Blend {
			blend = C.VK_TRUE
		}
		sba[0] = C.VkPipelineColorBlendAttachmentState{
			blendEnable:         blend,
			srcColorBlendFactor: convBlendFac(gs.Blend.Color[0].SrcFac[0]),
			dstColorBlendFactor: convBlendFac(gs.Blend.Color[0].DstFac[0]),
			colorBlendOp:        convBlendOp(gs.Blend.Color[0].Op[0]),
			srcAlphaBlendFactor: convBlendFac(gs.Blend.Color[0].SrcFac[1]),
			dstAlphaBlendFactor: convBlendFac(gs.Blend.Color[0].DstFac[1]),
			alphaBlendOp:        convBlendOp(gs.Blend.Color[0].Op[1]),
			colorWriteMask:      convColorMask(gs.Blend.Color[0].WriteMask),
		}
		for i := 1; i < ncolor; i++ {
			sba[i] = sba[0]
		}
	}
	pbs := (*C.VkPipelineColorBlendStateCreateInfo)(C.malloc(C.sizeof_VkPipelineColorBlendStateCreateInfo))
	*pbs = C.VkPipelineColorBlendStateCreateInfo{
		sType:           C.VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
		attachmentCount: C.uint32_t(ncolor),
		pAttachments:    pba,
	}
	info.pColorBlendState = pbs
	return func() {
		C.free(unsafe.Pointer(pba))
		C.free(unsafe.Pointer(pbs))
	}
}

// setGraphDynamic sets the dynamic state for graphics pipeline creation.
func setGraphDynamic(gs *driver.GraphState, info *C.VkGraphicsPipelineCreateInfo) (free func()) {
	const dmax = 4
	pd := (*C.VkDynamicState)(C.malloc(dmax * C.sizeof_VkDynamicState))
	sd := unsafe.Slice(pd, dmax)
	sd[0] = C.VK_DYNAMIC_STATE_VIEWPORT
	sd[1] = C.VK_DYNAMIC_STATE_SCISSOR
	nd := 2
	if gs.Pass.(*renderPass).ncolor[gs.Subpass] > 0 {
		sd[nd] = C.VK_DYNAMIC_STATE_BLEND_CONSTANTS
		nd++
	}
	if gs.DS.StencilTest {
		sd[nd] = C.VK_DYNAMIC_STATE_STENCIL_REFERENCE
		nd++
	}
	pdyn := (*C.VkPipelineDynamicStateCreateInfo)(C.malloc(C.sizeof_VkPipelineDynamicStateCreateInfo))
	*pdyn = C.VkPipelineDynamicStateCreateInfo{
		sType:             C.VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
		dynamicStateCount: C.uint32_t(nd),
		pDynamicStates:    pd,
	}
	info.pDynamicState = pdyn
	return func() {
		C.free(unsafe.Pointer(pd))
		C.free(unsafe.Pointer(pdyn))
	}
}

// newCompute creates a new compute pipeline.
func (d *Driver) newCompute(cs *driver.CompState) (driver.Pipeline, error) {
	p := &pipeline{d: d}
	var layout C.VkPipelineLayout
	if cs.Desc == nil {
		// Like newGraphics above.
		// This is unlikely to happen for compute however, since the
		// shader would have no resource to read from nor write to.
		if desc, err := d.NewDescTable(nil); err != nil {
			return nil, err
		} else {
			defer desc.Destroy()
			layout = desc.(*descTable).layout
		}
	} else {
		layout = cs.Desc.(*descTable).layout
	}
	info := C.VkComputePipelineCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO,
		stage: C.VkPipelineShaderStageCreateInfo{
			sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
			stage:  C.VK_SHADER_STAGE_COMPUTE_BIT,
			module: cs.Func.Code.(*shaderCode).mod,
			pName:  C.CString(cs.Func.Name),
		},
		layout:            layout,
		basePipelineIndex: -1,
	}
	defer C.free(unsafe.Pointer(info.stage.pName))
	// TODO: Pipeline cache.
	var cache C.VkPipelineCache
	err := checkResult(C.vkCreateComputePipelines(d.dev, cache, 1, &info, nil, &p.pl))
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Destroy destroys the pipeline.
func (p *pipeline) Destroy() {
	if p == nil {
		return
	}
	if p.d != nil {
		C.vkDestroyPipeline(p.d.dev, p.pl, nil)
	}
	*p = pipeline{}
}

// convVertexFmt converts from a driver.VertexFmt to a VkFormat.
func convVertexFmt(vf driver.VertexFmt) C.VkFormat {
	switch vf {
	case driver.Int8:
		return C.VK_FORMAT_R8_SINT
	case driver.Int8x2:
		return C.VK_FORMAT_R8G8_SINT
	case driver.Int8x3:
		return C.VK_FORMAT_R8G8B8_SINT
	case driver.Int8x4:
		return C.VK_FORMAT_R8G8B8A8_SINT

	case driver.Int16:
		return C.VK_FORMAT_R16_SINT
	case driver.Int16x2:
		return C.VK_FORMAT_R16G16_SINT
	case driver.Int16x3:
		return C.VK_FORMAT_R16G16B16_SINT
	case driver.Int16x4:
		return C.VK_FORMAT_R16G16B16A16_SINT

	case driver.Int32:
		return C.VK_FORMAT_R32_SINT
	case driver.Int32x2:
		return C.VK_FORMAT_R32G32_SINT
	case driver.Int32x3:
		return C.VK_FORMAT_R32G32B32_SINT
	case driver.Int32x4:
		return C.VK_FORMAT_R32G32B32A32_SINT

	case driver.UInt8:
		return C.VK_FORMAT_R8_UINT
	case driver.UInt8x2:
		return C.VK_FORMAT_R8G8_UINT
	case driver.UInt8x3:
		return C.VK_FORMAT_R8G8B8_UINT
	case driver.UInt8x4:
		return C.VK_FORMAT_R8G8B8A8_UINT

	case driver.UInt16:
		return C.VK_FORMAT_R16_UINT
	case driver.UInt16x2:
		return C.VK_FORMAT_R16G16_UINT
	case driver.UInt16x3:
		return C.VK_FORMAT_R16G16B16_UINT
	case driver.UInt16x4:
		return C.VK_FORMAT_R16G16B16A16_UINT

	case driver.UInt32:
		return C.VK_FORMAT_R32_UINT
	case driver.UInt32x2:
		return C.VK_FORMAT_R32G32_UINT
	case driver.UInt32x3:
		return C.VK_FORMAT_R32G32B32_UINT
	case driver.UInt32x4:
		return C.VK_FORMAT_R32G32B32A32_UINT

	case driver.Float32:
		return C.VK_FORMAT_R32_SFLOAT
	case driver.Float32x2:
		return C.VK_FORMAT_R32G32_SFLOAT
	case driver.Float32x3:
		return C.VK_FORMAT_R32G32B32_SFLOAT
	case driver.Float32x4:
		return C.VK_FORMAT_R32G32B32A32_SFLOAT
	}

	// Expected to be unreachable.
	return C.VK_FORMAT_UNDEFINED
}

// convTopology converts a driver.Topology to a VkPrimitiveTopology.
func convTopology(top driver.Topology) C.VkPrimitiveTopology {
	switch top {
	case driver.TPoint:
		return C.VK_PRIMITIVE_TOPOLOGY_POINT_LIST
	case driver.TLine:
		return C.VK_PRIMITIVE_TOPOLOGY_LINE_LIST
	case driver.TLnStrip:
		return C.VK_PRIMITIVE_TOPOLOGY_LINE_STRIP
	case driver.TTriangle:
		return C.VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST
	case driver.TTriStrip:
		return C.VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP
	}

	// Expected to be unreachable.
	return ^C.VkPrimitiveTopology(0)
}

// convCullMode converts a driver.CullMode to a VkCullModeFlags.
func convCullMode(cm driver.CullMode) C.VkCullModeFlags {
	switch cm {
	case driver.CNone:
		return C.VK_CULL_MODE_NONE
	case driver.CFront:
		return C.VK_CULL_MODE_FRONT_BIT
	case driver.CBack:
		return C.VK_CULL_MODE_BACK_BIT
	}

	// Expected to be unreachable.
	return ^C.VkCullModeFlags(0)
}

// convFillMode converts a driver.FillMode to a VkPolygonMode.
func convFillMode(fm driver.FillMode) C.VkPolygonMode {
	switch fm {
	case driver.FFill:
		return C.VK_POLYGON_MODE_FILL
	case driver.FLines:
		return C.VK_POLYGON_MODE_LINE
	}

	// Expected to be unreachable.
	return ^C.VkPolygonMode(0)
}

// convStencilOp converts a driver.StencilOp to a VkStencilOp.
func convStencilOp(op driver.StencilOp) C.VkStencilOp {
	switch op {
	case driver.SKeep:
		return C.VK_STENCIL_OP_KEEP
	case driver.SZero:
		return C.VK_STENCIL_OP_ZERO
	case driver.SReplace:
		return C.VK_STENCIL_OP_REPLACE
	case driver.SIncClamp:
		return C.VK_STENCIL_OP_INCREMENT_AND_CLAMP
	case driver.SDecClamp:
		return C.VK_STENCIL_OP_DECREMENT_AND_CLAMP
	case driver.SInvert:
		return C.VK_STENCIL_OP_INVERT
	case driver.SIncWrap:
		return C.VK_STENCIL_OP_INCREMENT_AND_WRAP
	case driver.SDecWrap:
		return C.VK_STENCIL_OP_DECREMENT_AND_WRAP
	}

	// Expected to be unreachable.
	return ^C.VkStencilOp(0)
}

// convBlendOp converts a driver.BlendOp to a VkBlendOp.
func convBlendOp(op driver.BlendOp) C.VkBlendOp {
	switch op {
	case driver.BAdd:
		return C.VK_BLEND_OP_ADD
	case driver.BSubtract:
		return C.VK_BLEND_OP_SUBTRACT
	case driver.BRevSubtract:
		return C.VK_BLEND_OP_REVERSE_SUBTRACT
	case driver.BMin:
		return C.VK_BLEND_OP_MIN
	case driver.BMax:
		return C.VK_BLEND_OP_MAX
	}

	// Expected to be unreachable.
	return ^C.VkBlendOp(0)
}

// convBlendFac converts a driver.BlendFac to a VkBlendFactor.
func convBlendFac(fac driver.BlendFac) C.VkBlendFactor {
	switch fac {
	case driver.BZero:
		return C.VK_BLEND_FACTOR_ZERO
	case driver.BOne:
		return C.VK_BLEND_FACTOR_ONE
	case driver.BSrcColor:
		return C.VK_BLEND_FACTOR_SRC_COLOR
	case driver.BInvSrcColor:
		return C.VK_BLEND_FACTOR_ONE_MINUS_SRC_COLOR
	case driver.BSrcAlpha:
		return C.VK_BLEND_FACTOR_SRC_ALPHA
	case driver.BInvSrcAlpha:
		return C.VK_BLEND_FACTOR_ONE_MINUS_SRC_ALPHA
	case driver.BDstColor:
		return C.VK_BLEND_FACTOR_DST_COLOR
	case driver.BInvDstColor:
		return C.VK_BLEND_FACTOR_ONE_MINUS_DST_COLOR
	case driver.BDstAlpha:
		return C.VK_BLEND_FACTOR_DST_ALPHA
	case driver.BInvDstAlpha:
		return C.VK_BLEND_FACTOR_ONE_MINUS_DST_ALPHA
	case driver.BSrcAlphaSaturated:
		return C.VK_BLEND_FACTOR_SRC_ALPHA_SATURATE
	case driver.BBlendColor:
		return C.VK_BLEND_FACTOR_CONSTANT_COLOR
	case driver.BInvBlendColor:
		return C.VK_BLEND_FACTOR_ONE_MINUS_CONSTANT_COLOR
	}

	// Expected to be unreachable.
	return ^C.VkBlendFactor(0)
}

// convColorMask converts a driver.ColorMask to a VkColorComponentFlags.
func convColorMask(cm driver.ColorMask) (flags C.VkColorComponentFlags) {
	if cm == driver.CAll {
		flags = C.VK_COLOR_COMPONENT_R_BIT | C.VK_COLOR_COMPONENT_G_BIT | C.VK_COLOR_COMPONENT_B_BIT | C.VK_COLOR_COMPONENT_A_BIT
	} else {
		if cm&driver.CRed != 0 {
			flags |= C.VK_COLOR_COMPONENT_R_BIT
		}
		if cm&driver.CGreen != 0 {
			flags |= C.VK_COLOR_COMPONENT_G_BIT
		}
		if cm&driver.CBlue != 0 {
			flags |= C.VK_COLOR_COMPONENT_B_BIT
		}
		if cm&driver.CAlpha != 0 {
			flags |= C.VK_COLOR_COMPONENT_A_BIT
		}
	}
	return
}
