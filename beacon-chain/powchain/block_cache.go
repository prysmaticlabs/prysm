package powchain

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/params"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotAHeaderInfo will be returned when a cache object is not a pointer to
	// a headerInfo struct.
	ErrNotAHeaderInfo = errors.New("object is not a header info")

	// maxCacheSize is 2x of the follow distance for additional cache padding.
	// Requests should be only accessing blocks within recent blocks within the
	// Eth1FollowDistance.
	maxCacheSize = 2 * params.BeaconConfig().Eth1FollowDistance

	// Metrics
	headerCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_header_cache_miss",
		Help: "The number of header requests that aren't present in the cache.",
	})
	headerCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_header_cache_hit",
		Help: "The number of header requests that are present in the cache.",
	})
	headerCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powchain_header_cache_size",
		Help: "The number of headers in the header cache",
	})
)

// headerInfo specifies the block header information in the ETH 1.0 chain.
type headerInfo struct {
	Number *big.Int
	Hash   common.Hash
	Time   uint64
}

func headerToHeaderInfo(hdr *gethTypes.Header) (*headerInfo, error) {
	if hdr.Number == nil {
		// A nil number will panic when calling *big.Int.Set(...)
		return nil, errors.New("cannot convert block header with nil block number")
	}

	return &headerInfo{
		Hash:   hdr.Hash(),
		Number: new(big.Int).Set(hdr.Number),
		Time:   hdr.Time,
	}, nil
}

// hashKeyFn takes the hex string representation as the key for a headerInfo.
func hashKeyFn(obj interface{}) (string, error) {
	hInfo, ok := obj.(*headerInfo)
	if !ok {
		return "", ErrNotAHeaderInfo
	}

	return hInfo.Hash.Hex(), nil
}

// heightKeyFn takes the string representation of the block header number as the key
// for a headerInfo.
func heightKeyFn(obj interface{}) (string, error) {
	hInfo, ok := obj.(*headerInfo)
	if !ok {
		return "", ErrNotAHeaderInfo
	}

	return hInfo.Number.String(), nil
}

// headerCache struct with two queues for looking up by hash or by block height.
type headerCache struct {
	hashCache   *cache.FIFO
	heightCache *cache.FIFO
	lock        sync.RWMutex
}

// newHeaderCache creates a new block cache for storing/accessing headerInfo from
// memory.
func newHeaderCache() *headerCache {
	return &headerCache{
		hashCache:   cache.NewFIFO(hashKeyFn),
		heightCache: cache.NewFIFO(heightKeyFn),
	}
}

// HeaderInfoByHash fetches headerInfo by its header hash. Returns true with a
// reference to the header info, if exists. Otherwise returns false, nil.
func (b *headerCache) HeaderInfoByHash(hash common.Hash) (bool, *headerInfo, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	obj, exists, err := b.hashCache.GetByKey(hash.Hex())
	if err != nil {
		return false, nil, err
	}

	if exists {
		headerCacheHit.Inc()
	} else {
		headerCacheMiss.Inc()
		return false, nil, nil
	}

	hInfo, ok := obj.(*headerInfo)
	if !ok {
		return false, nil, ErrNotAHeaderInfo
	}

	return true, hInfo, nil
}

// HeaderInfoByHeight fetches headerInfo by its header number. Returns true with a
// reference to the header info, if exists. Otherwise returns false, nil.
func (b *headerCache) HeaderInfoByHeight(height *big.Int) (bool, *headerInfo, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	obj, exists, err := b.heightCache.GetByKey(height.String())
	if err != nil {
		return false, nil, err
	}

	if exists {
		headerCacheHit.Inc()
	} else {
		headerCacheMiss.Inc()
		return false, nil, nil
	}

	hInfo, ok := obj.(*headerInfo)
	if !ok {
		return false, nil, ErrNotAHeaderInfo
	}

	return exists, hInfo, nil
}

// AddHeader adds a headerInfo object to the cache. This method also trims the
// least recently added header info if the cache size has reached the max cache
// size limit. This method should be called in sequential header number order if
// the desired behavior is that the blocks with the highest header number should
// be present in the cache.
func (b *headerCache) AddHeader(hdr *gethTypes.Header) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	hInfo, err := headerToHeaderInfo(hdr)
	if err != nil {
		return err
	}

	if err := b.hashCache.AddIfNotPresent(hInfo); err != nil {
		return err
	}
	if err := b.heightCache.AddIfNotPresent(hInfo); err != nil {
		return err
	}

	trim(b.hashCache, maxCacheSize)
	trim(b.heightCache, maxCacheSize)

	headerCacheSize.Set(float64(len(b.hashCache.ListKeys())))

	return nil
}

// trim the FIFO queue to the maxSize.
func trim(queue *cache.FIFO, maxSize uint64) {
	for s := uint64(len(queue.ListKeys())); s > maxSize; s-- {
		// #nosec G104 popProcessNoopFunc never returns an error
		if _, err := queue.Pop(popProcessNoopFunc); err != nil { // This never returns an error, but we'll handle anyway for sanity.
			panic(err)
		}
	}
}

// popProcessNoopFunc is a no-op function that never returns an error.
func popProcessNoopFunc(_ interface{}) error {
	return nil
}
