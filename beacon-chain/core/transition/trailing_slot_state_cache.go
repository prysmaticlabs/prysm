package transition

import (
	"bytes"
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
)

type nextSlotCache struct {
	sync.RWMutex
	root  []byte
	state state.BeaconState
}

var (
	nsc nextSlotCache
	// Metrics for the validator cache.
	nextSlotCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "next_slot_cache_hit",
		Help: "The total number of cache hits on the next slot state cache.",
	})
	nextSlotCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "next_slot_cache_miss",
		Help: "The total number of cache misses on the next slot state cache.",
	})
)

// NextSlotState returns the saved state if the input root matches the root in `nextSlotCache`. Returns nil otherwise.
// This is useful to check before processing slots. With a cache hit, it will return last processed state with slot plus
// one advancement.
func NextSlotState(_ context.Context, root []byte) (state.BeaconState, error) {
	nsc.RLock()
	defer nsc.RUnlock()
	if !bytes.Equal(root, nsc.root) || bytes.Equal(root, []byte{}) {
		nextSlotCacheMiss.Inc()
		return nil, nil
	}
	nextSlotCacheHit.Inc()
	// Returning copied state.
	return nsc.state.Copy(), nil
}

// UpdateNextSlotCache updates the `nextSlotCache`. It saves the input state after advancing the state slot by 1
// by calling `ProcessSlots`, it also saves the input root for later look up.
// This is useful to call after successfully processing a block.
func UpdateNextSlotCache(ctx context.Context, root []byte, state state.BeaconState) error {
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
