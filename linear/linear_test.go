// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package linear

import (
	"math"
	"testing"
)

func TestV(t *testing.T) {
	var u V3
	v := V3{1, 2, 4}
	w := V3{0, -1, 2}

	if u.Add(&v, &w); u != (V3{1, 1, 6}) {
		t.Fatalf("V3.Add\nhave %v\nwant [1 1 6]", u)
	}
	if u.Sub(&v, &w); u != (V3{1, 3, 2}) {
		t.Fatalf("V3.Sub\nhave %v\nwant [1 3 2]", u)
	}
	if u.Scale(-1, &v); u != (V3{-1, -2, -4}) {
		t.Fatalf("V3.Scale\nhave %v\nwant [-1 -2 -4]", u)
	}
	if u.Scale(2, &w); u != (V3{0, -2, 4}) {
		t.Fatalf("V3.Scale\nhave %v\nwant [0 -2 4]", u)
	}
	if d := v.Dot(&w); d != 6 {
		t.Fatalf("V3.Dot\nhave %v\nwant 6\n", d)
	}
	if d := v.Dot(&v); d != 21 {
		t.Fatalf("V3.Dot\nhave %v\nwant 21\n", d)
	}
	if l := v.Len(); l != float32(math.Sqrt(21)) {
		t.Fatalf("V3.Len\nhave %v\nwant %v\n", l, math.Sqrt(21))
	}
	if l := w.Len(); l != float32(math.Sqrt(5)) {
		t.Fatalf("V3.Len\nhave %v\nwant %v\n", l, math.Sqrt(5))
	}

	v = V3{0, 0, -2}
	w = V3{0, 4, 0}

	if v.Norm(&v); v != (V3{0, 0, -1}) {
		t.Fatalf("V3.Norm\nhave %v\nwant [0 0 -1]", v)
	}
	if w.Norm(&w); w != (V3{0, 1, 0}) {
		t.Fatalf("V3.Norm\nhave %v\nwant [0 1 0]", w)
	}
	if u.Cross(&v, &w); u != (V3{1, 0, 0}) {
		t.Fatalf("V3.Cross\nhave %v\nwant [1 0 0]", u)
	}
	if u.Cross(&w, &v); u != (V3{-1, 0, 0}) {
		t.Fatalf("V3.Cross\nhave %v\nwant [-1 0 0]", u)
	}
}
