// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package texture

import (
	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
	"github.com/gviegas/scene/internal/bitm"
)

// stagingBuffer is used to copy image data
// between the CPU and the GPU.
type stagingBuffer struct {
	buf driver.Buffer
	bm  bitm.Bitm[uint32]
}

// Use a large block size since textures usually
// need large allocations.
// 1024x1024 32-bit textures (no mip) will take
// one bitmap word with this configuration.
const (
	blockSize = 131072
	nbit      = 32
)

// newStaging creates a new stagingBuffer with the
// given size in bytes.
// n must be greater than 0; it will be rounded up
// to a multiple of blockSize * nbit.
// It fails if driver.NewBuffer fails.
func newStaging(n int) (*stagingBuffer, error) {
	if n <= 0 {
		panic("texture.newStaging: n <= 0")
	}
	n = (n + blockSize*nbit - 1) &^ (blockSize*nbit - 1)
	// No usage flags necessary; all buffers
	// support copying.
	buf, err := ctxt.GPU().NewBuffer(int64(n), true, 0)
	if err != nil {
		return nil, err
	}
	var bm bitm.Bitm[uint32]
	bm.Grow(n / blockSize / nbit)
	return &stagingBuffer{buf, bm}, nil
}

// free invalidates s and destroys the driver.Buffer.
func (s *stagingBuffer) free() {
	// TODO: Sync.
	if s.buf != nil {
		s.buf.Destroy()
	}
	*s = stagingBuffer{}
}
