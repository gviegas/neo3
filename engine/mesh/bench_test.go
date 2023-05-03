// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package mesh

import (
	"io"
	"testing"

	"gviegas/neo3/driver"
	"gviegas/neo3/engine/internal/ctxt"
)

const (
	nbufBench  = 64 << 20
	ntrisBench = 1000
)

// TODO: Currently, New locks the storage for writing
// during span searching, buffer growth/copying, and
// new data copying. The last step (new data copying)
// can be done with just a reading lock.
// Consider splitting the meshBuffer methods so New
// can release the writing lock as soon as all spans
// it needs have been reserved, and then copy the new
// data while holding a RLock.

func BenchmarkNewGrow(b *testing.B) {
	if buf := SetBuffer(nil); buf != nil {
		buf.Destroy()
	}
	data := dummyData1(ntrisBench)
	b.Run("x", func(b *testing.B) {
		// Will grow the buffer on every iteration.
		// Expected to be very slow.
		b.RunParallel(func(bp *testing.PB) {
			for bp.Next() {
				if storage.buf != nil && storage.buf.Cap() > nbufBench {
					continue
				}
				for i := range data.Srcs {
					data.Srcs[i].Seek(0, io.SeekStart)
				}
				if _, err := New(&data); err != nil {
					b.Fatalf("New failed:\n%#v", err)
				}
			}
		})
	})
	b.Log("buf.Cap():", storage.buf.Cap())
	b.Log("spanMap.Rem()/Len():", storage.spanMap.Rem(), storage.spanMap.Len())
	b.Log("primMap.Rem()/Len():", storage.primMap.Rem(), storage.primMap.Len())
}

func BenchmarkNewPre(b *testing.B) {
	buf, err := ctxt.GPU().NewBuffer(nbufBench, true, driver.UVertexData|driver.UIndexData)
	if err != nil {
		b.Fatalf("driver.GPU.NewBuffer failed:\n%#v", err)
	}
	if buf = SetBuffer(buf); buf != nil {
		buf.Destroy()
	}
	data := dummyData1(ntrisBench)
	b.Run("x", func(b *testing.B) {
		// Will use pre-allocated memory.
		// Expected to be fast.
		b.RunParallel(func(bp *testing.PB) {
			for bp.Next() {
				if storage.buf != nil && storage.buf.Cap() > nbufBench {
					continue
				}
				for i := range data.Srcs {
					data.Srcs[i].Seek(0, io.SeekStart)
				}
				if _, err := New(&data); err != nil {
					b.Fatalf("New failed:\n%#v", err)
				}
			}
		})
	})
	b.Log("buf.Cap():", storage.buf.Cap())
	b.Log("spanMap.Rem()/Len():", storage.spanMap.Rem(), storage.spanMap.Len())
	b.Log("primMap.Rem()/Len():", storage.primMap.Rem(), storage.primMap.Len())
}

func BenchmarkNewFree(b *testing.B) {
	if buf := SetBuffer(nil); buf != nil {
		buf.Destroy()
	}
	data := dummyData1(ntrisBench)
	b.Run("x", func(b *testing.B) {
		// Will create and then free the mesh,
		// so its spans can be reused.
		// Expected to be reasonably fast.
		b.RunParallel(func(bp *testing.PB) {
			for bp.Next() {
				if storage.buf != nil && storage.buf.Cap() > nbufBench {
					continue
				}
				for i := range data.Srcs {
					data.Srcs[i].Seek(0, io.SeekStart)
				}
				m, err := New(&data)
				if err != nil {
					b.Fatalf("New failed:\n%#v", err)
				}
				m.Free()
			}
		})
	})
	b.Log("buf.Cap():", storage.buf.Cap())
	b.Log("spanMap.Rem()/Len():", storage.spanMap.Rem(), storage.spanMap.Len())
	b.Log("primMap.Rem()/Len():", storage.primMap.Rem(), storage.primMap.Len())
}
