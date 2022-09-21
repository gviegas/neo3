// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"encoding/binary"
	"errors"
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

// SeekJSON seeks into r until it finds the beginning
// of the JSON string.
// If successful, it returns the length of the chunk.
// r must refer to an unread GLB blob.
func SeekJSON(r io.Reader) (n int, err error) {
	if !IsGLB(r) {
		err = errors.New("gltf: not a GLB blob")
		return
	}
	var c glbChunk
	err = binary.Read(r, binary.LittleEndian, c[:])
	switch {
	case err != nil:
	case c[chunkLength] == 0 || c[chunkType] != typeJSON:
		err = errors.New("gltf: invalid GLB chunk")
	default:
		n = int(c[chunkLength])
	}
	return
}

// SeekBIN seeks into r until if finds the beginning
// of the binary buffer.
// If successful, it returns the length of the chunk,
// which may be zero.
// Note that, the BIN chunk being optional, an error
// of io.EOF may indicate its absence.
// r must refer to an unread GLB blob.
func SeekBIN(r io.Reader) (n int, err error) {
	n, err = SeekJSON(r)
	if err != nil {
		return
	}
	if s, ok := r.(io.Seeker); ok {
		_, err = s.Seek(int64(n), io.SeekCurrent)
	} else {
		b := make([]byte, n)
		for len(b) > 0 && err == nil {
			n, err = r.Read(b)
			b = b[n:]
		}
	}
	n = 0
	if err != nil {
		return
	}
	var c glbChunk
	err = binary.Read(r, binary.LittleEndian, c[:])
	switch {
	case err != nil:
	case c[chunkType] != typeBIN:
		err = errors.New("gltf: invalid GLB chunk")
	default:
		n = int(c[chunkLength])
	}
	return
}
