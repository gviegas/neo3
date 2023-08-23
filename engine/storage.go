// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/internal/bitm"
)

// Global mesh storage.
var storage meshBuffer

// SetMeshBuffer sets the GPU buffer into which mesh data
// will be stored.
// The buffer must be host-visible, its usage must include
// both driver.UVertexData and driver.UIndexData, and its
// capacity must be a multiple of 16384 bytes.
// It returns the replaced buffer, if any.
//
// NOTE: Calls to this function invalidate all previously
// created meshes.
func SetMeshBuffer(buf driver.Buffer) driver.Buffer {
	storage.Lock()
	defer storage.Unlock()
	switch buf {
	case storage.buf:
		return nil
	case nil:
		storage.spanMap = bitm.Bitm[uint32]{}
		storage.primMap = bitm.Bitm[uint16]{}
		storage.prims = nil
	default:
		c := buf.Cap()
		n := c / (spanBlock * spanMapNBit)
		if n > int64(^uint(0)>>1) || c != n*(spanBlock*spanMapNBit) {
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
	sync.RWMutex
}

const (
	spanMapNBit = 32
	primMapNBit = 16
)

// store reads byteLen bytes from src and writes the data
// into the GPU buffer.
// It returns a span identifying the buffer range where
// the data was stored.
func (b *meshBuffer) store(src io.Reader, byteLen int) (span, error) {
	nb := (byteLen + (spanBlock - 1)) &^ (spanBlock - 1)
	ns := nb / spanBlock
	is, ok := b.spanMap.SearchRange(ns)
	if !ok {
		// TODO: Reconsider the growth strategy here.
		// Currently it assumes that SetMeshBuffer will
		// be called with a sensibly sized buffer and
		// that reallocations will not happen often,
		// so it optimizes for space.
		nplus := (ns + (spanMapNBit - 1)) / spanMapNBit
		bcap := int64(b.spanMap.Len()+nplus*spanMapNBit) * spanBlock
		buf, err := ctxt.GPU().NewBuffer(bcap, true, driver.UVertexData|driver.UIndexData)
		if err != nil {
			return span{}, err
		}
		if b.buf != nil {
			// TODO: Do this copy through the GPU
			// (requires driver.UCopySrc/UCopyDst).
			copy(buf.Bytes(), b.buf.Bytes())
			b.buf.Destroy()
		}
		b.buf = buf
		is = b.spanMap.Grow(nplus)
	}
	slc := b.buf.Bytes()[is*spanBlock : is*spanBlock+byteLen]
	for len(slc) > 0 {
		switch n, err := src.Read(slc); {
		case n > 0:
			slc = slc[n:]
		case err != nil:
			return span{}, err
		}
	}
	for i := 0; i < ns; i++ {
		b.spanMap.Set(is + i)
	}
	return span{is, is + ns}, nil
}

// newEntry creates a new entry in the buffer containing
// the primitive specified by data.
func (b *meshBuffer) newEntry(data *PrimitiveData, srcs []io.ReadSeeker) (p int, err error) {
	prim := primitive{
		topology: data.Topology,
		mask:     data.SemanticMask,
		next:     -1,
	}
	if data.IndexCount != 0 {
		prim.count = data.IndexCount
		prim.index.format = data.Index.Format
		var isz int
		switch prim.index.format {
		case driver.Index16:
			isz = 2
		case driver.Index32:
			isz = 4
		default:
			err = errors.New(meshPrefix + "undefined driver.IndexFmt constant")
		}
		src := srcs[data.Index.Src]
		off := data.Index.Offset
		if _, err = src.Seek(off, io.SeekStart); err != nil {
			return
		}
		if prim.index.span, err = b.store(src, prim.count*isz); err != nil {
			return
		}
	} else {
		prim.count = data.VertexCount
	}
	for i := range data.Semantics {
		sem := Semantic(1 << i)
		if data.SemanticMask&sem == 0 {
			continue
		}
		fmt := data.Semantics[i].Format
		src := srcs[data.Semantics[i].Src]
		off := data.Semantics[i].Offset
		if _, err = src.Seek(off, io.SeekStart); err != nil {
			b._freeEntry(&prim)
			return
		}
		var conv io.Reader
		if conv, err = sem.conv(fmt, src, data.VertexCount); err != nil {
			b._freeEntry(&prim)
			return
		}
		fmt = sem.format()
		prim.vertex[i].format = fmt
		if prim.vertex[i].span, err = b.store(conv, data.VertexCount*fmt.Size()); err != nil {
			b._freeEntry(&prim)
			return
		}
	}
	if i, ok := b.primMap.Search(); !ok {
		// TODO: Grow exponentially.
		var z [primMapNBit]primitive
		b.prims = append(b.prims, z[:]...)
		p = b.primMap.Grow(1)
	} else {
		p = i
	}
	b.primMap.Set(p)
	b.prims[p] = prim
	return
}

// next returns the next primitive in the list.
// If prim has no subsequent primitive (i.e., it was not
// linked to another primitive), then ok will be false.
// This is only relevant for meshes that contain multiple
// primitives.
func (b *meshBuffer) next(prim int) (p int, ok bool) {
	if i := b.prims[prim].next; i >= 0 {
		p = i
		ok = true
	}
	return
}

// freeEntry removes a primitive from the buffer.
// Any span held by prim is made available for use when
// creating new entries (it does not free GPU memory).
func (b *meshBuffer) freeEntry(prim int) {
	b.primMap.Unset(prim)
	b._freeEntry(&b.prims[prim])
}

func (b *meshBuffer) _freeEntry(prim *primitive) {
	// This ignores the mask and checks for
	// empty spans instead, so it is safe to
	// call from newEntry when it fails with
	// a partially set primitive.
	for i := range prim.vertex {
		for j := prim.vertex[i].start; j < prim.vertex[i].end; j++ {
			b.spanMap.Unset(j)
		}
	}
	for i := prim.index.start; i < prim.index.end; i++ {
		b.spanMap.Unset(i)
	}
	*prim = primitive{}
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

// span block size.
const spanBlock = 512

// byteStart computes the span's first byte.
func (s span) byteStart() int { return s.start * spanBlock }

// byteEnd computes the span's one-past-the-end byte.
func (s span) byteEnd() int { return s.end * spanBlock }

// byteLen computes the span's byte length.
func (s span) byteLen() int { return (s.end - s.start) * spanBlock }

// String implements fmt.Stringer.
func (s span) String() string {
	return fmt.Sprintf("{%d(%dB) %d(%dB)}", s.start, s.byteStart(), s.end, s.byteEnd())
}
