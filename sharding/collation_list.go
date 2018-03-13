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

// collationNumberHeap is a heap.Interface implementation over 64bit unsigned integers for
// retrieving sorted collations from the possibly gapped future queue.
type collationNumberHeap []uint64

func (h collationNumberHeap) Len() int           { return len(h) }
func (h collationNumberHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h collationNumberHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *collationNumberHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *collationNumberHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

//collationsSortedMap is a collationNumber->collation hash map with a heap based index to allow
//iterating over the contents in a collationNumber-incrementing way.

type collationsSortedMap struct {
	items map[uint64]*Collation // Hash map storing the collation data
	index *collationNumberHeap  // Heap of collationNumbers of all the stored collations under non-strict mode
	cache Collations            // Cache of the collations already sorted
}

// newCollationSortedMap creates a new number sorted collation map.
func newCollationSortedMap() *collationsSortedMap {
	return &collationsSortedMap{
		items: make(map[uint64]*Collation),
		index: new(collationNumberHeap),
	}
}

// Get retrieves the current collation associated with the given collationNumber.
func (m *collationsSortedMap) Get(collationNumber uint64) *Collation {
	return m.items[collationNumber]
}

// Put inserts a new collation into the map, also updating the map's collationNumber index
// If a collation is already exists with the same collationNumber, it's over written.
func (m *collationsSortedMap) Put(collation *Collation) {
	collationNumber := collation.header.collationNumber
	if m.items[collationNumber] == nil {
		heap.Push(m.index, collationNumber)
	}
	m.items[collationNumber], m.cache = collation, nil
}

// Forward removes all the collations from the map with a collationNumber lower than the
// provided threshold. Every removed collation is returns for any post-removal
// maintenance.
func (m *collationsSortedMap) Forward(threshold uint64) Collations {
	var removed Collations

	//Pop off head items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		collationNumber := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[collationNumber])
		delete(m.items, collationNumber)
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
	for collationNumber, collation := range m.items {
		if filter(collation) {
			removed = append(removed, collation)
			delete(m.items, collationNumber)
		}
	}
	// If collations were removed, the heap and cache are ruined
	// Fix heap and cache.
	if len(removed) > 0 {
		*m.index = make([]uint64, 0, len(m.items))
		for collationNumber := range m.items {
			*m.index = append(*m.index, collationNumber)
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
	// Otherwise gather and drop the highest collationNumber'd collations
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
func (m *collationsSortedMap) Remove(collationNumber uint64) bool {
	// Short circuit if no collation is present
	_, ok := m.items[collationNumber]
	if !ok {
		return false
	}
	// Otherwise delete the collation and fix the heap index
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == collationNumber {
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, collationNumber)
	m.cache = nil

	return true
}

// Ready retrieves a sequentially increasing list of collations starting at the
// provided collationNumber that is ready for processing. The returned collations will be
// removed from the list.
//
// Note, all collations with collationNumber lower than start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (m *collationsSortedMap) Ready(start uint64) Collations {
	// Short circuit if no collations are available
	if m.index.Len() == 0 || (*m.index)[0] > start {
		return nil
	}
	// Otherwise start accumulating incremental collations
	var ready Collations
	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
		ready = append(ready, m.items[next])
		delete(m.items, next)
		heap.Pop(m.index)
	}
	m.cache = nil

	return ready
}

// Len returns the length of the collation map
func (m *collationsSortedMap) Len() int {
	return len(m.items)
}

// Flatten creates a collationNumber-sorted slice of collations based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (m *collationsSortedMap) Flatten() Collations {
	// If the sorting was not cached yet, create and cache it
	if m.cache == nil {
		m.cache = make(Collations, 0, len(m.items))
		for _, tx := range m.items {
			m.cache = append(m.cache, tx)
		}
		sort.Sort(CollationByNumber(m.cache))
	}
	// Copy the cache to prevent accidental modifications
	cacheCopy := make(Collations, len(m.cache))
	copy(cacheCopy, m.cache)
	return cacheCopy
}

// bidHeap is a heap.Interface implementation over collations for retrieving
// bid price sorted collations to discard when the pool fills up.
type bidPriceHeap []*Collation

func (h bidPriceHeap) Len() int           { return len(h) }
func (h bidPriceHeap) Less(i, j int) bool { return h[i].BidPrice().Cmp(h[j].BidPrice()) < 0 }
func (h bidPriceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *bidPriceHeap) Push(x interface{}) {
	*h = append(*h, x.(*Collation))
}

func (h *bidPriceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// collationBidList is a bid-sorted heap to allow operating on proposer pool
// contents in a bid-incrementing way.
type collationBidList struct {
	all    *map[uint64]*Collation // Pointer to the map of all collations
	items  *bidPriceHeap          // Heap of bids of all the stored collations
	//TODO: DO WE NEED TO TRACK STALES BID POINTS?
	stales int                    // Number of stale bid points to (re-heap trigger)
}

// newCollationBidList creates a new bid-sorted collation heap.
func newCollationBidList(all *map[uint64]*Collation) *collationBidList {
	return &collationBidList{
		all:   all,
		items: new(bidPriceHeap),
	}
}

// Put inserts a new collation into the heap
func (l *collationBidList) Put(collation *Collation) {
	heap.Push(l.items, collation)
}

// Removed notifies the bid price collation list that an old collation dropped
// from the pool. The list will just keep a counter of stale objects and update
// the heap if a large enough ratio of transactions go stale.
func (l *collationBidList) Removed() {
	// Bump the stale counter, but exit if still too low (< 25%)
	l.stales++
	if l.stales <= len(*l.items)/4 {
		return
	}
	// Seems we've reached a critical number of stale transactions, reheap
	reheap := make(bidPriceHeap, 0, len(*l.all))

	l.stales, l.items = 0, &reheap
	for _, collation := range *l.all {
		*l.items = append(*l.items, collation)
	}
	heap.Init(l.items)
}

// Cap finds all the bids below the given price threshold, drops them
// from the priced list and returns them for further removal from the entire pool.
func (l *collationBidList) Cap(threshold *big.Int) Collations {
	drop := make(Collations, 0, 128) // Remote underpriced collations to drop
	save := make(Collations, 0, 64)  // Local underpriced collations to keep

	for len(*l.items) > 0 {
		// Discard stale collations if found during cleanup
		collation := heap.Pop(l.items).(*Collation)
		if _, ok := (*l.all)[collation.Number()]; !ok {
			l.stales--
			continue
		}
		// Stop the discards if we've reached the threshold
		if collation.BidPrice().Cmp(threshold) >= 0 {
			save = append(save, collation)
			break
		} else {
			drop = append(drop, collation)
		}
	}
	for _, collation := range save {
		heap.Push(l.items, collation)
	}
	return drop
}

// Underpriced checks whether a collation's bid is cheaper than (or as cheap as) the
// lowest priced collation currently being tracked.
func (l *collationBidList) Underpriced(collation Collation) bool {
	// Check if the transaction is underpriced or not
	if len(*l.items) == 0 {
		log.Error("Pricing query for empty pool") // This cannot happen, print to catch programming errors
		return false
	}
	cheapest := []*Collation(*l.items)[0]
	return cheapest.BidPrice().Cmp(collation.BidPrice()) >= 0
}

// Discard finds a number of most underpriced collations, removes them from the
// bid list and returns them for further removal from the entire pool.
func (l collationBidList) Discard(count int) Collations {
	drop := make(Collations, 0, count) // Remote underpriced collations to drop

	for len(*l.items) > 0 && count > 0 {
		// Discard stale collations if found during cleanup
		collation := heap.Pop(l.items).(*Collation)
		if _, ok := (*l.all)[collation.Number()]; !ok {
			l.stales--
			continue
		}
		// Non stale transaction found, discard
		drop = append(drop, collation)
		count--
	}
	return drop
}
