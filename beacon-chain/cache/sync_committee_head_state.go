package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1native "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v1"
	v2native "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v2"
	v3native "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v3"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
)

// SyncCommitteeHeadStateCache for the latest head state requested by a sync committee participant.
type SyncCommitteeHeadStateCache struct {
	useNativeState bool
	cache          *lru.Cache
	lock           sync.RWMutex
}

// NewSyncCommitteeHeadState initializes a LRU cache for `SyncCommitteeHeadState` with size of 1.
func NewSyncCommitteeHeadState(useNativeState bool) *SyncCommitteeHeadStateCache {
	c := lruwrpr.New(1) // only need size of 1 to avoid redundant state copies, hashing, and slot processing.
	return &SyncCommitteeHeadStateCache{cache: c, useNativeState: useNativeState}
}

// Put `slot` as key and `state` as value onto the cache.
func (c *SyncCommitteeHeadStateCache) Put(slot types.Slot, st state.BeaconState) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// Make sure that the provided state is non nil
	// and is of the correct type.
	if st == nil || st.IsNil() {
		return ErrNilValueProvided
	}

	if c.useNativeState {
		_, ok := st.(*v1native.BeaconState)
		if ok {
			return ErrIncorrectType
		}
	} else {
		_, ok := st.(*v1.BeaconState)
		if ok {
			return ErrIncorrectType
		}
	}

	c.cache.Add(slot, st)
	return nil
}

// Get `state` using `slot` as key. Return nil if nothing is found.
func (c *SyncCommitteeHeadStateCache) Get(slot types.Slot) (state.BeaconState, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, exists := c.cache.Get(slot)
	if !exists {
		return nil, ErrNotFound
	}
	var st state.BeaconState
	if c.useNativeState {
		var ok bool
		st, ok = val.(*v2native.BeaconState)
		if !ok {
			st, ok = val.(*v3native.BeaconState)
			if !ok {
				return nil, ErrIncorrectType
			}
		}
	} else {
		var ok bool
		st, ok = val.(*v2.BeaconState)
		if !ok {
			st, ok = val.(*v3.BeaconState)
			if !ok {
				return nil, ErrIncorrectType
			}
		}
	}

	return st, nil
}
