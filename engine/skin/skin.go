// Copyright 2023 Gustavo C. Viegas. All rights reserved.

// Package skin implements blend-weight skinning.
package skin

import (
	"errors"
	"sort"

	"github.com/gviegas/scene/linear"
)

const prefix = "skin: "

// Skin defines skinning data.
type Skin struct {
	// Sorted such that every parent comes
	// before any of its descendants.
	// The original ordering of the joints
	// can be inferred from the orig field.
	joints []joint
	// Only store inverse bind matrices that
	// are not the zero/identity matrix.
	ibm []linear.M4

	// TODO: Descriptors; const buffer
	// (per-instance, most likely).
}

// joint defines a skin's joint.
type joint struct {
	name string
	jm   linear.M4
	ibm  int
	// The original index of the joint's
	// parent (unchanged from Joint's).
	parent int
	// The original index of the joint,
	// i.e., what the mesh refers in
	// its Joints* semantic(s).
	// This is necessary because Skin
	// sorts the joints by parent.
	orig int
}

// jointSlice implements sort.Interface for joint slices.
type jointSlice []joint

func (c jointSlice) Len() int           { return len(c) }
func (c jointSlice) Less(i, j int) bool { return c[i].parent < c[j].parent }
func (c jointSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

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

// New creates a new skin from a joint hierarchy.
func New(joints []Joint) (*Skin, error) {
	n := len(joints)
	if n == 0 {
		return nil, errors.New(prefix + "[]Joint length is 0")
	}

	js := make(jointSlice, 0, n)
	var ibm []linear.M4
	var zero, ident linear.M4
	ident.I()

	for i := range joints {
		pnt := joints[i].Parent
		switch {
		case pnt >= n:
			return nil, errors.New(prefix + "Joint.Parent out of bounds")
		case pnt == i:
			return nil, errors.New(prefix + "Joint.Parent refers to itself")
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
			orig:   i,
		})
	}

	sort.Sort(js)
	return &Skin{js, ibm}, nil
}
