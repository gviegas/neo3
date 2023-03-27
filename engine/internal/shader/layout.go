// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package shader

import (
	"time"
	"unsafe"

	"github.com/gviegas/scene/driver"
	"github.com/gviegas/scene/linear"
)

// FrameLayout is the layout of per-frame, global data.
// It is defined as follows:
//
//	[0:16]  | view-projection matrix
//	[16:32] | view matrix
//	[32:48] | projection matrix
//	[48]    | elapsed time in seconds
//	[49]    | normalized random value
//	[50]    | viewport's x
//	[51]    | viewport's y
//	[52]    | viewport's width
//	[53]    | viewport's height
//	[54]    | viewport's near plane
//	[55]    | viewport's far plane
//	[56:64] | (unused)
//
// NOTE: This layout is likely to change.
type FrameLayout [64]float32

// SetVP sets the view-projection matrix.
func (l *FrameLayout) SetVP(m *linear.M4) { copyM4(l[:16], m) }

// SetV sets the view matrix.
func (l *FrameLayout) SetV(m *linear.M4) { copyM4(l[16:32], m) }

// SetP sets the projection matrix.
func (l *FrameLayout) SetP(m *linear.M4) { copyM4(l[32:48], m) }

// SetTime sets the elapsed time.
func (l *FrameLayout) SetTime(d time.Duration) { l[48] = float32(d.Seconds()) }

// SetRand sets the normalized random value.
func (l *FrameLayout) SetRand(v float32) { l[49] = v }

// SetBounds sets the viewport bounds.
func (l *FrameLayout) SetBounds(b *driver.Viewport) {
	l[50] = b.X
	l[51] = b.Y
	l[52] = b.Width
	l[53] = b.Height
	l[54] = b.Znear
	l[55] = b.Zfar
}

func copyM4(dst []float32, m *linear.M4) {
	copy(dst, unsafe.Slice((*float32)(unsafe.Pointer(m)), 16))
}
