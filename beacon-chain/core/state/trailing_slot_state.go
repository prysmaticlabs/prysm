package state

import (
	"bytes"
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

type trailingSlotCache struct {
	sync.RWMutex
	root  []byte
	state *state.BeaconState
}

var (
	tsCache trailingSlotCache
	// Metrics for the validator cache.
	trailingSlotCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trailing_slot_cache_hit",
		Help: "The total number of cache hits on the trailing slot state cache.",
	})
	trailingSlotCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trailing_slot_cache_miss",
		Help: "The total number of cache misses on the trailing slot state cache.",
	})
)

// GetTrailingSlotState returns the saved state if the input root matches the saved root in
// `trailingSlotCache`. Otherwise it returns nil.
// This is useful to call before processing slots. With a cache hit, it returns last processed state
// after already advancing slot by 1.
func GetTrailingSlotState(ctx context.Context, root []byte) (*state.BeaconState, error) {
	tsCache.Lock()
	defer tsCache.Unlock()
	if !bytes.Equal(root, tsCache.root) {
		trailingSlotCacheMiss.Inc()
		return nil, nil
	}
	trailingSlotCacheHit.Inc()
	// Returning copied state.
	return tsCache.state.Copy(), nil
}

// UpdateTrailingSlotState updates the `trailingSlotCache`. It saves the input state after processing its slot by 1,
// it also saves the input root for later look up.
// This is useful to call after successfully processing a block.
func UpdateTrailingSlotState(ctx context.Context, root []byte, state *state.BeaconState) error {
	tsCache.RLock()
	// Returning early if state is already in the cache.
	if bytes.Equal(root, tsCache.root) {
		defer tsCache.RUnlock()
		return nil
	}
	tsCache.RUnlock()

	// Advancing slot by one using a copied state.
	copied := state.Copy()
	copied, err := ProcessSlots(ctx, copied, copied.Slot()+1)
	if err != nil {
		return err
	}

	tsCache.Lock()
	defer tsCache.Unlock()

	tsCache.root = root
	tsCache.state = copied
	return nil
}
