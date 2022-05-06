// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"fmt"
	"testing"

	"github.com/gviegas/scene/driver"
)

func TestSampler(t *testing.T) {
	cases := [...]struct {
		min, mag, mip  driver.Filter
		maxAniso       int
		minLOD, maxLOD float32
		u, v, w        driver.AddrMode
		cmp            driver.CmpFunc
	}{
		{driver.FNearest, driver.FNearest, driver.FNoMipmap, 1, 0, 0.25, driver.AWrap, driver.AWrap, driver.AWrap, driver.CNever},
		{driver.FLinear, driver.FLinear, driver.FNoMipmap, 1, 0, 0.25, driver.AWrap, driver.AMirror, driver.AClamp, driver.CLess},
		{driver.FLinear, driver.FLinear, driver.FLinear, 1, 0, 10, driver.AMirror, driver.AWrap, driver.AWrap, driver.CEqual},
		{driver.FLinear, driver.FNearest, driver.FNearest, 1, 0, 11, driver.AClamp, driver.AWrap, driver.AClamp, driver.CLessEqual},
		{driver.FNearest, driver.FLinear, driver.FNearest, 1, 0, 12, driver.AMirror, driver.AMirror, driver.AMirror, driver.CGreater},
		{driver.FNearest, driver.FNearest, driver.FNearest, 1, 0, 0, driver.AClamp, driver.AMirror, driver.AWrap, driver.CNotEqual},
		{driver.FNearest, driver.FNearest, driver.FLinear, 4, 0, 1, driver.AWrap, driver.AWrap, driver.AWrap, driver.CGreaterEqual},
		{driver.FLinear, driver.FLinear, driver.FLinear, 16, 0, 2, driver.AClamp, driver.AClamp, driver.AClamp, driver.CAlways},
	}
	zs := sampler{}
	for _, c := range cases {
		call := fmt.Sprintf("tDrv.NewSampler(%v)", c)
		// NewSampler.
		if s, err := tDrv.NewSampler(c.min, c.mag, c.mip, c.maxAniso, c.minLOD, c.maxLOD, c.u, c.v, c.w, c.cmp); err == nil {
			if s == nil {
				t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
			}
			s := s.(*sampler)
			if s.d != &tDrv {
				t.Errorf("%s: s.d\nhave %p\nwant %p", call, s, &tDrv)
			}
			if s.splr == zs.splr {
				t.Errorf("%s: s.splr\nhave %v\nwant valid handle", call, s.splr)
			}
			// Destroy.
			s.Destroy()
			if *s != zs {
				t.Errorf("s.Destroy(): s\nhave %v\nwant %v", s, zs)
			}
		} else if s != nil {
			t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, s, err, err)
		} else {
			t.Logf("(error) %s: %v", s, err)
		}
	}
}
