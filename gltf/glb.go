// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"encoding/binary"
	"io"
)

// GLB header.
type glbHeader [3]uint32

// Indices in glbHeader.
const (
	headerMagic   = 0
	headerVersion = 1
	headerLength  = 2
)

// GLB chunk.
type (
	glbChunk     [2]uint32
	glbChunkData []byte
)

// Indices in glbChunk.
const (
	chunkLength = 0
	chunkType   = 1
	// Then payload (glbChunkData).
)

const (
	// glbHeader[headerMagic].
	magic = 0x46546c67

	// glbChunk[chunkType].
	typeJSON = 0x4e4f534a
	typeBIN  = 0x004e4942
)

// IsGLB returns whether r refers to a binary glTF (version 2).
// It assumes that r was positioned accordingly.
func IsGLB(r io.Reader) bool {
	var h glbHeader
	err := binary.Read(r, binary.LittleEndian, h[:])
	switch {
	case err != nil, h[headerMagic] != magic, h[headerVersion] != 2:
		return false
	default:
		return true
	}
}
