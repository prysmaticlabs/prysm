package cache

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

const (
	// maxSyncCommitteeHeadStateCacheSize only need size of 1 to avoid redundant state copies,
	// hashing, and slot processing.
	maxSyncCommitteeHeadStateCacheSize = int(1)
)

var (
	// BalanceCacheMiss tracks the number of balance requests that aren't present in the cache.
	maxSyncCommitteeHeadStateCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "max_sync_committee_head_state_cache_miss",
		Help: "The number of get requests that aren't present in the cache.",
	})
	// BalanceCacheHit tracks the number of balance requests that are in the cache.
	maxSyncCommitteeHeadStateCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "max_sync_committee_head_state_cache_hit",
		Help: "The number of get requests that are present in the cache.",
	})
)

// SyncCommitteeHeadStateCache for the latest head state requested by a sync committee participant.
type SyncCommitteeHeadStateCache[K primitives.Slot, V state.BeaconState] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter
}

// NewSyncCommitteeHeadStateCache creates a new sync committee head state cache
func NewSyncCommitteeHeadStateCache[K primitives.Slot, V state.BeaconState]() (*SyncCommitteeHeadStateCache[K, V], error) {
	cache, err := lru.New[K, V](maxSyncCommitteeHeadStateCacheSize)
	if err != nil {
		return nil, ErrCacheCannotBeNil
	}

	if maxSyncCommitteeHeadStateCacheMiss == nil || maxSyncCommitteeHeadStateCacheHit == nil {
		return nil, ErrCacheMetricsCannotBeNil
	}

	return &SyncCommitteeHeadStateCache[K, V]{
		lru:           cache,
		promCacheMiss: maxSyncCommitteeHeadStateCacheMiss,
		promCacheHit:  maxSyncCommitteeHeadStateCacheHit,
	}, nil
}

func (c *SyncCommitteeHeadStateCache[K, V]) get() *lru.Cache[K, V] {
	return c.lru
}

func (c *SyncCommitteeHeadStateCache[K, V]) hitCache() {
	c.promCacheHit.Inc()
}

func (c *SyncCommitteeHeadStateCache[K, V]) missCache() {
	c.promCacheMiss.Inc()
}

// Put `slot` as key and `state` as value onto the cache.
func (c *SyncCommitteeHeadStateCache[K, V]) Put(slot K, st V) error {
	// Make sure that the provided state is non nil
	// and is of the correct type.
	if isNil(st) || st.IsNil() {
		return ErrNilValueProvided
	}

	if st.Version() == version.Phase0 {
		return ErrIncorrectType
	}

	return Add[K, V](c, slot, st)
}

// Get `state` using `slot` as key. Return nil if nothing is found.
func (c *SyncCommitteeHeadStateCache[K, V]) Get(slot K) (V, error) {
	var (
		noState V
	)

	state, err := Get[K, V](c, slot)
	if err != nil {
		return noState, err
	}

	// Sync committee is not supported in phase 0.
	if state.Version() == version.Phase0 {
		return noState, ErrIncorrectType
	}

	return state, nil
}

func (c *SyncCommitteeHeadStateCache[K, V]) Clear() {
	Purge[K, V](c)
}
