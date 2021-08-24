// +build !libfuzzer

package cache

import (
	"encoding/binary"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	// maxBalanceCacheSize defines the max number of active balance on per epoch basis can cache.
	maxBalanceCacheSize = uint64(4)

	// BalanceCacheMiss tracks the number of balance requests that aren't present in the cache.
	balanceCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "balance_cache_miss",
		Help: "The number of Get requests that aren't present in the cache.",
	})
	// BalanceCacheHit tracks the number of balance requests that are in the cache.
	balanceCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "balance_cache_hit",
		Help: "The number of Get requests that are present in the cache.",
	})


)

// BalanceCache is a struct with 1 LRU cache for looking up balance by epoch.
type BalanceCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewBalanceCache creates a new balance cache for storing/accessing total balance by epoch.
func NewBalanceCache() *BalanceCache {
	c, err := lru.New(int(maxBalanceCacheSize))
	// An error is only returned if the size of the cache is <= 0.
	if err != nil {
		panic(err)
	}
	return &BalanceCache{
		cache: c,
	}
}

// AddBalance adds a new balance entry for current balance for state `st` into the cache.
func (c *BalanceCache) AddBalance(st state.BeaconState, balance uint64) error {
	key, err := balanceKey(st)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	_ = c.cache.Add(key, balance)
	return nil
}

// Get returns the current epoch's balance for state `st` in cache.
func (c *BalanceCache) Get(st state.BeaconState) (uint64, error) {
	key, err := balanceKey(st)
	if err != nil {
		return 0, err
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	value, exists := c.cache.Get(key)
	if !exists {
		balanceCacheMiss.Inc()
		return 0, ErrNonExistingBalanceKey
	}
	balanceCacheHit.Inc()
	return value.(uint64), nil
}

// Given input state `st`, balance key is constructed as:
// (block_root in `st` at epoch_start_slot - 1) + current_epoch
func balanceKey(st state.BeaconState) (string, error) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	currentEpoch := st.Slot().DivSlot(slotsPerEpoch)
	epochStartSlot, err := slotsPerEpoch.SafeMul(uint64(currentEpoch))
	if err != nil {
		// impossible condition due to early division
		return "", errors.Errorf("start slot calculation overflows: %v", err)
	}
	prevSlot := epochStartSlot - 1
	r, err := st.BlockRootAtIndex(uint64(prevSlot % params.BeaconConfig().SlotsPerHistoricalRoot))
	if err != nil {
		// impossible condition because index is always constrained within state
		return "", err
	}

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(currentEpoch))
	return string(append(r, b...)), nil
}
