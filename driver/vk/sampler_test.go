// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"gviegas/neo3/driver"
)

func TestSampler(t *testing.T) {
	cases := [...]driver.Sampling{
		{
			Min:      driver.FNearest,
			Mag:      driver.FNearest,
			Mipmap:   driver.FNoMipmap,
			AddrU:    driver.AWrap,
			AddrV:    driver.AWrap,
			AddrW:    driver.AWrap,
			MaxAniso: 1,
			Cmp:      driver.CNever,
			MinLOD:   0,
			MaxLOD:   0.25,
		},
		{
			Min:      driver.FLinear,
			Mag:      driver.FLinear,
			Mipmap:   driver.FNoMipmap,
			AddrU:    driver.AWrap,
			AddrV:    driver.AMirror,
			AddrW:    driver.AClamp,
			MaxAniso: 1,
			Cmp:      driver.CLess,
			MinLOD:   0,
			MaxLOD:   0.25,
		},
		{
			Min:      driver.FLinear,
			Mag:      driver.FLinear,
			Mipmap:   driver.FLinear,
			AddrU:    driver.AMirror,
			AddrV:    driver.AWrap,
			AddrW:    driver.AWrap,
			MaxAniso: 1,
			Cmp:      driver.CEqual,
			MinLOD:   0,
			MaxLOD:   10,
		},
		{
			Min:      driver.FLinear,
			Mag:      driver.FNearest,
			Mipmap:   driver.FNearest,
			AddrU:    driver.AClamp,
			AddrV:    driver.AWrap,
			AddrW:    driver.AClamp,
			MaxAniso: 1,
			Cmp:      driver.CLessEqual,
			MinLOD:   0,
			MaxLOD:   11,
		},
		{
			Min:      driver.FNearest,
			Mag:      driver.FLinear,
			Mipmap:   driver.FNearest,
			AddrU:    driver.AMirror,
			AddrV:    driver.AMirror,
			AddrW:    driver.AMirror,
			MaxAniso: 1,
			Cmp:      driver.CGreater,
			MinLOD:   0,
			MaxLOD:   12,
		},
		{
			Min:      driver.FNearest,
			Mag:      driver.FNearest,
			Mipmap:   driver.FNearest,
			AddrU:    driver.AClamp,
			AddrV:    driver.AMirror,
			AddrW:    driver.AWrap,
			MaxAniso: 1,
			Cmp:      driver.CNotEqual,
			MinLOD:   0,
			MaxLOD:   0,
		},
		{
			Min:      driver.FNearest,
			Mag:      driver.FNearest,
			Mipmap:   driver.FLinear,
			AddrU:    driver.AWrap,
			AddrV:    driver.AWrap,
			AddrW:    driver.AWrap,
			MaxAniso: 4,
			Cmp:      driver.CGreaterEqual,
			MinLOD:   0,
			MaxLOD:   1,
		},
		{
			Min:      driver.FLinear,
			Mag:      driver.FLinear,
			Mipmap:   driver.FLinear,
			AddrU:    driver.AClamp,
			AddrV:    driver.AClamp,
			AddrW:    driver.AClamp,
			MaxAniso: 16,
			Cmp:      driver.CAlways,
			MinLOD:   0,
			MaxLOD:   2,
		},
	}
	zs := sampler{}
	for i := range cases {
		call := fmt.Sprintf("tDrv.NewSampler(%v)", cases[i])
		// NewSampler.
		if s, err := tDrv.NewSampler(&cases[i]); err == nil {
			if s == nil {
				t.Fatalf("%s\nhave nil, nil\nwant non-nil, nil", call)
			}
			s := s.(*sampler)
			if s.d != &tDrv {
				t.Fatalf("%s: s.d\nhave %p\nwant %p", call, s, &tDrv)
			}
			if s.splr == zs.splr {
				t.Fatalf("%s: s.splr\nhave %v\nwant valid handle", call, s.splr)
			}
			// Destroy.
			s.Destroy()
			if *s != zs {
				t.Fatalf("s.Destroy(): s\nhave %v\nwant %v", s, zs)
			}
		} else if s != nil {
			t.Fatalf("%s\nhave %p, %v\nwant nil, %v", call, s, err, err)
		} else {
			t.Logf("(error) %s: %v", s, err)
		}
	}
}
