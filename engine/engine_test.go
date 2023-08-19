// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"math"
	"testing"

	"gviegas/neo3/engine/internal/ctxt"
	"gviegas/neo3/linear"
)

func TestCtxt(t *testing.T) {
	drv := ctxt.Driver()
	if drv == nil {
		t.Fatal("ctxt.Driver: unexpected nil driver.Driver")
	}
	gpu := ctxt.GPU()
	if gpu == nil {
		t.Fatal("ctxt.GPU: unexpected nil driver.GPU")
	}
	if d := ctxt.Driver(); d != drv {
		t.Fatalf("ctxt.Driver: ctxt mismatch\nhave %v\nwant %v", d, drv)
	}
	if u := ctxt.GPU(); u != gpu {
		t.Fatalf("ctxt.GPU: ctxt mismatch\nhave %v\nwant %v", u, gpu)
	}
	if u, err := drv.Open(); u != gpu || err != nil {
		t.Fatalf("drv.Open: ctxt mismatch\nhave %v, %v\nwant %v, nil", u, err, gpu)
	}
}

func TestLight(t *testing.T) {
	t.Run("Sun", func(t *testing.T) {
		sun := SunLight{
			Direction: linear.V3{0, 0, -1},
			Intensity: 100_000,
			R:         1,
			G:         1,
			B:         1,
		}
		light := sun.Light()
		if other := sun.Light(); light != other {
			t.Fatalf("SunLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// Intensity should not go below 0.
		sun.Intensity = -sun.Intensity
		if other := sun.Light(); light == other {
			t.Fatalf("SunLight.Light: created Lights don't differ\n1st: %v\n2nd: %v", light, other)
		} else {
			light = other
		}
		sun.Intensity = 0
		if other := sun.Light(); light != other {
			t.Fatalf("SunLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
	})

	t.Run("Point", func(t *testing.T) {
		point := PointLight{
			Position:  linear.V3{},
			Range:     0.5,
			Intensity: 1,
			R:         1,
			G:         1,
			B:         1,
		}
		light := point.Light()
		if other := point.Light(); light != other {
			t.Fatalf("PointLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// Intensity should not go below 0.
		point.Intensity = -1
		if other := point.Light(); light == other {
			t.Fatalf("PointLight.Light: created Lights don't differ\n1st: %v\n2nd: %v", light, other)
		} else {
			light = other
		}
		point.Intensity = 0
		if other := point.Light(); light != other {
			t.Fatalf("PointLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
	})

	t.Run("Spot", func(t *testing.T) {
		spot := SpotLight{
			Direction:  linear.V3{0, 0, -1},
			Position:   linear.V3{},
			InnerAngle: math.Pi / 16,
			OuterAngle: math.Pi / 4,
			Range:      36,
			Intensity:  100,
			R:          1,
			G:          1,
			B:          1,
		}
		light := spot.Light()
		if other := spot.Light(); light != other {
			t.Fatalf("SpotLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// Intensity and InnerAngle should not go below 0.
		// OuterAngle should not go above Pi/2.
		spot.Intensity = -123
		spot.InnerAngle = -math.Pi
		spot.OuterAngle = math.Pi
		if other := spot.Light(); light == other {
			t.Fatalf("SpotLight.Light: created Lights don't differ\n1st: %v\n2nd: %v", light, other)
		} else {
			light = other
		}
		spot.Intensity = 0
		spot.InnerAngle = 0
		spot.OuterAngle = math.Pi / 2
		if other := spot.Light(); light != other {
			t.Fatalf("SpotLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
		// InnerAngle should be less than OuterAngle.
		spot.InnerAngle = math.Pi / 4
		spot.OuterAngle = math.Pi/4 + 1e-6
		light = spot.Light()
		spot.OuterAngle = spot.InnerAngle
		if other := spot.Light(); light != other {
			t.Fatalf("SpotLight.Light: created Lights differ\n1st: %v\n2nd: %v", light, other)
		}
	})
}
