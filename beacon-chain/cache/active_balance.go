//go:build !fuzz

package cache

import (
	"encoding/binary"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

const (
	// maxBalanceCacheSize defines the max number of active balances that can be cached.
	maxBalanceCacheSize = int(4)
)

var (
	// BalanceCacheMiss tracks the number of balance requests that aren't present in the cache.
	balanceCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_effective_balance_cache_miss",
		Help: "The number of get requests that aren't present in the cache.",
	})
	// BalanceCacheHit tracks the number of balance requests that are in the cache.
	balanceCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_effective_balance_cache_hit",
		Help: "The number of get requests that are present in the cache.",
	})
)

// BalanceCache is a struct with 1 LRU cache for looking up balance by epoch.
type BalanceCache[K string, V uint64] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter
}

// NewEffectiveBalanceCache creates a new effective balance cache for storing/accessing total balance by epoch.
func NewEffectiveBalanceCache[K string, V uint64]() (*BalanceCache[K, V], error) {
	cache, err := lru.New[K, V](maxBalanceCacheSize)
	if err != nil {
		return nil, err
	}

	if balanceCacheMiss == nil || balanceCacheHit == nil {
		return nil, errors.New("balance cache prometheus metrics are not initialized")
	}

	return &BalanceCache[K, V]{
		lru:           cache,
		promCacheMiss: balanceCacheMiss,
		promCacheHit:  balanceCacheHit,
	}, nil
}

func (c *BalanceCache[K, V]) get() *lru.Cache[K, V] {
	return c.lru
}

func (c *BalanceCache[K, V]) hitCache() {
	c.promCacheHit.Inc()
}

func (c *BalanceCache[K, V]) missCache() {
	c.promCacheMiss.Inc()
}

// Clear the BalanceCache to its initial state
func (c *BalanceCache[K, V]) Clear() {
	Purge[K, V](c)
}

// AddTotalEffectiveBalance adds a new total effective balance entry for current balance for state `st` into the cache.
func (c *BalanceCache[K, V]) AddTotalEffectiveBalance(st state.ReadOnlyBeaconState, balance V) error {
	key, err := c.balanceCacheKey(st)
	if err != nil {
		return err
	}

	Add[K, V](c, key, balance)
	return nil
}

// Get returns the current epoch's effective balance for state `st` in cache.
func (c *BalanceCache[K, V]) Get(st state.ReadOnlyBeaconState) (balance V, err error) {
	var (
		zero V
		key  K
	)
	if key, err = c.balanceCacheKey(st); err != nil {
		return zero, err
	}

	return Get[K, V](c, key)
}

// Given input state `st`, balance key is constructed as:
// (block_root in `st` at epoch_start_slot - 1) + current_epoch + validator_count
func (c *BalanceCache[K, V]) balanceCacheKey(st state.ReadOnlyBeaconState) (keyStr K, err error) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	currentEpoch := st.Slot().DivSlot(slotsPerEpoch)

	var epochStartSlot primitives.Slot
	if epochStartSlot, err = slotsPerEpoch.SafeMul(uint64(currentEpoch)); err != nil {
		// impossible condition due to early division
		return keyStr, errors.Errorf("start slot calculation overflows: %v", err)
	}
	prevSlot := primitives.Slot(0)
	if epochStartSlot > 1 {
		prevSlot = epochStartSlot - 1
	}

	var r []byte
	if r, err = st.BlockRootAtIndex(uint64(prevSlot % params.BeaconConfig().SlotsPerHistoricalRoot)); err != nil {
		// impossible condition because index is always constrained within state
		return keyStr, err
	}

	// Mix in current epoch
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(currentEpoch))
	key := append(r, b...)

	// Mix in validator count
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(st.NumValidators()))
	key = append(key, b...)
	keyStr = K(key)

	return
}
