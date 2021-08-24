package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
)

// SyncCommitteeHeadStateCache for the latest head state requested by a sync committee participant.
type SyncCommitteeHeadStateCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewSyncCommitteeHeadState initializes a LRU cache for `SyncCommitteeHeadState` with size of 1.
func NewSyncCommitteeHeadState() (*SyncCommitteeHeadStateCache, error) {
	c, err := lru.New(1) // only need size of 1 to avoid redundant state copies, hashing, and slot processing.
	if err != nil {
		return nil, err
	}
	return &SyncCommitteeHeadStateCache{cache: c}, nil
}

// Put `slot` as key and `state` as value onto the cache.
func (c *SyncCommitteeHeadStateCache) Put(slot types.Slot, st state.BeaconState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(slot, st)
}

// Get `state` using `slot` as key. Return nil if nothing is found.
func (c *SyncCommitteeHeadStateCache) Get(slot types.Slot) (state.BeaconState, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, exists := c.cache.Get(slot)
	if !exists || val == nil {
		return nil, ErrNotFound
	}
	return val.(*stateAltair.BeaconState), nil
}
