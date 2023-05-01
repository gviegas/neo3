// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package skin

import (
	"fmt"
	"testing"
)

func BenchmarkNew(b *testing.B) {
	for _, x := range [...][2]int{
		{1, 0},
		{4, 0},
		{4, 1},
		{4, 2},
		{4, 3},
		{15, 0},
		{15, 7},
		{15, 14},
		{64, 0},
		{64, 32},
		{64, 63},
		{128, 0},
		{128, 64},
		{128, 127},
		{255, 0},
		{255, 127},
		{255, 254},
		{65535, 0},
		{65535, 32767},
		{65535, 65534},
	} {
		in := dummyJoints(x[0], x[1])
		s := fmt.Sprintf("{len=%d,dep=%d}", x[0], x[1])

		b.Run(s, func(b *testing.B) {
			if _, err := New(in); err != nil {
				b.Fatalf("New failed:\n%#v", err)
			}
		})
	}
}

func BenchmarkNewReversed(b *testing.B) {
	for _, x := range [...]int{0, 1, 2, 15, 31, 63, 127, 255, 256, 1023, 8192, 32767, 65534} {
		in := dummyJointsRev(x)
		s := fmt.Sprintf("{depth=%d}", x)

		b.Run(s, func(b *testing.B) {
			if _, err := New(in); err != nil {
				b.Fatalf("New failed:\n%#v", err)
			}
		})
	}
}
