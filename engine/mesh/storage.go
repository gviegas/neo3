// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"errors"
	"io"
	"sync"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/engine/internal/ctxt"
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
		n := c / (blockSize * spanMapNBit)
		if n > int64(^uint(0)>>1) || c != n*(blockSize*spanMapNBit) {
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
	sync.Mutex
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
	b.Lock()
	defer b.Unlock()
	nb := (byteLen + (blockSize - 1)) &^ (blockSize - 1)
	ns := nb / blockSize
	is, ok := b.spanMap.SearchRange(ns)
	if !ok {
		// TODO: Reconsider the growth strategy here.
		// Currently, it assumes that SetBuffer will
		// be called with a sensibly sized buffer and
		// that reallocations will not happen often,
		// so it optimizes for space.
		nplus := (ns + (spanMapNBit - 1)) / spanMapNBit
		bcap := int64(b.spanMap.Len()+nplus*spanMapNBit) * blockSize
		buf, err := ctxt.GPU().NewBuffer(bcap, true, driver.UVertexData|driver.UIndexData)
		if err != nil {
			return span{}, err
		}
		if b.buf != nil {
			// TODO: Do this copy through the GPU.
			copy(buf.Bytes(), b.buf.Bytes())
			b.buf.Destroy()
		}
		b.buf = buf
		is = b.spanMap.Grow(nplus)
	}
	slc := b.buf.Bytes()[is*blockSize : is*blockSize+byteLen]
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
func (b *meshBuffer) newEntry(data *PrimitiveData, srcs []io.ReadSeeker) (p Primitive, err error) {
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
			err = errors.New(prefix + "invalid driver.IndexFmt value")
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
			panic("TODO: free spans")
			return
		}
		var conv io.Reader
		if conv, err = sem.conv(fmt, src, prim.count); err != nil {
			panic("TODO: free spans")
			return
		}
		fmt = sem.format()
		prim.vertex[i].format = fmt
		if prim.vertex[i].span, err = b.store(conv, prim.count*fmt.Size()); err != nil {
			panic("TODO: free spans")
			return
		}
	}
	if i, ok := b.primMap.Search(); !ok {
		// TODO: Grow exponentially.
		var z [primMapNBit]primitive
		b.prims = append(b.prims, z[:]...)
		p.index = b.primMap.Grow(1)
	} else {
		p.index = i
	}
	b.primMap.Set(p.index)
	// Currently, p.bufIdx is always 0.
	return
}

// link links a primitive entry to another.
// This is only relevant for meshes that contain multiple
// primitives.
func (b *meshBuffer) link(prim Primitive, next Primitive) {
	b.Lock()
	defer b.Unlock()
	if prim.bufIdx != next.bufIdx {
		panic("attempt to link primitives from different buffers")
	}
	b.prims[prim.index].next = next.index
}

// freeEntry removes a primitive from the buffer.
// Any span held by prim is made available for use when
// creating new entries (it does not free GPU memory).
func (b *meshBuffer) freeEntry(prim Primitive) {
	b.primMap.Unset(prim.index)
	b._freeEntry(&b.prims[prim.index])
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

// span granularity.
const blockSize = 512

// byteStart computes the span's first byte.
func (s span) byteStart() int { return s.start * blockSize }

// byteEnd computes the span's one-past-the-end byte.
func (s span) byteEnd() int { return s.end * blockSize }

// byteLen computes the span's byte length.
func (s span) byteLen() int { return (s.end - s.start) * blockSize }
