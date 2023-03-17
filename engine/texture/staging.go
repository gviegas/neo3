// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package texture

import (
	"errors"
	"runtime"
	"sync"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
	"github.com/gviegas/scene/internal/bitm"
)

var (
	// Global staging buffer(s).
	staging chan *stagingBuffer
	// Variables for Commit calls.
	commitMu    sync.Mutex
	commitCache []*stagingBuffer
	commitWk    chan *driver.WorkItem
)

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
	commitCache = make([]*stagingBuffer, 0, n)
	commitWk = make(chan *driver.WorkItem, 1)
	commitWk <- &driver.WorkItem{Work: make([]driver.CmdBuffer, 0, n)}
}

// Commit executes all pending Texture copies.
// It blocks until execution completes.
func Commit() (err error) {
	commitMu.Lock()
	cwk := <-commitWk

	// This deferral correctly clears global
	// state, regardless of the outcome.
	// Code below ensures that the command
	// buffers are reset if necessary.
	defer func() {
		for _, x := range commitCache {
			x.bm.Clear()
			x.drainPending(err != nil)
			staging <- x
		}
		commitCache = commitCache[:0]
		cwk.Work = cwk.Work[:0]
		commitWk <- cwk
		commitMu.Unlock()
	}()

	n := cap(staging)
	for i := 0; i < n; i++ {
		commitCache = append(commitCache, <-staging)
	}

	for i, x := range commitCache {
		wk := <-x.wk
		if !wk.Work[0].IsRecording() {
			if len(x.pend) != 0 {
				// This should never happen.
				panic("texture.Commit: pending copies while not recording")
			}
		} else if err = wk.Work[0].End(); err != nil {
			x.wk <- wk
			for _, x := range cwk.Work {
				// Need to reset these since
				// they won't be committed.
				x.Reset()
			}
			for _, x := range commitCache[i+1:] {
				// Need to reset these since
				// they won't be ended.
				wk := <-x.wk
				wk.Work[0].Reset()
				x.wk <- wk
			}
			return
		} else {
			cwk.Work = append(cwk.Work, wk.Work[0])
		}
		x.wk <- wk
	}

	if len(cwk.Work) == 0 {
		return
	}
	if err = ctxt.GPU().Commit(cwk, commitWk); err != nil {
		return
	}
	cwk = <-commitWk
	err, cwk.Err = cwk.Err, nil
	return
}

// stagingBuffer is used to copy image data
// between the CPU and the GPU.
type stagingBuffer struct {
	wk   chan *driver.WorkItem
	buf  driver.Buffer
	bm   bitm.Bitm[uint32]
	pend []pendingCopy
}

// pendingCopy is used to track Texture/view
// pairs that have a pending copy operation.
type pendingCopy struct {
	tex  *Texture
	view int
	// The layout that will be set
	// after the copy executes.
	layout driver.Layout
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
	return &stagingBuffer{wk, buf, bm, nil}, nil
}

// copyToView records a copy command that copies
// data from s's buffer into view.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// blockSize).
// Only the first mip level must be provided.
// If t is arrayed and view is the last view, then
// the buffer must contain the first level of
// every layer, in order and tightly packed.
func (s *stagingBuffer) copyToView(t *Texture, view int, off int64) (err error) {
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
	if off+int64(n*nl) > s.buf.Cap() {
		return errors.New(prefix + "not enough buffer capacity for copying")
	}

	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if err = wk.Work[0].Begin(); err != nil {
			s.bm.Clear()
			s.wk <- wk
			return
		}
	}

	wk.Work[0].Transition([]driver.Transition{
		{
			Barrier: driver.Barrier{
				SyncBefore:   driver.SNone,
				SyncAfter:    driver.SCopy,
				AccessBefore: driver.ANone,
				AccessAfter:  driver.ACopyWrite,
			},
			LayoutBefore: driver.LUndefined,
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
			// TODO: Stride[0] must be 256-byte aligned.
			Stride: [2]int64{int64(t.param.Dim3D.Width)},
			Img:    t.views[view].Image(),
			ImgOff: driver.Off3D{},
			Layer:  il + i,
			Level:  0,
			Size:   t.param.Dim3D,
			// TODO: Handle depth/stencil formats.
		})
		// The current layout is not relevant
		// because the whole layer is going to
		// be overwritten by this command.
		// TODO: Change this when adding support
		// for sub-view copying.
		_ = t.setPending(il + i)
		s.pend = append(s.pend, pendingCopy{t, il + i, driver.LCopyDst})
	}
	if t.param.Levels > 1 {
		// TODO
		panic("stagingBuffer.copyToView: no mip gen yet")
	}

	s.wk <- wk
	return
}

// copyFromView records a copy command that copies
// data from view into s's buffer.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// blockSize).
func (s *stagingBuffer) copyFromView(t *Texture, view int, off int64) (err error) {
	if t.param.Samples != 1 {
		return errors.New(prefix + "cannot copy data from MS texture")
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
	// TODO: Consider the required space for
	// all mip levels.
	n := t.param.PixelFmt.Size() * t.param.Dim3D.Width * t.param.Dim3D.Height
	if off+int64(n*nl) > s.buf.Cap() {
		return errors.New(prefix + "not enough buffer capacity for copying")
	}
	// Need separate transitions if not all
	// layers are in the same layout.
	var differ bool
	before := []driver.Layout{t.setPending(il)}
	for i := 1; i < nl; i++ {
		layout := t.setPending(il + i)
		before = append(before, layout)
		differ = differ || layout != before[0]
	}

	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if err = wk.Work[0].Begin(); err != nil {
			s.bm.Clear()
			s.wk <- wk
			return
		}
	}

	if differ {
		for i := 0; i < nl; i++ {
			wk.Work[0].Transition([]driver.Transition{
				{
					Barrier: driver.Barrier{
						SyncBefore:   driver.SNone,
						SyncAfter:    driver.SCopy,
						AccessBefore: driver.ANone,
						AccessAfter:  driver.ACopyRead,
					},
					LayoutBefore: before[i],
					LayoutAfter:  driver.LCopySrc,
					Img:          t.views[view].Image(),
					Layer:        il + i,
					Layers:       1,
					Level:        0,
					Levels:       1, // TODO
				},
			})
		}
	} else {
		wk.Work[0].Transition([]driver.Transition{
			{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SNone,
					SyncAfter:    driver.SCopy,
					AccessBefore: driver.ANone,
					AccessAfter:  driver.ACopyRead,
				},
				LayoutBefore: before[0],
				LayoutAfter:  driver.LCopySrc,
				Img:          t.views[view].Image(),
				Layer:        il,
				Layers:       nl,
				Level:        0,
				Levels:       1, // TODO
			},
		})
	}

	for i := 0; i < nl; i++ {
		wk.Work[0].CopyImgToBuf(&driver.BufImgCopy{
			Buf:    s.buf,
			BufOff: off + int64(n*i),
			// TODO: Stride[0] must be 256-byte aligned.
			Stride: [2]int64{int64(t.param.Dim3D.Width)},
			Img:    t.views[view].Image(),
			ImgOff: driver.Off3D{},
			Layer:  il + i,
			Level:  0,
			Size:   t.param.Dim3D,
			// TODO: Handle depth/stencil formats.
		})
		s.pend = append(s.pend, pendingCopy{t, il + i, driver.LCopySrc})
	}
	if t.param.Levels > 1 {
		// TODO
		panic("stagingBuffer.copyFromView: no mip copy yet")
	}

	s.wk <- wk
	return
}

// stage writes CPU data to s's buffer.
// It may need to commit pending copy commands to
// grow the buffer.
// It returns an offset from the start of s.buf
// identifying where data was copied to.
func (s *stagingBuffer) stage(data []byte) (off int64, err error) {
	if off, err = s.reserve(len(data)); err == nil {
		copy(s.buf.Bytes()[off:], data)
	}
	return
}

// unstage writes s.buf's data to dst.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// blockSize).
// It returns the number of bytes written.
//
// NOTE: Since stagingBuffer methods may flush
// the command buffer and/or clear the bitmap,
// unstage usually should be called right after a
// copy-back command is committed and before
// staging new copy commands.
func (s *stagingBuffer) unstage(off int64, dst []byte) (n int) {
	if off >= s.buf.Cap() {
		return
	}
	if off%blockSize != 0 {
		panic("stagingBuffer.unstage: misaligned off")
	}
	n = copy(dst, s.buf.Bytes()[off:])
	ib := int(off) / blockSize
	nb := (n + blockSize - 1) / blockSize
	for i := 0; i < nb; i++ {
		s.bm.Unset(ib + i)
	}
	return
}

// reserve reserves a contiguous range of n bytes
// within s.buf.
// It may need to commit pending copy commands to
// grow the buffer.
// It returns an offset from the start of s.buf
// identifying where the range starts.
func (s *stagingBuffer) reserve(n int) (off int64, err error) {
	if n <= 0 {
		panic("stagingBuffer.reserve: n <= 0")
	}
	n = (n + blockSize - 1) / blockSize
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
		n = n * blockSize * nbit
		if s.buf != nil {
			n += int(s.buf.Cap())
			s.buf.Destroy()
		}
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
	return
}

// commit commits the copy commands for execution.
// It blocks until execution completes.
func (s *stagingBuffer) commit() (err error) {
	wk := <-s.wk
	if !wk.Work[0].IsRecording() {
		if len(s.pend) != 0 {
			// This should never happen.
			panic("stagingBuffer.commit: pending copies while not recording")
		}
		s.wk <- wk
		return
	}
	// TODO: May have to clear the
	// bitmap unconditionally.
	s.bm.Clear()
	if err = wk.Work[0].End(); err != nil {
		s.drainPending(true)
		s.wk <- wk
		return
	}
	if err = ctxt.GPU().Commit(wk, s.wk); err != nil {
		s.drainPending(true)
		s.wk <- wk
		return
	}
	wk = <-s.wk
	err, wk.Err = wk.Err, nil
	s.drainPending(err != nil)
	s.wk <- wk
	return
}

// drainPending removes every element from s.pend
// and updates the textures accordingly.
// If failed is true, then the layouts are set to
// driver.LUndefined instead.
func (s *stagingBuffer) drainPending(failed bool) {
	if failed {
		for _, x := range s.pend {
			x.tex.unsetPending(x.view, driver.LUndefined)
		}
	} else {
		for _, x := range s.pend {
			x.tex.unsetPending(x.view, x.layout)
		}
	}
	s.pend = s.pend[:0]
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
	s.drainPending(true)
	*s = stagingBuffer{}
}
