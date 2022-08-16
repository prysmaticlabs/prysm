//go:build !fuzz

package cache

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"k8s.io/client-go/tools/cache"
)

var (
	// maxProposerIndicesCacheSize defines the max number of proposer indices on per block root basis can cache.
	// Due to reorgs and long finality, it's good to keep the old cache around for quickly switch over.
	maxProposerIndicesCacheSize = uint64(8)

	// ProposerIndicesCacheMiss tracks the number of proposerIndices requests that aren't present in the cache.
	ProposerIndicesCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proposer_indices_cache_miss",
		Help: "The number of proposer indices requests that aren't present in the cache.",
	})
	// ProposerIndicesCacheHit tracks the number of proposerIndices requests that are in the cache.
	ProposerIndicesCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proposer_indices_cache_hit",
		Help: "The number of proposer indices requests that are present in the cache.",
	})
)

// ProposerIndicesCache is a struct with 1 queue for looking up proposer indices by root.
type ProposerIndicesCache struct {
	proposerIndicesCache *cache.FIFO
	lock                 sync.RWMutex
}

// proposerIndicesKeyFn takes the block root as the key to retrieve proposer indices in a given epoch.
func proposerIndicesKeyFn(obj interface{}) (string, error) {
	info, ok := obj.(*ProposerIndices)
	if !ok {
		return "", ErrNotProposerIndices
	}

	return key(info.BlockRoot), nil
}

// NewProposerIndicesCache creates a new proposer indices cache for storing/accessing proposer index assignments of an epoch.
func NewProposerIndicesCache() *ProposerIndicesCache {
	return &ProposerIndicesCache{
		proposerIndicesCache: cache.NewFIFO(proposerIndicesKeyFn),
	}
}

// AddProposerIndices adds ProposerIndices object to the cache.
// This method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *ProposerIndicesCache) AddProposerIndices(p *ProposerIndices) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.proposerIndicesCache.AddIfNotPresent(p); err != nil {
		return err
	}
	trim(c.proposerIndicesCache, maxProposerIndicesCacheSize)
	return nil
}

// HasProposerIndices returns the proposer indices of a block root seed.
func (c *ProposerIndicesCache) HasProposerIndices(r [32]byte) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, exists, err := c.proposerIndicesCache.GetByKey(key(r))
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ProposerIndices returns the proposer indices of a block root seed.
func (c *ProposerIndicesCache) ProposerIndices(r [32]byte) ([]types.ValidatorIndex, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.proposerIndicesCache.GetByKey(key(r))
	if err != nil {
		return nil, err
	}

	if exists {
		ProposerIndicesCacheHit.Inc()
	} else {
		ProposerIndicesCacheMiss.Inc()
		return nil, nil
	}

	item, ok := obj.(*ProposerIndices)
	if !ok {
		return nil, ErrNotProposerIndices
	}

	return item.ProposerIndices, nil
}

// Len returns the number of keys in the underlying cache.
func (c *ProposerIndicesCache) Len() int {
	return len(c.proposerIndicesCache.ListKeys())
}
