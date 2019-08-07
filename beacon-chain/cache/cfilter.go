package cache

import (
	"sync"

	"github.com/irfansharif/cfilter"
)

// ActiveIndicesCFilter defines cuckoo Filter for storing and lookup of active indices
// Cuckoo filter is a Bloom filter replacement for approximated set-membership queries. Cuckoo filters support adding and removing items dynamically while achieving even higher performance than Bloom filters
type ActiveIndicesCFilter struct {
	filter *cfilter.CFilter
	lock   sync.RWMutex
}

// NewCFilter a new cuckoo filter for storing and lookup active indices
func NewCFilter() *ActiveIndicesCFilter {
	return &ActiveIndicesCFilter{filter: cfilter.New()}
}

// InsertActiveIndicesCFilter adds a byte representation of active indices to a cuckoo filter
func (cf *ActiveIndicesCFilter) InsertActiveIndicesCFilter(byteIndices [][]byte) {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	for _, i := range byteIndices {
		cf.filter.Insert(i)
	}
}

// LookupActiveIndicesCFilter makes a membership check if active indices are in cuckoo filter
func (cf *ActiveIndicesCFilter) LookupActiveIndicesCFilter(byteIndices [][]byte) bool {
	cf.lock.Lock()
	defer cf.lock.Unlock()
	for _, i := range byteIndices {
		if !cf.filter.Lookup(i) {
			return false
		}
	}
	return true
}
