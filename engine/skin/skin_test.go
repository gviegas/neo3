// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package skin

import (
	"strconv"
	"strings"
	"testing"

	"github.com/gviegas/scene/linear"
)

func dummyJoints(len, depth int) []Joint {
	js := make([]Joint, 0, len)
	for i := 0; i < len; i++ {
		js = append(js, Joint{
			Name:   "Joint " + strconv.Itoa(i),
			JM:     linear.M4{{1}, {1: 1}, {2: 1}, {3: 1}},
			IBM:    linear.M4{{1}, {1: 1}, {2: 1}, {float32(i % (depth + 1)), 0, 0, 1}},
			Parent: i%(depth+1) - 1,
		})
	}
	return js
}

func TestNew(t *testing.T) {
	for _, x := range [...][2]int{
		{1, 0},
		{2, 0},
		{2, 1},
		{4, 0},
		{4, 1},
		{4, 2},
		{4, 3},
		{15, 0},
		{15, 1},
		{15, 2},
		{15, 3},
		{15, 8},
		{15, 14},
		{127, 0},
		{127, 1},
		{127, 63},
		{127, 64},
		{127, 65},
		{127, 125},
		{127, 126},
		{200, 0},
		{200, 1},
		{200, 16},
		{200, 127},
		{200, 199},
		{255, 0},
		{255, 1},
		{255, 128},
		{255, 254},
		{65535, 0},
		{65535, 1},
		{65535, 2},
		{65535, 15},
		{65535, 256},
		{65535, 32767},
		{65535, 60000},
		{65535, 65534},
	} {
		in := dummyJoints(x[0], x[1])
		sk, err := New(in)
		if sk == nil || err != nil {
			t.Fatalf("New:\nhave %v, %#v\nwant non-nil, nil", sk, err)
		}
		if x, y := len(sk.joints), len(in); x != y {
			t.Fatalf("New: len(Skin.joints)\nhave %d\nwant %d", x, y)
		}
		var cnt int
		for i := range sk.joints {
			orig := sk.joints[i].orig
			if orig < 0 || orig >= len(in) {
				t.Fatalf("New: bad Skin.joints.orig index\nhave %d\nwant [0, %d)", orig, len(in))
			}
			if x := sk.joints[i].name; x != in[orig].Name {
				t.Fatalf("New: bad Skin.joints.name\nhave %s\nwant %s", x, in[orig].Name)
			}
			if x := sk.joints[i].ibm; x < -1 || x >= len(sk.ibm) {
				t.Fatalf("New: bad Skin.joints.ibm index\nhave %d\nwant [-1, %d)", x, len(sk.ibm))
			}
			if x := sk.joints[i].parent; x < -1 || x >= len(in) {
				t.Fatalf("New: bad Skin.joints.parent index\nhave %d\nwant [-1, %d)", x, len(in))
			}
			cnt += 1 + orig
		}
		if x := (x[0]*x[0] + x[0]) / 2; x != cnt {
			t.Fatalf("New: bad Skin.joints.orig count\nhave %d\nwant %d", cnt, x)
		}
	}
}

func TestNewFail(t *testing.T) {
	var sk *Skin
	var err error

	checkFail := func(reason string) {
		if sk != nil || err == nil {
			t.Fatalf("New:\nhave %v, %#v\nwant nil, non-nil", sk, err)
		}
		if x := err.Error(); !strings.HasSuffix(x, reason) {
			t.Fatalf("New: error.Error()\nhave \"%s\"\nwant \"%s%s\"", x, prefix, reason)
		}
	}

	sk, err = New([]Joint{})
	checkFail("[]Joint length is 0")
	sk, err = New(nil)
	checkFail("[]Joint length is 0")

	j1 := dummyJoints(1, 0)
	j1[0].Parent = 1
	sk, err = New(j1)
	checkFail("Joint.Parent out of bounds")
	j1[0].Parent = 0
	sk, err = New(j1)
	checkFail("Joint.Parent refers to itself")

	j20 := dummyJoints(20, 5)
	j20[19].Parent = 20
	sk, err = New(j20)
	checkFail("Joint.Parent out of bounds")
	j20[10].Parent = 10
	sk, err = New(j20)
	checkFail("Joint.Parent refers to itself")
}
