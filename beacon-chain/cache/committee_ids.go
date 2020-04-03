package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type committeeIDs struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// CommitteeIDs for attestations.
var CommitteeIDs = newCommitteeIDs()

func newCommitteeIDs() *committeeIDs {
	maxCommitteesPerEpoch := int(params.BeaconConfig().MaxCommitteesPerSlot * params.BeaconConfig().SlotsPerEpoch)
	cache, err := lru.New(maxCommitteesPerEpoch)
	if err != nil {
		panic(err)
	}
	return &committeeIDs{cache: cache}
}

// AddID adds committee ID for subscribing subnet for the attester and/or aggregator of a given slot.
func (t *committeeIDs) AddID(committeeID uint64, slot uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	committeeIDs := []uint64{committeeID}
	val, exists := t.cache.Get(slot)
	if exists {
		committeeIDs = sliceutil.UnionUint64(append(val.([]uint64), committeeIDs...))
	}
	t.cache.Add(slot, committeeIDs)
}

// GetIDs gets the committee ID for subscribing subnet for attester and/or aggregator of the slot.
func (t *committeeIDs) GetIDs(slot uint64) []uint64 {
	val, exists := t.cache.Get(slot)
	if !exists {
		return []uint64{}
	}
	return val.([]uint64)
}
