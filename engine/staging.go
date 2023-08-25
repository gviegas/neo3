// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"
	"runtime"
	"sync"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/internal/bitm"
)

var (
	// Global staging buffer(s).
	staging chan *stagingBuffer
	// Variables for CommitStaging calls.
	stagingMu    sync.Mutex
	stagingCache []*stagingBuffer
	stagingWk    chan *driver.WorkItem
)

func init() {
	n := runtime.GOMAXPROCS(-1)
	staging = make(chan *stagingBuffer, n)
	for i := 0; i < n; i++ {
		s, err := newStaging(stagingBlock * stagingNBit)
		if err != nil {
			s = &stagingBuffer{}
		}
		staging <- s
	}
	stagingCache = make([]*stagingBuffer, 0, n)
	stagingWk = make(chan *driver.WorkItem, 1)
	stagingWk <- &driver.WorkItem{Work: make([]driver.CmdBuffer, 0, n)}
}

// commitStaging executes all pending Texture copies.
// It blocks until execution completes.
func commitStaging() (err error) {
	stagingMu.Lock()
	swk := <-stagingWk

	// This deferral correctly clears global
	// state, regardless of the outcome.
	// Code below ensures that the command
	// buffers are reset if necessary.
	defer func() {
		for _, x := range stagingCache {
			x.bm.Clear()
			x.drainPending(err != nil)
			staging <- x
		}
		stagingCache = stagingCache[:0]
		swk.Work = swk.Work[:0]
		stagingWk <- swk
		stagingMu.Unlock()
	}()

	n := cap(staging)
	for i := 0; i < n; i++ {
		stagingCache = append(stagingCache, <-staging)
	}

	for i, x := range stagingCache {
		wk := <-x.wk
		if !wk.Work[0].IsRecording() {
			if len(x.pend) != 0 {
				// This should never happen.
				panic("commitStaging: pending copies while not recording")
			}
		} else if err = wk.Work[0].End(); err != nil {
			x.wk <- wk
			for _, x := range swk.Work {
				// Need to reset these since
				// they won't be committed.
				x.Reset()
			}
			for _, x := range stagingCache[i+1:] {
				// Need to reset these since
				// they won't be ended.
				wk := <-x.wk
				wk.Work[0].Reset()
				x.wk <- wk
			}
			return
		} else {
			swk.Work = append(swk.Work, wk.Work[0])
		}
		x.wk <- wk
	}

	if len(swk.Work) == 0 {
		return
	}
	if err = ctxt.GPU().Commit(swk, stagingWk); err != nil {
		return
	}
	swk = <-stagingWk
	err, swk.Err = swk.Err, nil
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
	stagingBlock = 131072
	stagingNBit  = 32
)

// newStaging creates a new stagingBuffer with the
// given size in bytes.
// n must be greater than 0; it will be rounded up
// to a multiple of stagingBlock * stagingNBit.
func newStaging(n int) (*stagingBuffer, error) {
	if n <= 0 {
		panic("newStaging: n <= 0")
	}
	cb, err := ctxt.GPU().NewCmdBuffer()
	if err != nil {
		return nil, err
	}
	wk := make(chan *driver.WorkItem, 1)
	wk <- &driver.WorkItem{Work: []driver.CmdBuffer{cb}}
	n = (n + stagingBlock*stagingNBit - 1) &^ (stagingBlock*stagingNBit - 1)
	buf, err := ctxt.GPU().NewBuffer(int64(n), true, driver.UCopySrc|driver.UCopyDst)
	if err != nil {
		cb.Destroy()
		return nil, err
	}
	var bm bitm.Bitm[uint32]
	bm.Grow(n / stagingBlock / stagingNBit)
	return &stagingBuffer{wk, buf, bm, nil}, nil
}

// copyToView records a copy command that copies
// data from s's buffer into view.
// off must have been returned by a previous call
// to s.reserve (i.e., it must be a multiple of
// stagingBlock).
// Only the first mip level must be provided.
// If t is arrayed and view is the last view, then
// the buffer must contain the first level of
// every layer, in order and tightly packed.
func (s *stagingBuffer) copyToView(t *Texture, view int, off int64) (err error) {
	if t.param.Samples != 1 {
		return errors.New(texPrefix + "cannot copy data to MS texture")
	}
	if view < 0 || view >= len(t.views) {
		return errors.New(texPrefix + "view index out of bounds")
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
		return errors.New(texPrefix + "not enough buffer capacity for copying")
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

	wk.Work[0].CopyBufToImg(&driver.BufImgCopy{
		Buf:    s.buf,
		BufOff: off,
		// TODO: Stride[0] must be 256-byte aligned.
		Stride: [2]int{t.param.Dim3D.Width, t.param.Dim3D.Height},
		Img:    t.views[view].Image(),
		ImgOff: driver.Off3D{},
		Layer:  il,
		Level:  0,
		Size:   t.param.Dim3D,
		Layers: nl,
		// TODO: Handle depth/stencil formats.
	})
	for i := 0; i < nl; i++ {
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
// stagingBlock).
func (s *stagingBuffer) copyFromView(t *Texture, view int, off int64) (err error) {
	if t.param.Samples != 1 {
		return errors.New(texPrefix + "cannot copy data from MS texture")
	}
	if view < 0 || view >= len(t.views) {
		return errors.New(texPrefix + "view index out of bounds")
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
		return errors.New(texPrefix + "not enough buffer capacity for copying")
	}
	// Need separate transitions if not all
	// layers are in the same layout.
	// TODO: Maybe try to merge contiguous
	// layers that share the same layout.
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
		// TODO: Consider caching this on s
		// (or t; see Texture.Transition).
		xs := make([]driver.Transition, nl)
		img := t.views[view].Image()
		for i := 0; i < nl; i++ {
			xs = append(xs, driver.Transition{
				Barrier: driver.Barrier{
					SyncBefore:   driver.SNone,
					SyncAfter:    driver.SCopy,
					AccessBefore: driver.ANone,
					AccessAfter:  driver.ACopyRead,
				},
				LayoutBefore: before[i],
				LayoutAfter:  driver.LCopySrc,
				Img:          img,
				Layer:        il + i,
				Layers:       1,
				Level:        0,
				Levels:       1, // TODO
			})
		}
		wk.Work[0].Transition(xs)
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

	wk.Work[0].CopyImgToBuf(&driver.BufImgCopy{
		Buf:    s.buf,
		BufOff: off,
		// TODO: Stride[0] must be 256-byte aligned.
		Stride: [2]int{t.param.Dim3D.Width, t.param.Dim3D.Height},
		Img:    t.views[view].Image(),
		ImgOff: driver.Off3D{},
		Layer:  il,
		Level:  0,
		Size:   t.param.Dim3D,
		Layers: nl,
		// TODO: Handle depth/stencil formats.
	})
	for i := 0; i < nl; i++ {
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
// stagingBlock).
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
	if off%stagingBlock != 0 {
		panic("stagingBuffer.unstage: misaligned off")
	}
	n = copy(dst, s.buf.Bytes()[off:])
	ib := int(off) / stagingBlock
	nb := (n + stagingBlock - 1) / stagingBlock
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
	n = (n + stagingBlock - 1) / stagingBlock
	idx, ok := s.bm.SearchRange(n)
	if !ok {
		if err = s.commit(); err != nil {
			return
		}
		// TODO: Consider using idx 0 instead.
		idx = s.bm.Len()
		n := (n + stagingNBit - 1) / stagingNBit
		s.bm.Grow(n)
		// TODO: Make buffer cap bounds configurable.
		n = n * stagingBlock * stagingNBit
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
	off = int64(idx) * stagingBlock
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
