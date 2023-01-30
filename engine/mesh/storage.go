// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"sync"

	"github.com/gviegas/scene/driver"
)

// Global mesh storage.
var storage meshBuffer

// SetBuffer sets the GPU buffer into which mesh data will
// be stored.
// The buffer must be host-visible and its usage must include
// both driver.UVertexData and driver.UIndexData.
// It returns the replaced buffer, if any.
// NOTE: Calls to this function invalidate all previously
// created meshes.
func SetBuffer(buf driver.Buffer) driver.Buffer {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	switch buf {
	case storage.buf:
		return nil
	case nil:
		storage.prims = nil
	default:
		storage.prims = storage.prims[:0]
	}
	prev := storage.buf
	storage.buf = buf
	return prev
}

// meshBuffer manages vertex/index data of created meshes.
type meshBuffer struct {
	buf   driver.Buffer
	mu    sync.Mutex
	prims []primitive
}

// primitive is an entry in a mesh buffer.
type primitive struct {
	topology driver.Topology
	count    int
	mask     Semantic
	vertex   [MaxSemantic]struct {
		format driver.VertexFmt
		offset int64
	}
	index struct {
		format driver.IndexFmt
		offset int64
	}
	span
}

// span defines a buffer range in number of blocks.
type span struct {
	start int
	end   int
}

// span granularity.
const blockSize = 512
