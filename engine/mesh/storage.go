// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"io"
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
// Note: Calls to this function invalidate all previously
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
		storage.primMap = bitm.Bitm[uint16]{}
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
	primMap bitm.Bitm[uint16]
	prims   []primitive
	mu      sync.Mutex
}

// newEntry creates a new entry in the buffer containing
// the primitive specified by data.
func (b *meshBuffer) newEntry(data *PrimitiveData, srcs []io.ReadSeeker) (Primitive, error) {
	panic("not implemented")
}

// link links a primitive entry to another.
// This is only relevant for meshes that contain multiple
// primitives.
func (b *meshBuffer) link(prim Primitive, next Primitive) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if prim.bufIdx != next.bufIdx {
		panic("attempt to link primitives from different buffers")
	}
	b.prims[prim.index].next = next.index
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
	// Index into meshBuffer.prims identifying
	// the next primitive of a mesh. Whether
	// this value is meaningful or not depends
	// on the Mesh.primLen field.
	next int
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
