package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	a    *ethpb.AttestationData
	lock sync.RWMutex
}

// NewAttestationCache creates a new instance of AttestationCache.
func NewAttestationCache() *AttestationCache {
	return &AttestationCache{}
}

// Get retrieves cached attestation data, recording a cache hit or miss.
func (c *AttestationCache) Get(ctx context.Context, req *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	if req.Slot == c.a.Slot {
		attestationCacheHit.Inc()
		return ethpb.CopyAttestationData(c.a), nil
	}
	attestationCacheMiss.Inc()
	return nil, nil
}

// Put adds a response to the cache.
func (c *AttestationCache) Put(ctx context.Context, res *ethpb.AttestationData) error {
	if res == nil {
		return errors.New("attestation cannot be nil")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.a = res

	return nil
}
