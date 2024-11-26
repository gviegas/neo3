// Copyright 2022 Gustavo C. Viegas. All rights reserved.

// Package gltf implements glTF 2.0 serialization.
package gltf

import (
	"encoding/json"
	"io"
)

// Root glTF object.
type GLTF struct {
	ExtensionsUsed     []string    `json:"extensionsUsed,omitempty"`
	ExtensionsRequired []string    `json:"extensionsRequired,omitempty"`
	Accessors          []Accessor  `json:"accessors,omitempty"`
	Animations         []Animation `json:"animations,omitempty"`
	Asset              struct {
		Copyright  string `json:"copyright,omitempty"`
		Generator  string `json:"generator,omitempty"`
		Version    string `json:"version"`
		MinVersion string `json:"minVersion,omitempty"`
		Extensions any    `json:"extensions,omitempty"`
		Extras     any    `json:"extras,omitempty"`
	} `json:"asset"`
	Buffers     []Buffer     `json:"buffers,omitempty"`
	BufferViews []BufferView `json:"bufferViews,omitempty"`
	Cameras     []Camera     `json:"cameras,omitempty"`
	Images      []Image      `json:"images,omitempty"`
	Materials   []Material   `json:"materials,omitempty"`
	Meshes      []Mesh       `json:"meshes,omitempty"`
	Nodes       []Node       `json:"nodes,omitempty"`
	Samplers    []Sampler    `json:"samplers,omitempty"`
	Scene       *int64       `json:"scene,omitempty"`
	Scenes      []Scene      `json:"scenes,omitempty"`
	Skins       []Skin       `json:"skins,omitempty"`
	Textures    []Texture    `json:"textures,omitempty"`
	Extensions  any          `json:"extensions,omitempty"`
	Extras      any          `json:"extras,omitempty"`
}

// glTF.accessors' element.
type Accessor struct {
	BufferView    *int64    `json:"bufferView,omitempty"`
	ByteOffset    int64     `json:"byteOffset,omitempty"` // Default is 0.
	ComponentType int64     `json:"componentType"`
	Normalized    bool      `json:"normalized,omitempty"`
	Count         int64     `json:"count"`
	Type          string    `json:"type"`
	Max           []float32 `json:"max,omitempty"`
	Min           []float32 `json:"min,omitempty"`
	Sparse        *Sparse   `json:"sparse,omitempty"`
	Name          string    `json:"name,omitempty"`
	Extensions    any       `json:"extensions,omitempty"`
	Extras        any       `json:"extras,omitempty"`
}

// accessor.sparse.
type Sparse struct {
	Count   int64 `json:"count"`
	Indices struct {
		BufferView    int64 `json:"bufferView"`
		ByteOffset    int64 `json:"byteOffset,omitempty"` // Default is 0.
		ComponentType int64 `json:"componentType"`
		Extensions    any   `json:"extensions,omitempty"`
		Extras        any   `json:"extras,omitempty"`
	} `json:"indices"`
	Values struct {
		BufferView int64 `json:"bufferView"`
		ByteOffset int64 `json:"byteOffset,omitempty"` // Default is 0.
		Extensions any   `json:"extensions,omitempty"`
		Extras     any   `json:"extras,omitempty"`
	} `json:"values"`
	Extensions any `json:"extensions,omitempty"`
	Extras     any `json:"extras,omitempty"`
}

// accessor.*.componentType values.
const (
	BYTE           = 5120
	UNSIGNED_BYTE  = 5121
	SHORT          = 5122
	UNSIGNED_SHORT = 5123
	UNSIGNED_INT   = 5125
	FLOAT          = 5126
)

// accessor.type values.
const (
	SCALAR = "SCALAR"
	VEC2   = "VEC2"
	VEC3   = "VEC3"
	VEC4   = "VEC4"
	MAT2   = "MAT2"
	MAT3   = "MAT3"
	MAT4   = "MAT4"
)

// glTF.animations' element.
type Animation struct {
	Channels   []AChannel `json:"channels"`
	Samplers   []ASampler `json:"samplers"`
	Name       string     `json:"name,omitempty"`
	Extensions any        `json:"extensions,omitempty"`
	Extras     any        `json:"extras,omitempty"`
}

// animation.channels' element.
type AChannel struct {
	Sampler int64 `json:"sampler"`
	Target  struct {
		Node       *int64 `json:"node,omitempty"`
		Path       string `json:"path"`
		Extensions any    `json:"extensions,omitempty"`
		Extras     any    `json:"extras,omitempty"`
	} `json:"target"`
	Extensions any `json:"extensions,omitempty"`
	Extras     any `json:"extras,omitempty"`
}

// animation.samplers' element.
type ASampler struct {
	Input         int64  `json:"input"`
	Interpolation string `json:"interpolation,omitempty"` // Default is "LINEAR".
	Output        int64  `json:"output"`
	Extensions    any    `json:"extensions,omitempty"`
	Extras        any    `json:"extras,omitempty"`
}

// animation.channel.target.path values.
const (
	Ptranslation = "translation"
	Protation    = "rotation"
	Pscale       = "scale"
	Pweights     = "weights"
)

// animation.sampler.interpolation values.
const (
	ILINEAR     = "LINEAR"
	STEP        = "STEP"
	CUBICSPLINE = "CUBICSPLINE"
)

// glTF.buffers' element.
type Buffer struct {
	URI        string `json:"uri,omitempty"`
	ByteLength int64  `json:"byteLength"`
	Name       string `json:"name,omitempty"`
	Extensions any    `json:"extensions,omitempty"`
	Extras     any    `json:"extras,omitempty"`
}

// glTF.bufferViews' element.
type BufferView struct {
	Buffer     int64  `json:"buffer"`
	ByteOffset int64  `json:"byteOffset,omitempty"` // Default is 0.
	ByteLength int64  `json:"byteLength"`
	ByteStride int64  `json:"byteStride,omitempty"` // 0 for tightly packed.
	Target     int64  `json:"target,omitempty"`     // 0 for no hint.
	Name       string `json:"name,omitempty"`
	Extensions any    `json:"extensions,omitempty"`
	Extras     any    `json:"extras,omitempty"`
}

// bufferView.target values.
const (
	ARRAY_BUFFER = iota + 34962
	ELEMENT_ARRAY_BUFFER
)

// glTF.cameras' element.
type Camera struct {
	Orthographic *Orthographic `json:"orthographic,omitempty"`
	Perspective  *Perspective  `json:"perspective,omitempty"`
	Type         string        `json:"type"`
	Name         string        `json:"name,omitempty"`
	Extenions    any           `json:"extensions,omitempty"`
	Extras       any           `json:"extras,omitempty"`
}

// camera.orthographic.
type Orthographic struct {
	Xmag       float32 `json:"xmag"`
	Ymag       float32 `json:"ymag"`
	Zfar       float32 `json:"zfar"`
	Znear      float32 `json:"znear"`
	Extensions any     `json:"extensions,omitempty"`
	Extras     any     `json:"extras,omitempty"`
}

// camera.perspective.
type Perspective struct {
	AspectRatio float32 `json:"aspectRatio,omitempty"`
	YFOV        float32 `json:"yfov"`
	Zfar        float32 `json:"zfar,omitempty"` // 0 for infinite perspective.
	Znear       float32 `json:"znear"`
	Extensions  any     `json:"extensions,omitempty"`
	Extras      any     `json:"extras,omitempty"`
}

// camera.type values.
const (
	Tperspective  = "perspective"
	Torthographic = "ortographic"
)

// glTF.images' element.
type Image struct {
	URI        string `json:"uri,omitempty"`
	MimeType   string `json:"mimeType,omitempty"`
	BufferView *int64 `json:"bufferView,omitempty"`
	Name       string `json:"name,omitempty"`
	Extensions any    `json:"extensions,omitempty"`
	Extras     any    `json:"extras,omitempty"`
}

// image.mimeType values.
const (
	JPEG = "image/jpeg"
	PNG  = "image/png"
)

// glTF.materials' element.
type Material struct {
	PBRMetallicRoughness *PBRMetallicRoughness `json:"pbrMetallicRoughness,omitempty"`
	NormalTexture        *NormalTextureInfo    `json:"normalTexture,omitempty"`
	OcclusionTexture     *OcclusionTextureInfo `json:"occlusionTexture,omitempty"`
	EmissiveTexture      *TextureInfo          `json:"emissiveTexture,omitempty"`
	EmissiveFactor       *[3]float32           `json:"emissiveFactor,omitempty"` // Default is [0, 0, 0].
	AlphaMode            string                `json:"alphaMode,omitempty"`      // Default is "OPAQUE".
	AlphaCutoff          *float32              `json:"alphaCutoff,omitempty"`    // Default is 0.5.
	DoubleSided          bool                  `json:"doubleSided,omitempty"`    // Default is false.
	Name                 string                `json:"name,omitempty"`
	Extensions           any                   `json:"extensions,omitempty"`
	Extras               any                   `json:"extras,omitempty"`
}

// material.normalTextureInfo.
type NormalTextureInfo struct {
	Index      int64    `json:"index"`
	TexCoord   int64    `json:"texCoord,omitempty"` // Default is TEXCOORD_0.
	Scale      *float32 `json:"scale,omitempty"`    // Default is 1.
	Extensions any      `json:"extensions,omitempty"`
	Extras     any      `json:"extras,omitempty"`
}

// material.occlusionTextureInfo.
type OcclusionTextureInfo struct {
	Index      int64    `json:"index"`
	TexCoord   int64    `json:"texCoord,omitempty"` // Default is TEXCOORD_0.
	Strength   *float32 `json:"strength,omitempty"` // Default is 1.
	Extensions any      `json:"extensions,omitempty"`
	Extras     any      `json:"extras,omitempty"`
}

// material.pbrMetallicRoughness.
type PBRMetallicRoughness struct {
	BaseColorFactor          *[4]float32  `json:"baseColorFactor,omitempty"` // Default is [1, 1, 1, 1].
	BaseColorTexture         *TextureInfo `json:"baseColorTexture,omitempty"`
	MetallicFactor           *float32     `json:"metallicFactor,omitempty"`  // Default is 1.
	RoughnessFactor          *float32     `json:"roughnessFactor,omitempty"` // Default is 1.
	MetallicRoughnessTexture *TextureInfo `json:"metallicRoughnessTexture,omitempty"`
	Extensions               any          `json:"extensions,omitempty"`
	Extras                   any          `json:"extras,omitempty"`
}

// material.alphaMode values.
const (
	OPAQUE = "OPAQUE"
	MASK   = "MASK"
	BLEND  = "BLEND"
)

// glTF.meshes' element.
type Mesh struct {
	Primitives []Primitive `json:"primitives"`
	Weights    []float32   `json:"weights,omitempty"`
	Name       string      `json:"name,omitempty"`
	Extensions any         `json:"extensions,omitempty"`
	Extras     any         `json:"extras,omitempty"`
}

// mesh.primitives' element.
type Primitive struct {
	Attributes map[string]int64   `json:"attributes"`
	Indices    *int64             `json:"indices,omitempty"`
	Material   *int64             `json:"material,omitempty"`
	Mode       *int64             `json:"mode,omitempty"` // Default is 4.
	Targets    []map[string]int64 `json:"targets,omitempty"`
	Extensions any                `json:"extensions,omitempty"`
	Extras     any                `json:"extras,omitempty"`
}

// mesh.primitive.mode values.
const (
	POINTS = iota
	LINES
	LINE_LOOP
	LINE_STRIP
	TRIANGLES
	TRIANGLE_STRIP
	TRIANGLE_FAN
)

// glTF.nodes' element.
// XXX: Way too many pointers here.
type Node struct {
	Camera      *int64       `json:"camera,omitempty"`
	Children    []int64      `json:"children,omitempty"`
	Skin        *int64       `json:"skin,omitempty"`
	Matrix      *[16]float32 `json:"matrix,omitempty"` // Default is identity.
	Mesh        *int64       `json:"mesh,omitempty"`
	Rotation    *[4]float32  `json:"rotation,omitempty"`    // Default is [0, 0, 0, 1].
	Scale       *[3]float32  `json:"scale,omitempty"`       // Default is [1, 1, 1].
	Translation *[3]float32  `json:"translation,omitempty"` // Default is [0, 0, 0].
	Weights     []float32    `json:"weights,omitempty"`
	Name        string       `json:"name,omitempty"`
	Extensions  any          `json:"extensions,omitempty"`
	Extras      any          `json:"extras,omitempty"`
}

// node.extensions.KHR_lights_punctual.
type NodeLight struct {
	Light      int64 `json:"light,omitempty"`
	Extensions any   `json:"extensions,omitempty"`
	Extras     any   `json:"extras,omitempty"`
}

// glTF.samplers' element.
type Sampler struct {
	// Valid filter/wrap mode values differ from 0.
	MagFilter  int64  `json:"magFilter,omitempty"`
	MinFilter  int64  `json:"minFilter,omitempty"`
	WrapS      int64  `json:"wrapS,omitempty"` // Default is 10497.
	WrapT      int64  `json:"wrapT,omitempty"` // Default is 10497.
	Name       string `json:"name,omitempty"`
	Extensions any    `json:"extensions,omitempty"`
	Extras     any    `json:"extras,omitempty"`
}

// sampler.*Filter values.
const (
	NEAREST                = 9728
	FLINEAR                = 9729
	NEAREST_MIPMAP_NEAREST = 9984
	LINEAR_MIPMAP_NEAREST  = 9985
	NEAREST_MIPMAP_LINEAR  = 9986
	LINEAR_MIPMAP_LINEAR   = 9987
)

// sampler.wrap* values.
const (
	CLAMP_TO_EDGE   = 33071
	MIRRORED_REPEAT = 33648
	REPEAT          = 10497
)

// glTF.scenes' element.
type Scene struct {
	Nodes      []int64 `json:"nodes,omitempty"`
	Name       string  `json:"name,omitempty"`
	Extensions any     `json:"extensions,omitempty"`
	Extras     any     `json:"extras,omitempty"`
}

// glTF.skins' element.
type Skin struct {
	InverseBindMatrices *int64  `json:"inverseBindMatrices,omitempty"`
	Skeleton            *int64  `json:"skeleton,omitempty"`
	Joints              []int64 `json:"joints"`
	Name                string  `json:"name,omitempty"`
	Extensions          any     `json:"extensions,omitempty"`
	Extras              any     `json:"extras,omitempty"`
}

// glTF.textures' element.
type Texture struct {
	Sampler    *int64 `json:"sampler,omitempty"`
	Source     *int64 `json:"source,omitempty"`
	Name       string `json:"name,omitempty"`
	Extensions any    `json:"extensions,omitempty"`
	Extras     any    `json:"extras,omitempty"`
}

// textureInfo.
type TextureInfo struct {
	Index      int64 `json:"index"`
	TexCoord   int64 `json:"texCoord,omitempty"` // Default is TEXCOORD_0.
	Extensions any   `json:"extensions,omitempty"`
	Extras     any   `json:"extras,omitempty"`
}

// glTF.extensions.KHR_lights_punctual.
type KHRLightsPunctual struct {
	Lights     []Light `json:"lights"`
	Extensions any     `json:"extensions,omitempty`
	Extras     any     `json:"extras,omitempty`
}

// KHR_lights_punctual.lights' element.
type Light struct {
	Color      *[3]float32 `json:"color,omitempty"`     // Default is [1, 1, 1].
	Intensity  *float32    `json:"intensity,omitempty"` // Default is 1.
	Spot       *Spot       `json:"spot,omitempty"`
	Range      float32     `json:"range"` // 0 for infinite range.
	Type       string      `json:"type"`
	Name       string      `json:"name,omitempty"`
	Extensions any         `json:"extensions,omitempty`
	Extras     any         `json:"extras,omitempty`
}

// KHR_lights_punctual.light.spot.
type Spot struct {
	InnerConeAngle float32  `json:"innerConeAngle,omitempty"` // Default is 0.
	OuterConeAngle *float32 `json:"outerConeAngle,omitempty"` // Default is 0.7853981633974483.
	Extensions     any      `json:"extensions,omitempty`
	Extras         any      `json:"extras,omitempty`
}

// Encode encodes gltf into w.
func Encode(w io.Writer, gltf *GLTF) error {
	enc := json.NewEncoder(w)
	err := enc.Encode(gltf)
	if err != nil {
		return err
	}
	return nil
}

// Decode decodes r into a new GLTF instance.
func Decode(r io.Reader) (*GLTF, error) {
	var gltf GLTF
	dec := json.NewDecoder(r)
	err := dec.Decode(&gltf)
	if err != nil {
		return nil, err
	}
	return &gltf, nil
}
