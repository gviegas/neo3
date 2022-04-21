// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

import (
	"math"
	"testing"
)

func TestV(t *testing.T) {
	v := V3{1, 2, 4}
	w := V3{0, -1, 2}

	if u := AddV3(v, w); u != (V3{1, 1, 6}) {
		t.Fatalf("AddV3\nhave %v\nwant [1 1 6]", u)
	}
	if u := SubV3(v, w); u != (V3{1, 3, 2}) {
		t.Fatalf("SubV3\nhave %v\nwant [1 3 2]", u)
	}
	if u := ScaleV3(-1, v); u != (V3{-1, -2, -4}) {
		t.Fatalf("ScaleV3\nhave %v\nwant [-1 -2 -4]", u)
	}
	if u := ScaleV3(2, w); u != (V3{0, -2, 4}) {
		t.Fatalf("ScaleV3\nhave %v\nwant [0 -2 4]", u)
	}
	if d := DotV3(v, w); d != 6 {
		t.Fatalf("DotV3\nhave %v\nwant 6\n", d)
	}
	if d := DotV3(v, v); d != 21 {
		t.Fatalf("DotV3\nhave %v\nwant 21\n", d)
	}
	if l := LenV3(v); l != float32(math.Sqrt(21)) {
		t.Fatalf("LenV3\nhave %v\nwant %v\n", l, math.Sqrt(21))
	}
	if l := LenV3(w); l != float32(math.Sqrt(5)) {
		t.Fatalf("LenV3\nhave %v\nwant %v\n", l, math.Sqrt(5))
	}

	v = V3{0, 0, -2}
	w = V3{0, 4, 0}

	if v = NormV3(v); v != (V3{0, 0, -1}) {
		t.Fatalf("NormV3\nhave %v\nwant [0 0 -1]", v)
	}
	if w = NormV3(w); w != (V3{0, 1, 0}) {
		t.Fatalf("NormV3\nhave %v\nwant [0 1 0]", w)
	}
	if u := Cross(v, w); u != (V3{1, 0, 0}) {
		t.Fatalf("Cross\nhave %v\nwant [1 0 0]", u)
	}
	if u := Cross(w, v); u != (V3{-1, 0, 0}) {
		t.Fatalf("Cross\nhave %v\nwant [-1 0 0]", u)
	}
}
