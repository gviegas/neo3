// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package scene

import (
	"testing"

	"gviegas/neo3/node"
)

func TestNew(t *testing.T) {
	var z Scene
	s := New()
	if s.graph.Len() != z.graph.Len() {
		t.Fatal("New().graph.Len: New should not insert any nodes")
	}
	if *s.graph.World(node.Nil) != *z.graph.World(node.Nil) {
		t.Fatal("New().graph.World: New should not set the global world transform")
	}
}
