// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package gltf

import (
	"errors"
	"math"
	"strconv"
)

func newErr(reason string) error {
	return errors.New("gltf: " + reason)
}

// Check checks that f is a valid glTF object.
// TODO
func (f *GLTF) Check() error {
	vers, err := strconv.ParseFloat(f.Asset.Version, 64)
	if err != nil {
		return newErr("invalid GLTF.Asset.Version string")
	}
	minVers, err := strconv.ParseFloat(f.Asset.MinVersion, 64)
	if err == nil && minVers >= 3 {
		return newErr("unsupported GLTF.Asset.MinVersion")
	} else if vers < 2 || vers >= 3 {
		return newErr("unsupported GLTF.Asset.Version")
	}

	if s := f.Scene; s != nil && (*s < 0 || *s >= int64(len(f.Scenes))) {
		return newErr("invalid GLTF.Scene index")
	}

	for i := range f.Accessors {
		if err := f.Accessors[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Animations {
		if err := f.Animations[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Buffers {
		if err := f.Buffers[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.BufferViews {
		if err := f.BufferViews[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Cameras {
		if err := f.Cameras[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Images {
		if err := f.Images[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Materials {
		if err := f.Materials[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Meshes {
		if err := f.Meshes[i].Check(f); err != nil {
			return err
		}
	}
	// TODO: Check that the graph has no cycles and that
	// nodes have one parent at most.
	for i := range f.Nodes {
		if err := f.Nodes[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Samplers {
		if err := f.Samplers[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Scenes {
		if err := f.Scenes[i].Check(f); err != nil {
			return err
		}
	}
	for i := range f.Skins {
		if err := f.Skins[i].Check(f); err != nil {
			return err
		}
	}
	return nil
}

// Check checks that a is a valid glTF.accessors element.
func (a *Accessor) Check(gltf *GLTF) error {
	if a.BufferView != nil {
		idx := *a.BufferView
		if idx < 0 || idx > int64(len(gltf.BufferViews)) {
			return newErr("invalid Accessor.BufferView index")
		}
	}
	if a.ByteOffset < 0 { // TODO: Check upper bound.
		return newErr("invalid Accessor.BufferOffset value")
	}
	switch a.ComponentType {
	case BYTE, UNSIGNED_BYTE, SHORT, UNSIGNED_SHORT, UNSIGNED_INT, FLOAT:
	default:
		return newErr("invalid Accessor.ComponentType value")
	}
	if a.Count < 1 {
		return newErr("invalid Accessor.Count value")
	}
	switch a.Type {
	case SCALAR, VEC2, VEC3, VEC4, MAT2, MAT3, MAT4:
	default:
		return newErr("invalid Accessor.Type value")
	}
	// TODO: Check Accessor.Max/Min.

	if s := a.Sparse; s != nil {
		if s.Count < 1 || s.Count > a.Count {
			return newErr("invalid Accessor.Sparse.Count value")
		}

		if s.Indices.BufferView < 0 || s.Indices.BufferView > int64(len(gltf.BufferViews)) {
			return newErr("invalid Accessor.Sparse.Indices.BufferView index")
		}
		if s.Indices.ByteOffset < 0 { // TODO: Check upper bound.
			return newErr("invalid Accessor.Sparse.Indices.ByteOffset value")
		}
		switch s.Indices.ComponentType {
		case UNSIGNED_BYTE, UNSIGNED_SHORT, UNSIGNED_INT:
		default:
			return newErr("invalid Accessor.Sparse.Indices.ComponentType value")
		}

		if s.Values.BufferView < 0 || s.Values.BufferView > int64(len(gltf.BufferViews)) {
			return newErr("invalid Accessor.Sparse.Values.BufferView index")
		}
		if s.Values.ByteOffset < 0 { // TODO: Check upper bound.
			return newErr("invalid Accessor.Sparse.Values.ByteOffset value")
		}
	}
	return nil
}

// Check checks that a is a valid glTF.animations element.
func (a *Animation) Check(gltf *GLTF) error {
	if len(a.Channels) == 0 {
		return newErr("invalid Animation.Channels length")
	}
	if len(a.Samplers) == 0 {
		return newErr("invalid Animation.Samplers length")
	}

	for i := range a.Channels {
		c := &a.Channels[i]
		if c.Sampler < 0 || c.Sampler >= int64(len(a.Samplers)) {
			return newErr("invalid Animation.Channels[].Sampler index")
		}
		if c.Target.Node != nil {
			nd := *c.Target.Node
			if nd < 0 || nd >= int64(len(gltf.Nodes)) {
				return newErr("invalid Animation.Channels[].Target.Node index")
			}
		}
		switch c.Target.Path {
		case Ptranslation, Protation, Pscale, Pweights:
		default:
			return newErr("invalid Animation.Channels[].Target.Path value")
		}
	}

	for i := range a.Samplers {
		s := &a.Samplers[i]
		if s.Input < 0 || s.Input >= int64(len(gltf.Accessors)) {
			return newErr("invalid Animation.Samplers[].Input index")
		}
		switch s.Interpolation {
		case ILINEAR, STEP, CUBICSPLINE:
		default:
			return newErr("invalid Animation.Samplers[].Interpolation value")
		}
		if s.Output < 0 || s.Output >= int64(len(gltf.Accessors)) {
			return newErr("invalid Animation.Samplers[].Output index")
		}
	}
	return nil
}

// Check checks that b is a valid glTF.buffers element.
func (b *Buffer) Check(gltf *GLTF) error {
	if b.ByteLength < 1 {
		return newErr("invalid Buffer.ByteLength value")
	}
	return nil
}

// Check checks that v is a valid glTF.bufferViews element.
func (v *BufferView) Check(gltf *GLTF) error {
	if v.Buffer < 0 || v.Buffer >= int64(len(gltf.BufferViews)) {
		return newErr("invalid BufferView.Buffer index")
	}
	if v.ByteOffset < 0 {
		return newErr("invalid BufferView.ByteOffset value")
	}
	if v.ByteLength < 1 || v.ByteOffset+v.ByteLength > gltf.Buffers[v.Buffer].ByteLength {
		return newErr("invalid BufferView.ByteLength value")
	}
	if v.ByteStride != 0 && (v.ByteStride < 4 || v.ByteStride > 252) {
		return newErr("invalid BufferView.ByteStride value")
	}
	switch v.Target {
	case 0, ARRAY_BUFFER, ELEMENT_ARRAY_BUFFER:
	default:
		return newErr("invalid BufferView.Target value")
	}
	return nil
}

// Check checks that c is a valid glTF.cameras element.
func (c *Camera) Check(gltf *GLTF) error {
	switch c.Type {
	case Torthographic:
		if c.Orthographic == nil || c.Perspective != nil {
			return newErr("invalid Camera.Orthographic setup")
		}
		if c.Orthographic.Xmag <= 0 {
			return newErr("invalid Camera.Orthographic.Zmag value")
		}
		if c.Orthographic.Ymag <= 0 {
			return newErr("invalid Camera.Orthographic.Ymag value")
		}
		if c.Orthographic.Zfar == 0 || c.Orthographic.Zfar <= c.Orthographic.Znear {
			return newErr("invalid Camera.Orthographic.Zfar value")
		}
	case Tperspective:
		if c.Perspective == nil || c.Orthographic != nil {
			return newErr("invalid Camera.Perspective setup")
		}
		if c.Perspective.AspectRatio <= 0 {
			return newErr("invalid Camera.Perspective.AspectRatio value")
		}
		if c.Perspective.YFOV >= math.Pi {
			return newErr("invalid Camera.Perspective.YFOV value")
		}
		if c.Perspective.Zfar != 0 && c.Perspective.Zfar <= c.Perspective.Znear {
			return newErr("invalid Camera.Perspective.Zfar value")
		}
		if c.Perspective.Znear <= 0 {
			return newErr("invalid Camera.Perspective.Znear value")
		}
	default:
		return newErr("invalid Camera.Type value")
	}
	return nil
}

// Check checks that i is a valid glTF.images element.
func (i *Image) Check(gltf *GLTF) error {
	switch i.URI {
	case "":
		if i.BufferView == nil {
			return newErr("invalid Image.URI/BufferView non-definitions")
		}
		if *i.BufferView < 0 || *i.BufferView >= int64(len(gltf.BufferViews)) {
			return newErr("invalid Image.BufferView index")
		}
		switch i.MimeType {
		case JPEG, PNG:
		default:
			return newErr("invalid Image.MimeType value")
		}
	default:
		if i.BufferView != nil {
			return newErr("invalid Image.URI/BufferView definitions")
		}
	}
	return nil
}

// Check checks that m is a valid glTF.materials element.
func (m *Material) Check(gltf *GLTF) error {
	if pbr := m.PBRMetallicRoughness; pbr != nil {
		if fac := pbr.BaseColorFactor; fac != nil {
			for _, x := range fac {
				if x < 0 || x > 1 {
					return newErr("invalid Material.PBRMetallicRoughness.BaseColorFactor value")
				}
			}
		}
		if tex := pbr.BaseColorTexture; tex != nil {
			if tex.Index < 0 || tex.Index >= int64(len(gltf.Textures)) {
				return newErr("invalid Material.PBRMetallicRoughness.BaseColorTexture.Index index")
			}
			if tex.TexCoord < 0 {
				return newErr("invalid Material.Texture.PBRMetallicRoughness.BaseColorTexture.TexCoord set")
			}
			// XXX: Only two sets are supported currently.
			if tex.TexCoord > 1 {
				return newErr("unsupported Material.PBRMetallicRoughness.BaseColorTexture.TexCoord set")
			}
		}
		if fac := pbr.MetallicFactor; fac != nil {
			if *fac < 0 || *fac > 1 {
				return newErr("invalid Material.PBRMetallicRoughness.MetallicFactor value")
			}
		}
		if fac := pbr.RoughnessFactor; fac != nil {
			if *fac < 0 || *fac > 1 {
				return newErr("invalid Material.PBRMetallicRoughness.RoughnessFactor value")
			}
		}
		if tex := pbr.MetallicRoughnessTexture; tex != nil {
			if tex.Index < 0 || tex.Index >= int64(len(gltf.Textures)) {
				return newErr("invalid Material.PBRMetallicRoughness.MetallicRoughnessTexture.Index index")
			}
			if tex.TexCoord < 0 {
				return newErr("invalid Material.Texture.PBRMetallicRoughness.MetallicRoughnessTexture.TexCoord set")
			}
			// XXX: Only two sets are supported currently.
			if tex.TexCoord > 1 {
				return newErr("unsupported Material.PBRMetallicRoughness.MetallicRoughnessTexture.TexCoord set")
			}
		}
	}

	if norm := m.NormalTexture; norm != nil {
		if norm.Index < 0 || norm.Index >= int64(len(gltf.Textures)) {
			return newErr("invalid Material.NormalTexture.Index index")
		}
		if norm.TexCoord < 0 {
			return newErr("invalid Material.NormalTexture.TexCoord set")
		}
		// XXX: Only two sets are supported currently.
		if norm.TexCoord > 1 {
			return newErr("unsupported Material.NormalTexture.TexCoord set")
		}
	}

	if occ := m.OcclusionTexture; occ != nil {
		if occ.Index < 0 || occ.Index >= int64(len(gltf.Textures)) {
			return newErr("invalid Material.OcclusionTexture.Index index")
		}
		if occ.TexCoord < 0 {
			return newErr("invalid Material.OcclusionTexture.TexCoord set")
		}
		// XXX: Only two sets are supported currently.
		if occ.TexCoord > 1 {
			return newErr("unsupported Material.OcclusionTexture.TexCoord set")
		}
		if str := occ.Strength; str != nil {
			if *str < 0 || *str > 1 {
				return newErr("invalid Material.OcclusionTexture.Strength value")
			}
		}
	}

	if emis := m.EmissiveTexture; emis != nil {
		if emis.Index < 0 || emis.Index >= int64(len(gltf.Textures)) {
			return newErr("invalid Material.EmissiveTexture.Index index")
		}
		if emis.TexCoord < 0 {
			return newErr("invalid Material.EmissiveTexture.TexCoord set")
		}
		// XXX: Only two sets are supported currently.
		if emis.TexCoord > 1 {
			return newErr("unsupported Material.EmissiveTexture.TexCoord set")
		}
	}
	if fac := m.EmissiveFactor; fac != nil {
		for _, x := range fac {
			if x < 0 || x > 1 {
				return newErr("invalid Material.EmissiveFactor value")
			}
		}
	}

	switch m.AlphaMode {
	case "", OPAQUE, MASK, BLEND:
	default:
		return newErr("invalid Material.AlphaMode value")
	}
	if cut := m.AlphaCutoff; cut != nil {
		if m.AlphaMode == "" {
			return newErr("invalid Material.AlphaCutoff definition")
		}
		if *cut < 0 {
			return newErr("invalid Material.AlphaCutoff value")
		}
	}
	return nil
}

// Check checks that m is a valid glTF.meshes element.
func (m *Mesh) Check(gltf *GLTF) error {
	if len(m.Primitives) == 0 {
		return newErr("invalid Mesh.Primitives length")
	}
	for i := range m.Primitives {
		p := &m.Primitives[i]
		if _, ok := p.Attributes["POSITION"]; !ok {
			return newErr("invalid Mesh.Primitives[].Attributes map")
		}
		// TODO: Should check for unsupported semantics.
		for _, v := range p.Attributes {
			if v < 0 || v >= int64(len(gltf.Accessors)) {
				return newErr("invalid Mesh.Primitives[].Attributes index")
			}
		}
		if i := p.Indices; i != nil {
			if *i < 0 || *i >= int64(len(gltf.Accessors)) {
				return newErr("invalid Mesh.Primitives[].Indices index")
			}
		}
		if i := p.Material; i != nil {
			if *i < 0 || *i >= int64(len(gltf.Accessors)) {
				return newErr("invalid Mesh.Primitives[].Material index")
			}
		}
		if p.Mode != nil {
			switch *p.Mode {
			case POINTS, LINES, LINE_LOOP, LINE_STRIP, TRIANGLES, TRIANGLE_STRIP, TRIANGLE_FAN:
			default:
				return newErr("invalid Mesh.Primitives[].Mode value")
			}
		}
	}
	return nil
}

// Check checks that n is a valid glTF.nodes element.
func (n *Node) Check(gltf *GLTF) error {
	if cam := n.Camera; cam != nil {
		if *cam < 0 || *cam >= int64(len(gltf.Cameras)) {
			return newErr("invalid Node.Camera index")
		}
	}
	if sk := n.Skin; sk != nil {
		if *sk < 0 || *sk >= int64(len(gltf.Skins)) {
			return newErr("invalid Node.Skin index")
		}
	}
	if m := n.Matrix; m != nil {
		if n.Rotation != nil || n.Scale != nil || n.Translation != nil {
			return newErr("invalid Node.Matrix/TRS definitions")
		}
	}
	if msh := n.Mesh; msh != nil {
		if *msh < 0 || *msh >= int64(len(gltf.Meshes)) {
			return newErr("invalid Node.Mesh index")
		}
	}
	clen := len(n.Children)
	switch clen {
	case 0:
	case 1:
		if n.Children[0] < 0 || n.Children[0] >= int64(len(gltf.Nodes)) {
			return newErr("invalid Node.Children[] index")
		}
		// TODO: Checking for cycles on gltf.Check should
		// handle this case.
		if &gltf.Nodes[n.Children[0]] == n {
			return newErr("invalid Node.Children[] hierarchy")
		}
	default:
		cmap := make(map[int64]bool, clen)
		for _, chd := range n.Children {
			if chd < 0 || chd >= int64(len(gltf.Nodes)) {
				return newErr("invalid Node.Children[] index")
			}
			// TODO: See above.
			if &gltf.Nodes[chd] == n {
				return newErr("invalid Node.Children[] hierarchy")
			}
			cmap[chd] = true
		}
		if clen != len(cmap) {
			return newErr("invalid Node.Children list")
		}
	}
	return nil
}

// Check checks that s is a valid glTF.samplers element.
func (s *Sampler) Check(gltf *GLTF) error {
	switch s.MagFilter {
	case 0, NEAREST, FLINEAR:
	default:
		return newErr("invalid Sampler.MagFilter value")
	}
	switch s.MinFilter {
	case 0, NEAREST, FLINEAR, NEAREST_MIPMAP_NEAREST, LINEAR_MIPMAP_NEAREST, NEAREST_MIPMAP_LINEAR, LINEAR_MIPMAP_LINEAR:
	default:
		return newErr("invalid Sampler.MinFilter value")
	}
	for _, w := range [2]int64{s.WrapS, s.WrapT} {
		switch w {
		case 0, CLAMP_TO_EDGE, MIRRORED_REPEAT, REPEAT:
		default:
			return newErr("invalid Sampler.WrapS/T value")
		}
	}
	return nil
}

// Check checks that s is a valid glTF.scenes element.
func (s *Scene) Check(gltf *GLTF) error {
	nlen := len(s.Nodes)
	switch nlen {
	case 0:
	case 1:
		if s.Nodes[0] < 0 || s.Nodes[0] >= int64(len(gltf.Nodes)) {
			return newErr("invalid Scene.Nodes[] index")
		}
	default:
		nmap := make(map[int64]bool, nlen)
		for _, nd := range s.Nodes {
			if nd < 0 || nd >= int64(len(gltf.Nodes)) {
				return newErr("invalid Scene.Nodes[] index")
			}
			nmap[nd] = true
		}
		if nlen != len(nmap) {
			return newErr("invalid Scene.Nodes list")
		}
	}
	return nil
}

// Check checks that s is a valid glTF.skins element.
func (s *Skin) Check(gltf *GLTF) error {
	if ibm := s.InverseBindMatrices; ibm != nil {
		if *ibm < 0 || *ibm >= int64(len(gltf.Accessors)) {
			return newErr("invalid Skin.InverseBindMatrices index")
		}
		acc := &gltf.Accessors[*ibm]
		if acc.Count < int64(len(s.Joints)) || acc.Type != MAT4 {
			return newErr("invalid Skin.InverseBindMatrices accessor")
		}
	}
	if skl := s.Skeleton; skl != nil {
		if *skl < 0 || *skl >= int64(len(gltf.Nodes)) {
			return newErr("invalid Skin.Skeleton index")
		}
	}
	jlen := len(s.Joints)
	switch jlen {
	case 0:
		return newErr("invalid Skin.Joints length")
	case 1:
		if s.Joints[0] < 0 || s.Joints[0] >= int64(len(gltf.Nodes)) {
			return newErr("invalid Skin.Joints[] index")
		}
	default:
		jmap := make(map[int64]bool, jlen)
		for _, jnt := range s.Joints {
			if jnt < 0 || jnt >= int64(len(gltf.Nodes)) {
				return newErr("invalid Skin.Joints[] index")
			}
			jmap[jnt] = true
		}
		if jlen != len(jmap) {
			return newErr("invalid Skin.Joints list")
		}
	}
	return nil
}
