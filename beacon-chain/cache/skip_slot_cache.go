package cache

import (
	"context"
	"math"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"go.opencensus.io/trace"
)

var (
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

// SkipSlotCache is used to store the cached results of processing skip slots in state.ProcessSlots.
type SkipSlotCache struct {
	cache      *lru.Cache
	lock       sync.RWMutex
	disabled   bool // Allow for programmatic toggling of the cache, useful during initial sync.
	inProgress map[[32]byte]bool
}

// NewSkipSlotCache initializes the map and underlying cache.
func NewSkipSlotCache() *SkipSlotCache {
	cache, err := lru.New(8)
	if err != nil {
		panic(err)
	}
	return &SkipSlotCache{
		cache:      cache,
		inProgress: make(map[[32]byte]bool),
	}
}

// Enable the skip slot cache.
func (c *SkipSlotCache) Enable() {
	c.disabled = false
}

// Disable the skip slot cache.
func (c *SkipSlotCache) Disable() {
	c.disabled = true
}

// Get waits for any in progress calculation to complete before returning a
// cached response, if any.
func (c *SkipSlotCache) Get(ctx context.Context, r [32]byte) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "skipSlotCache.Get")
	defer span.End()
	if c.disabled {
		// Return a miss result if cache is not enabled.
		skipSlotCacheMiss.Inc()
		return nil, nil
	}

	delay := minDelay

	// Another identical request may be in progress already. Let's wait until
	// any in progress request resolves or our timeout is exceeded.
	inProgress := false
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
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

	item, exists := c.cache.Get(r)

	if exists && item != nil {
		skipSlotCacheHit.Inc()
		span.AddAttributes(trace.BoolAttribute("hit", true))
		return item.(iface.BeaconState).Copy(), nil
	}
	skipSlotCacheMiss.Inc()
	span.AddAttributes(trace.BoolAttribute("hit", false))
	return nil, nil
}

// MarkInProgress a request so that any other similar requests will block on
// Get until MarkNotInProgress is called.
func (c *SkipSlotCache) MarkInProgress(r [32]byte) error {
	if c.disabled {
		return nil
	}

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
func (c *SkipSlotCache) MarkNotInProgress(r [32]byte) error {
	if c.disabled {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.inProgress, r)
	return nil
}

// Put the response in the cache.
func (c *SkipSlotCache) Put(_ context.Context, r [32]byte, state iface.BeaconState) error {
	if c.disabled {
		return nil
	}

	// Copy state so cached value is not mutated.
	c.cache.Add(r, state.Copy())

	return nil
}
