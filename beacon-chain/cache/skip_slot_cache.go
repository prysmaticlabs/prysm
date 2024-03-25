package cache

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"go.opencensus.io/trace"
)

const (
	// maxSkipSlotCacheSize defines the max number of active balances that can be cached.
	maxSkipSlotCacheSize = int(8)
)

var (
	// Delay parameters
	minDelay    = float64(10)        // 10 nanoseconds
	maxDelay    = float64(100000000) // 0.1 second
	delayFactor = 1.1

	// Metrics
	skipSlotCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "skip_slot_cache_hit",
		Help: "The total number of cache hits on the skip slot cache.",
	})
	skipSlotCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "skip_slot_cache_miss",
		Help: "The total number of cache misses on the skip slot cache.",
	})
)

type addr = [32]byte

// SkipSlotCache is used to store the cached results of processing skip slots in transition.ProcessSlots.
type SkipSlotCache[K addr, V state.BeaconState] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter

	lock       sync.RWMutex
	disabled   bool // Allow for programmatic toggling of the cache, useful during initial sync.
	inProgress map[[32]byte]bool
}

// NewSkipSlotCache creates a new effective balance cache for storing/accessing total balance by epoch.
func NewSkipSlotCache[K addr, V state.BeaconState]() (*SkipSlotCache[K, V], error) {
	cache, err := lru.New[K, V](maxSkipSlotCacheSize)
	if err != nil {
		return nil, ErrCacheCannotBeNil
	}

	if skipSlotCacheMiss == nil || skipSlotCacheHit == nil {
		return nil, ErrCacheMetricsCannotBeNil
	}

	return &SkipSlotCache[K, V]{
		lru:           cache,
		promCacheMiss: skipSlotCacheMiss,
		promCacheHit:  skipSlotCacheHit,
		inProgress:    make(map[[32]byte]bool),
	}, nil
}

func (c *SkipSlotCache[K, V]) get() *lru.Cache[K, V] {
	return c.lru
}

func (c *SkipSlotCache[K, V]) hitCache() {
	c.promCacheHit.Inc()
}

func (c *SkipSlotCache[K, V]) missCache() {
	c.promCacheMiss.Inc()
}

// Clear the SkipSlotCache to its initial state
func (c *SkipSlotCache[K, V]) Clear() {
	purge[K, V](c)
}

// Enable the skip slot cache.
func (c *SkipSlotCache[K, V]) Enable() {
	c.disabled = false
}

// Disable the skip slot cache.
func (c *SkipSlotCache[K, V]) Disable() {
	c.disabled = true
}

// Get waits for any in progress calculation to complete before returning a
// cached response, if any.
func (c *SkipSlotCache[K, V]) Get(ctx context.Context, r K) (V, error) {
	ctx, span := trace.StartSpan(ctx, "skipSlotCache.Get")
	defer span.End()
	var noState V
	if c.disabled {
		// Return a miss result if cache is not enabled.
		skipSlotCacheMiss.Inc()
		return noState, nil
	}

	delay := minDelay

	// Another identical request may be in progress already. Let's wait until
	// any in progress request resolves or our timeout is exceeded.
	inProgress := false
	for {
		if ctx.Err() != nil {
			return noState, ctx.Err()
		}

		c.lock.RLock()
		if !c.inProgress[r] {
			c.lock.RUnlock()
			break
		}
		inProgress = true
		c.lock.RUnlock()

		// This increasing backoff is to decrease the CPU cycles while waiting
		// for the in progress boolean to flip to false.
		time.Sleep(time.Duration(delay) * time.Nanosecond)
		delay *= delayFactor
		delay = math.Min(delay, maxDelay)
	}
	span.AddAttributes(trace.BoolAttribute("inProgress", inProgress))

	item, err := get(c, r)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			span.AddAttributes(trace.BoolAttribute("hit", false))
			return noState, nil
		}
		return noState, err
	}

	span.AddAttributes(trace.BoolAttribute("hit", true))
	switch beaconState := any(item).(type) {
	case state.BeaconState:
		return beaconState.Copy().(V), nil
	}

	return noState, errors.Wrap(ErrCastingFailed, "item in cache is not of type state.BeaconState")
}

// MarkInProgress a request so that any other similar requests will block on
// Get until MarkNotInProgress is called.
func (c *SkipSlotCache[K, V]) MarkInProgress(r [32]byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.inProgress[r] {
		return ErrAlreadyInProgress
	}
	c.inProgress[r] = true
	return nil
}

// MarkNotInProgress will release the lock on a given request. This should be
// called after put.
func (c *SkipSlotCache[K, V]) MarkNotInProgress(r [32]byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.inProgress, r)
}

// Put the response in the cache.
func (c *SkipSlotCache[K, V]) Put(_ context.Context, r K, state V) error {
	if c.disabled {
		return nil
	}
	// Copy state so cached value is not mutated.
	return add(c, r, state.Copy().(V))
}
