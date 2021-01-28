package state

import (
	"bytes"
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

type nextSlotCache struct {
	sync.RWMutex
	root  []byte
	state *state.BeaconState
}

var (
	nsc nextSlotCache
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

// GetNextSlotState returns the saved state if the input root matches the root in `nextSlotCache`. Returns nil otherwise.
// This is useful to check before processing slots. With a cache hit, it will return last processed state with slot plus
// one advancement.
func GetNextSlotState(ctx context.Context, root []byte) (*state.BeaconState, error) {
	nsc.Lock()
	defer nsc.Unlock()
	if !bytes.Equal(root, nsc.root) {
		trailingSlotCacheMiss.Inc()
		return nil, nil
	}
	trailingSlotCacheHit.Inc()
	// Returning copied state.
	return nsc.state.Copy(), nil
}

// UpdateNextSlotCache updates the `nextSlotCache`. It saves the input state after advancing the state slot by 1
// by calling `ProcessSlots`, it also saves the input root for later look up.
// This is useful to call after successfully processing a block.
func UpdateNextSlotCache(ctx context.Context, root []byte, state *state.BeaconState) error {
	nsc.RLock()
	// Returning early if state exists in cache.
	if bytes.Equal(root, nsc.root) {
		defer nsc.RUnlock()
		return nil
	}
	nsc.RUnlock()

	// Advancing one slot by using a copied state.
	copied := state.Copy()
	copied, err := ProcessSlots(ctx, copied, copied.Slot()+1)
	if err != nil {
		return err
	}

	nsc.Lock()
	defer nsc.Unlock()

	nsc.root = root
	nsc.state = copied
	return nil
}
