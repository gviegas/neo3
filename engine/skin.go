// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"errors"
	"sort"

	"gviegas/neo3/linear"
)

const skinPrefix = "skin: "

func newSkinErr(reason string) error { return errors.New(skinPrefix + reason) }

// Skin defines skinning data.
type Skin struct {
	joints []joint
	// Only store inverse bind matrices that
	// are not the zero/identity matrix.
	ibm []linear.M4
	// Sorted such that every parent comes
	// before any of its descendants.
	hier []int

	// TODO: Descriptors; const buffer
	// (per-instance, most likely).
}

// joint defines a skin's joint.
type joint struct {
	name   string
	jm     linear.M4
	ibm    int
	parent int
}

// Joint describes a single joint in a skin.
// A joint hierarchy is defined by setting the Parent
// field to refer to another Joint's index within the
// slice presented to New.
// Joint.Parent can be set to -1 or less to indicate
// that the joint has no parent.
type Joint struct {
	Name   string
	JM     linear.M4
	IBM    linear.M4
	Parent int
}

// NewSkin creates a new skin from a joint hierarchy.
func NewSkin(joints []Joint) (*Skin, error) {
	n := len(joints)
	if n == 0 {
		return nil, newSkinErr("[]Joint length is 0")
	}

	js := make([]joint, 0, n)
	var ibm []linear.M4
	var zero, ident linear.M4
	ident.I()

	for i := range joints {
		pnt := joints[i].Parent
		switch {
		case pnt >= n:
			return nil, newSkinErr("Joint.Parent out of bounds")
		case pnt == i:
			return nil, newSkinErr("Joint.Parent refers to itself")
		case pnt < 0:
			pnt = -1
		}

		iibm := -1
		switch joints[i].IBM {
		case zero, ident:
		default:
			iibm = len(ibm)
			ibm = append(ibm, joints[i].IBM)
		}

		js = append(js, joint{
			name:   joints[i].Name,
			jm:     joints[i].JM,
			ibm:    iibm,
			parent: pnt,
		})
	}

	// Use an auxiliar stack to prevent deep,
	// reverse-sorted hierarchies from
	// degenerating the algorithm.
	var stk []int
	wgts := make([]struct{ wgt, idx int }, len(js))
	for i := range js {
		wgt := 1
		pnt := js[i].parent
		for pnt >= 0 {
			if wgts[pnt].wgt != 0 {
				wgt += wgts[pnt].wgt
				break
			}
			stk = append(stk, pnt)
			wgt++
			pnt = js[pnt].parent
		}
		wgts[i] = struct{ wgt, idx int }{wgt, i}
		for j := range stk {
			wgt--
			wgts[stk[j]] = struct{ wgt, idx int }{wgt, stk[j]}
		}
		stk = stk[:0]
	}
	sort.Slice(wgts, func(i, j int) bool { return wgts[i].wgt < wgts[j].wgt })
	hier := make([]int, len(js))
	for i := range wgts {
		hier[i] = wgts[i].idx
	}

	return &Skin{js, ibm, hier}, nil
}
