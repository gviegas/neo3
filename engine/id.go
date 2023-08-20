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

// insert inserts data into m.
// It returns an I value that identifies data in m.
func (m *dataMap[I, D]) insert(data D) I {
	if m.idMap.Rem() == 0 {
		switch n := m.idMap.Len(); {
		case n > 0:
			cnt := 1 + (n-31)/32
			m.ids = append(m.ids, m.ids...)
			m.idMap.Grow(cnt)
		default:
			var elems [32]dataID
			m.ids = append(m.ids, elems[:]...)
			m.idMap.Grow(1)
		}
	}
	var id I
	if idx, ok := m.idMap.Search(); ok {
		m.idMap.Set(idx)
		id = I(idx)
	} else {
		// Should never happen.
		panic("unexpected failure from bitm.Bitm.Search")
	}
	m.ids[id] = dataID{data: len(m.data)}
	m.data = append(m.data, dataEntry[D]{data, int(id)})
	return id
}

// remove removes the data identified by id.
// It returns the removed data.
// id must belong to m.
func (m *dataMap[I, D]) remove(id I) D {
	d := m.ids[id].data
	data := m.data[d]
	last := len(m.data) - 1
	if d < last {
		swap := m.data[last].id
		m.ids[swap].data = d
		m.data[d] = m.data[last]
	}
	m.ids[id].data = -1
	m.idMap.Unset(int(id))
	m.data[last] = dataEntry[D]{}
	m.data = m.data[:last]
	return data.data
}

// get returns a pointer to the data identified by id.
// id must belong to m.
func (m *dataMap[I, D]) get(id I) *D { return &m.data[m.ids[id].data].data }

// entries returns the dataEntry slice of m.
// This slice aliases m's entries and as such must not
// be mutated by the caller.
func (m *dataMap[I, D]) entries() []dataEntry[D] { return m.data }

// len is equivalent to len(m.entries()).
func (m *dataMap[_, _]) len() int { return len(m.data) }
