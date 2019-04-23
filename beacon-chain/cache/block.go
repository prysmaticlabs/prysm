package cache

import (
	"errors"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotAncestorCacheObj will be returned when a cache object is not a pointer to
	// block ancestor cache obj.
	ErrNotAncestorCacheObj = errors.New("object is not an ancestor object for cache")
	// Metrics
	ancestorBlockCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ancestor_block_cache_miss",
		Help: "The number of ancestor block requests that aren't present in the cache.",
	})
	ancestorBlockCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ancestor_block_cache_hit",
		Help: "The number of ancestor block requests that are present in the cache.",
	})
	ancestorBlockCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ancestor_block_cache_size",
		Help: "The number of ancestor blocks in the ancestorBlock cache",
	})
)

// AncestorInfo defines the cached ancestor block object for height.
type AncestorInfo struct {
	Height uint64
	Hash   []byte
	Target *pb.AttestationTarget
}

// AncestorBlockCache structs with 1 queue for looking up block ancestor by height.
type AncestorBlockCache struct {
	ancestorBlockCache *cache.FIFO
	lock               sync.RWMutex
}

// heightKeyFn takes the string representation of the block hash + height as the key
// for the ancestor of a given block (AncestorInfo).
func heightKeyFn(obj interface{}) (string, error) {
	aInfo, ok := obj.(*AncestorInfo)
	if !ok {
		return "", ErrNotAncestorCacheObj
	}

	return string(aInfo.Hash) + strconv.Itoa(int(aInfo.Height)), nil
}

// NewBlockAncestorCache creates a new block ancestor cache for storing/accessing block ancestor
// from memory.
func NewBlockAncestorCache() *AncestorBlockCache {
	return &AncestorBlockCache{
		ancestorBlockCache: cache.NewFIFO(heightKeyFn),
	}
}

// AncestorBySlot fetches block's ancestor by height. Returns true with a
// reference to the ancestor block, if exists. Otherwise returns false, nil.
func (a *AncestorBlockCache) AncestorBySlot(blockHash []byte, height uint64) (*AncestorInfo, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	obj, exists, err := a.ancestorBlockCache.GetByKey(string(blockHash) + strconv.Itoa(int(height)))
	if err != nil {
		return nil, err
	}

	if exists {
		ancestorBlockCacheHit.Inc()
	} else {
		ancestorBlockCacheMiss.Inc()
		return nil, nil
	}

	aInfo, ok := obj.(*AncestorInfo)
	if !ok {
		return nil, ErrNotACommitteeInfo
	}

	return aInfo, nil
}

// AddBlockAncestor adds block ancestor object to the cache. This method also trims the least
// recently added ancestor if the cache size has ready the max cache size limit.
func (a *AncestorBlockCache) AddBlockAncestor(ancestorInfo *AncestorInfo) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.ancestorBlockCache.AddIfNotPresent(ancestorInfo); err != nil {
		return err
	}

	trim(a.ancestorBlockCache, maxCacheSize)
	ancestorBlockCacheSize.Set(float64(len(a.ancestorBlockCache.ListKeys())))
	return nil
}
