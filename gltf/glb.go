// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"bytes"
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
// whence must be either io.SeekStart, indicating that
// r refers to an unread GLB blob, or io.SeekCurrent,
// in which case r is assumed to be positioned at the
// beginning of the JSON chunk.
func SeekJSON(r io.Reader, whence int) (n int, err error) {
	switch whence {
	case io.SeekStart:
		if !IsGLB(r) {
			err = errors.New("gltf: not a GLB blob")
			return
		}
	case io.SeekCurrent:
	default:
		err = errors.New("gltf: invalid whence value")
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
// whence must be either io.SeekStart, indicating that
// r refers to an unread GLB blob, or io.SeekCurrent,
// in which case r is assumed to be positioned at the
// the beginning of the BIN chunk.
func SeekBIN(r io.Reader, whence int) (n int, err error) {
	switch whence {
	case io.SeekStart:
		n, err = SeekJSON(r, whence)
		if err != nil {
			return
		}
		if s, ok := r.(io.Seeker); ok {
			_, err = s.Seek(int64(n), io.SeekCurrent)
		} else {
			b := make(glbChunkData, n)
			for len(b) > 0 && err == nil {
				n, err = r.Read(b)
				b = b[n:]
			}
		}
		n = 0
		if err != nil {
			return
		}
	case io.SeekCurrent:
	default:
		err = errors.New("gltf: invalid whence value")
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

// Pack writes to w a GLB blob assembled from gltf and bin
// as JSON and BIN chunks, respectively.
// If len(bin) is 0, the BIN chunk is omitted.
func Pack(w io.Writer, gltf *GLTF, bin []byte) (err error) {
	h := glbHeader{
		headerMagic:   magic,
		headerVersion: 2,
	}
	var buf bytes.Buffer
	if err = Encode(&buf, gltf); err != nil {
		return
	}
	// Encoding produces compacted JSON, but appends a
	// newline at the end.
	jn := buf.Len() - 1
	buf.Truncate(jn)
	if pad := jn % 4; pad != 0 {
		for ; pad != 4; pad++ {
			buf.WriteByte(0x20)
		}
		jn = buf.Len()
	}
	jc := glbChunk{
		chunkLength: uint32(jn),
		chunkType:   typeJSON,
	}
	// Note that the binary buffer is optional.
	if bn := len(bin); bn == 0 {
		if uint64(20+jn) > uint64(^uint32(0)-3) {
			err = errors.New("gltf: GLB length overflow")
			return
		}
		h[headerLength] = 12 + 8 + jc[chunkLength]
		if err = binary.Write(w, binary.LittleEndian, h[:]); err != nil {
			return
		}
		if err = binary.Write(w, binary.LittleEndian, jc[:]); err != nil {
			return
		}
		if _, err = w.Write(buf.Bytes()); err != nil {
			return
		}
	} else {
		pad := bn % 4
		if pad == 0 {
			pad = 4
		}
		bc := glbChunk{
			chunkLength: uint32(bn + 4 - pad),
			chunkType:   typeBIN,
		}
		if uint64(32+jn+bn-pad) > uint64(^uint32(0)-3) {
			err = errors.New("gltf: GLB length overflow")
			return
		}
		h[headerLength] = 12 + 8 + jc[chunkLength] + 8 + bc[chunkLength]
		if err = binary.Write(w, binary.LittleEndian, h[:]); err != nil {
			return
		}
		if err = binary.Write(w, binary.LittleEndian, jc[:]); err != nil {
			return
		}
		if _, err = w.Write(buf.Bytes()); err != nil {
			return
		}
		if err = binary.Write(w, binary.LittleEndian, bc[:]); err != nil {
			return
		}
		if _, err = w.Write(bin); err != nil {
			return
		}
		for ; pad != 4; pad++ {
			if _, err = w.Write([]byte{0}); err != nil {
				return
			}
		}
	}
	return
}

// Unpack reads the GLB blob from r to decode the JSON chunk
// (structured JSON content) into a new GLTF struct.
// If the BIN chunk (binary buffer) is present, its contents
// are copied as-is into a new byte slice.
func Unpack(r io.Reader) (gltf *GLTF, bin []byte, err error) {
	n := 0
	if n, err = SeekJSON(r, io.SeekStart); err != nil {
		return
	}
	if gltf, err = Decode(io.LimitReader(r, int64(n))); err != nil {
		return
	}
	if n, err = SeekBIN(r, io.SeekCurrent); err != nil {
		if n == 0 && err == io.EOF {
			err = nil
		}
		return
	}
	bin = make([]byte, n)
	for err == nil {
		off := len(bin) - n
		x := 0
		x, err = r.Read(bin[off:])
		n -= x
		if n == 0 {
			err = nil
			break
		}
	}
	return
}
