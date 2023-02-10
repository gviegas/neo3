// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"sync"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/internal/bitm"
)

// Global mesh storage.
var storage meshBuffer

// SetBuffer sets the GPU buffer into which mesh data will
// be stored.
// The buffer must be host-visible, its usage must include
// both driver.UVertexData and driver.UIndexData, and its
// capacity must be a multiple of 16384 bytes.
// It returns the replaced buffer, if any.
//
// NOTE: Calls to this function invalidate all previously
// created meshes.
func SetBuffer(buf driver.Buffer) driver.Buffer {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	switch buf {
	case storage.buf:
		return nil
	case nil:
		storage.spanMap = bitm.Bitm[uint32]{}
		storage.prims = nil
	default:
		c := buf.Cap()
		n := c / (blockSize * 32)
		if n > int64(^uint(0)>>1) || c != n*(blockSize*32) {
			panic("invalid mesh buffer capacity")
		}
		storage.spanMap = bitm.Bitm[uint32]{}
		storage.spanMap.Grow(int(n))
		storage.prims = storage.prims[:0]
	}
	prev := storage.buf
	storage.buf = buf
	return prev
}

// meshBuffer manages vertex/index data of created meshes.
type meshBuffer struct {
	buf     driver.Buffer
	spanMap bitm.Bitm[uint32]
	prims   []primitive
	mu      sync.Mutex
}

// primitive is an entry in a mesh buffer.
type primitive struct {
	topology driver.Topology
	count    int
	mask     Semantic
	vertex   [MaxSemantic]struct {
		format driver.VertexFmt
		span
	}
	index struct {
		format driver.IndexFmt
		span
	}
	// Indices into meshBuffer.prims defining
	// a primitive list. Only used by meshes
	// with multiple primitives.
	next int
	prev int
}

// span defines a buffer range in number of blocks.
type span struct {
	start int
	end   int
}

// span granularity.
const blockSize = 512

// byteStart computes the span's first byte.
func (s span) byteStart() int { return s.start * blockSize }

// byteEnd computes the span's one-past-the-end byte.
func (s span) byteEnd() int { return s.end * blockSize }

// byteLen computes the span's byte length.
func (s span) byteLen() int { return (s.end - s.start) * blockSize }
