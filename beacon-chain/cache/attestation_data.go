package cache

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

const maxSize = 4

var (
	// Prometheus counters for cache hits and misses.
	attestationCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "attestation_cache_miss",
		Help: "Number of cache misses for attestation data requests.",
	})
	attestationCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "attestation_cache_hit",
		Help: "Number of cache hits for attestation data requests.",
	})
)

// AttestationCache stores cached results of AttestationData requests.
type AttestationCache struct {
	cache map[primitives.Slot]*ethpb.AttestationData
	lock  sync.RWMutex
}

// NewAttestationCache creates a new instance of AttestationCache.
func NewAttestationCache() *AttestationCache {
	return &AttestationCache{
		cache: make(map[primitives.Slot]*ethpb.AttestationData),
	}
}

// Get retrieves cached attestation data, recording a cache hit or miss.
func (c *AttestationCache) Get(ctx context.Context, req *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	if res, exists := c.cache[req.Slot]; exists {
		attestationCacheHit.Inc()
		return ethpb.CopyAttestationData(res), nil
	}

	attestationCacheMiss.Inc()
	return nil, nil
}

// Put adds a response to the cache.
func (c *AttestationCache) Put(ctx context.Context, req *ethpb.AttestationDataRequest, res *ethpb.AttestationData) error {
	if req == nil || res == nil {
		return errors.New("request or response cannot be nil")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache[req.Slot] = ethpb.CopyAttestationData(res)

	c.trimCache()

	return nil
}

// trimCache trims the cache to maintain a maximum size.
func (c *AttestationCache) trimCache() {
	if maxSize > len(c.cache) {
		return
	}

	minSlot := primitives.Slot(math.MaxInt64)
	for slot := range c.cache {
		if slot < minSlot {
			minSlot = slot
		}
	}

	delete(c.cache, minSlot)
}
