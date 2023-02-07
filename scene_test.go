// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package scene

import (
	"testing"

	"github.com/gviegas/scene/node"
)

func TestNew(t *testing.T) {
	var z Scene
	s := New()
	if s.Len() != z.Len() {
		t.Fatal("New().Graph.Len: New should not insert any nodes")
	}
	if *s.World(node.Nil) != *z.World(node.Nil) {
		t.Fatal("New().Graph.World: New should not set the global world transform")
	}
}
