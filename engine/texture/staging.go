// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package texture

import (
	"errors"
	"runtime"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
	"github.com/gviegas/scene/internal/bitm"
)

// Global staging buffer(s).
var staging chan *stagingBuffer

func init() {
	n := runtime.GOMAXPROCS(-1)
	staging = make(chan *stagingBuffer, n)
	for i := 0; i < n; i++ {
		s, err := newStaging(blockSize * nbit)
		if err != nil {
			s = &stagingBuffer{}
		}
		staging <- s
	}
}

// stagingBuffer is used to copy image data
// between the CPU and the GPU.
type stagingBuffer struct {
	wk  chan *driver.WorkItem
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
func newStaging(n int) (*stagingBuffer, error) {
	if n <= 0 {
		panic("texture.newStaging: n <= 0")
	}
	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		return nil, err
	}
	wk := make(chan *driver.WorkItem, 1)
	wk <- &driver.WorkItem{Work: []driver.CmdBuffer{cb}}
	n = (n + blockSize*nbit - 1) &^ (blockSize*nbit - 1)
	// No usage flags necessary; all buffers
	// support copying.
	buf, err := ctxt.GPU().NewBuffer(int64(n), true, 0)
	if err != nil {
		cb.Destroy()
		return nil, err
	}
	var bm bitm.Bitm[uint32]
	bm.Grow(n / blockSize / nbit)
	return &stagingBuffer{wk, buf, bm}, nil
}

// copyView writes data to s's buffer and records a
// copy command that updates the given view of t.
// Only the first mip level must be provided in
// data. If t is arrayed and view is the last view,
// then data must contain the first level of every
// layer, in order and tightly packed.
func (s *stagingBuffer) copyView(t *Texture, view int, before, after driver.Layout, data []byte) (err error) {
	if t.param.Samples != 1 {
		return errors.New(prefix + "cannot copy data to MS texture")
	}
	if view < 0 || view >= len(t.views) {
		return errors.New(prefix + "view index out of bounds")
	}

	il := view
	nl := 1
	if t.param.Layers > 1 {
		switch n := len(t.views); {
		case view == n-1:
			il = 0
			nl = t.param.Layers
		case n < t.param.Layers:
			// Cube texture.
			il = view * 6
			nl = 6
		}
	}
	n := t.param.PixelFmt.Size() * t.param.Dim3D.Width * t.param.Dim3D.Height
	if len(data) < n*nl {
		return errors.New(prefix + "not enough data for copying")
	}

	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if err = wk.Work[0].Begin(); err != nil {
			s.wk <- wk
			return
		}
	}
	// s.stage may call s.commit.
	s.wk <- wk
	// TODO: Levels > 1.
	var off int64
	if off, err = s.stage(data); err != nil {
		return
	}

	wk = <-s.wk
	wk.Work[0].Transition([]driver.Transition{
		{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SAll,
				SyncAfter:    driver.SCopy,
				AccessBefore: driver.AAnyRead | driver.AAnyWrite,
				AccessAfter:  driver.ACopyRead | driver.ACopyWrite,
			},
			LayoutBefore: before,
			LayoutAfter:  driver.LCopyDst,
			Img:          t.views[view].Image(),
			Layer:        il,
			Layers:       nl,
			Level:        0,
			Levels:       1, // TODO
		},
	})

	for i := 0; i < nl; i++ {
		wk.Work[0].CopyBufToImg(&driver.BufImgCopy{
			Buf:    s.buf,
			BufOff: off + int64(n*i),
			Stride: [2]int64{int64(t.param.Dim3D.Width)},
			Img:    t.views[view].Image(),
			ImgOff: driver.Off3D{},
			Layer:  il + i,
			Level:  0,
			Size:   t.param.Dim3D,
			// TODO: Handle depth/stencil formats.
		})
	}
	if t.param.Levels > 1 {
		// TODO
		panic("stagingBuffer.copyView: no mip gen yet")
	}

	wk.Work[0].Transition([]driver.Transition{
		{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SCopy,
				SyncAfter:    driver.SAll,
				AccessBefore: driver.ACopyRead | driver.ACopyWrite,
				AccessAfter:  driver.AAnyRead | driver.AAnyWrite,
			},
			LayoutBefore: driver.LCopyDst,
			LayoutAfter:  after,
			Img:          t.views[view].Image(),
			Layer:        il,
			Layers:       nl,
			Level:        0,
			Levels:       1, // TODO
		},
	})
	s.wk <- wk
	return
}

// stage writes CPU data to s's buffer.
// It may need to commit pending copy commands to
// grow the buffer.
// It returns an offset from the start of s.buf
// identifying where data was copied to.
func (s *stagingBuffer) stage(data []byte) (off int64, err error) {
	n := (len(data) + blockSize - 1) / blockSize
	if n == 0 {
		panic("stagingBuffer.stage: len(data) == 0")
	}
	idx, ok := s.bm.SearchRange(n)
	if !ok {
		if err = s.commit(); err != nil {
			return
		}
		// TODO: Consider using idx 0 instead.
		idx = s.bm.Len()
		n := (n + nbit - 1) / nbit
		s.bm.Grow(n)
		// TODO: Make buffer cap bounds configurable.
		n = int(s.buf.Cap()) + n*blockSize*nbit
		s.buf.Destroy()
		if s.buf, err = ctxt.GPU().NewBuffer(int64(n), true, 0); err != nil {
			// TODO: Try again ignoring previous
			// s.buf.Cap() value (if not 0).
			s.bm = bitm.Bitm[uint32]{}
			return
		}
	}
	for i := 0; i < n; i++ {
		s.bm.Set(idx + i)
	}
	off = int64(idx) * blockSize
	copy(s.buf.Bytes()[off:], data)
	return
}

// commit commits the copy commands for execution.
// It blocks until execution completes.
func (s *stagingBuffer) commit() (err error) {
	wk := <-s.wk
	if s.bm.Len() == s.bm.Rem() {
		s.wk <- wk
		return
	}
	s.bm.Clear()
	if err = wk.Work[0].End(); err != nil {
		s.wk <- wk
		return
	}
	if err = ctxt.GPU().Commit(wk, s.wk); err != nil {
		s.wk <- wk
		return
	}
	wk = <-s.wk
	err, wk.Err = wk.Err, nil
	s.wk <- wk
	return
}

// free invalidates s and destroys the driver
// resources.
func (s *stagingBuffer) free() {
	if s.wk != nil {
		wk := <-s.wk
		wk.Work[0].Destroy()
	}
	if s.buf != nil {
		s.buf.Destroy()
	}
	*s = stagingBuffer{}
}
