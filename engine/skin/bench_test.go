// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package skin

import (
	"fmt"
	"sort"
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

func fromDummyJoints(in []Joint) []joint {
	out := make([]joint, 0, len(in))
	for i := range in {
		out = append(out, joint{
			name:   in[i].Name,
			parent: in[i].Parent,
			orig:   i,
		})
	}
	return out
}

func BenchmarkJointSort(b *testing.B) {
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
		out1, out2 := fromDummyJoints(in), fromDummyJoints(in)
		s := fmt.Sprintf("{len=%d,dep=%d}", x[0], x[1])

		b.Run("sort.Sort"+s, func(b *testing.B) {
			sort.Sort(jointSlice(out1))
		})

		b.Run("sort.Slice"+s, func(b *testing.B) {
			sort.Slice(out2, func(i, j int) bool {
				return out2[i].parent < out2[j].parent
			})
		})
	}
}
