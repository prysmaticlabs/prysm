package sharding

import (
	"container/heap"
	"math"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// nonceHeap is a heap.Interface implementation over 64bit unsigned integers for
// retrieving sorted collations from the possibly gapped future queue.
type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

//collationsSortedMap is a nonce->collation hash map with a heap based index to allow
//iterating over the contents in a nonce-incrementing way.

type collationsSortedMap struct {
	items map[uint64]*Collation // Hash map storing the collation data
	index *nonceHeap            // Heap of nonces of all the stored collations under non-strict mode
	cache []*Collation            // Cache of the collations already sorted
}

// newCollationSortedMap creates a new nonce-sorted collation map.
func newCollationSortedMap() *collationsSortedMap {
	return &collationsSortedMap{
		items: make(map[uint64]*Collation),
		index: new(nonceHeap),
	}
}

// Get retrieves the current collation associated with the given nonce.
func (m *collationsSortedMap) Get(nonce uint64) *Collation {
	return m.items[nonce]
}

// Put inserts a new collation into the map, also updating the map's nonce index
// If a collation is already exists with the same nonce, it's over written.
func (m *collationsSortedMap) Put(collation *Collation) {
	nonce := collation.Nonce()
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
	}
	m.items[nonce], m.cache = collation, nil
}

// Forward removes all the collations from the map with a nonce lower than the
// provided threshold. Every removed collation is returns for any post-removal
// maintenance.
func (m *collationsSortedMap) Forward(threshold uint64) Collation {
	var removed Collations

	//Pop off head items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		delete(m.items, nonce)
	}

	// Shift the cached order to the front
	if m.cache != nil {
		m.cache = m.cache[len(removed):]

	}
	return removed
}

// Filter iterates over the list of collations and removes all of them for which
// the specified function evaluates to true.
func (m *collationsSortedMap) Filter(filter func(*Collation) bool) Collations {
	var removed Collations

	//Collect all the collations to filter out
	for nonce, collation := range m.items {
		if filter(collation) {
			removed = append(removed, collation)
			delete(m.items, nonce)
		}
	}
	// If collations were removed, the heap and cache are ruined
	// Fix heap and cache.
	if len(removed) > 0 {
		*m.index = make([]uint64, 0, len(m.items))
		for nonce := range m.items {
			*m.index = append(*m.index, nonce)
		}
		heap.Init(m.index)

		m.cache = nil
	}
	return removed
}

// Cap places a hard limit on the number of items, returning all collations
// exceeding that limit.
func (m *collationsSortedMap) Cap(threshold int) Collations {
	// Short circuit if the number of items is under the limit
	if len(m.items) <= threshold {
		return nil
	}
	// Otherwise gather and drop the highest nonce'd collations
	var drops Collations

	sort.Sort(*m.index)
	for size := len(m.items); size > threshold; size-- {
		drops = append(drops, m.items[(*m.index)[size-1]])
		delete(m.items, (*m.index)[size-1])
	}
	*m.index = (*m.index)[:threshold]
	heap.Init(m.index)

	// If we had a cache, shift the back
	if m.cache != nil {
		m.cache = m.cache[:len(m.cache)-len(drops)]
	}
	return drops
}

// Remove deletes a collation from the maintained map, returning whether
// the collation was found.
func (m *collationsSortedMap) Remove(nonce uint64) bool {
	// Short circuit if no collation is present
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	// Otherwise delete the collation and fix the heap index
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i]==nonce {
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	m.cache = nil

	return true
}