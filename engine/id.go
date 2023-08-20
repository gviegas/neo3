// Copyright 2023 Gustavo C. Viegas. All rights reserved.

package engine

import (
	"gviegas/neo3/internal/bitm"
)

// dataID identifies a dataMap.data element.
type dataID struct {
	data int
}

// dataEntry is what a dataMap stores.
type dataEntry[T any] struct {
	data T
	id   int
}

// dataMap stores data of type D with identifiers
// of type I.
type dataMap[I ~int, D any] struct {
	ids   []dataID
	idMap bitm.Bitm[uint32]
	data  []dataEntry[D]
}
