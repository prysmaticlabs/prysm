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
	// ErrNotABlockInfo will be returned when a cache object is not a pointer to
	// a blockInfo struct.
	ErrNotABlockInfo = errors.New("object is not a block info")

	// maxCacheSize is 2x of the follow distance for additional cache padding.
	// Requests should be only accessing blocks within recent blocks within the
	// Eth1FollowDistance.
	maxCacheSize = 2 * params.BeaconConfig().Eth1FollowDistance

	// Metrics
	blockCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_block_cache_miss",
		Help: "The number of block requests that aren't present in the cache.",
	})
	blockCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_block_cache_hit",
		Help: "The number of block requests that are present in the cache.",
	})
	blockCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powchain_block_cache_size",
		Help: "The number of blocks in the block cache",
	})
)

// blockInfo specifies the block information in the ETH 1.0 chain.
type blockInfo struct {
	Number *big.Int
	Hash   common.Hash
	Time   uint64
}

func blockToBlockInfo(blk *gethTypes.Block) *blockInfo {
	return &blockInfo{
		Hash:   blk.Hash(),
		Number: blk.Number(),
		Time:   blk.Time(),
	}
}

// hashKeyFn takes the hex string representation as the key for a blockInfo.
func hashKeyFn(obj interface{}) (string, error) {
	bInfo, ok := obj.(*blockInfo)
	if !ok {
		return "", ErrNotABlockInfo
	}

	return bInfo.Hash.Hex(), nil
}

// heightKeyFn takes the string representation of the block number as the key
// for a blockInfo.
func heightKeyFn(obj interface{}) (string, error) {
	bInfo, ok := obj.(*blockInfo)
	if !ok {
		return "", ErrNotABlockInfo
	}

	return bInfo.Number.String(), nil
}

// blockCache struct with two queues for looking up by hash or by block height.
type blockCache struct {
	hashCache   *cache.FIFO
	heightCache *cache.FIFO
	lock        sync.RWMutex
}

// newBlockCache creates a new block cache for storing/accessing blockInfo from
// memory.
func newBlockCache() *blockCache {
	return &blockCache{
		hashCache:   cache.NewFIFO(hashKeyFn),
		heightCache: cache.NewFIFO(heightKeyFn),
	}
}

// BlockInfoByHash fetches blockInfo by its block hash. Returns true with a
// reference to the block info, if exists. Otherwise returns false, nil.
func (b *blockCache) BlockInfoByHash(hash common.Hash) (bool, *blockInfo, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	obj, exists, err := b.hashCache.GetByKey(hash.Hex())
	if err != nil {
		return false, nil, err
	}

	if exists {
		blockCacheHit.Inc()
	} else {
		blockCacheMiss.Inc()
		return false, nil, nil
	}

	bInfo, ok := obj.(*blockInfo)
	if !ok {
		return false, nil, ErrNotABlockInfo
	}

	return true, bInfo, nil
}

// BlockInfoByHeight fetches blockInfo by its block number. Returns true with a
// reference to the block info, if exists. Otherwise returns false, nil.
func (b *blockCache) BlockInfoByHeight(height *big.Int) (bool, *blockInfo, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	obj, exists, err := b.heightCache.GetByKey(height.String())
	if err != nil {
		return false, nil, err
	}

	if exists {
		blockCacheHit.Inc()
	} else {
		blockCacheMiss.Inc()
		return false, nil, nil
	}

	bInfo, ok := obj.(*blockInfo)
	if !ok {
		return false, nil, ErrNotABlockInfo
	}

	return exists, bInfo, nil
}

// AddBlock adds a blockInfo object to the cache. This method also trims the
// least recently added block info if the cache size has reached the max cache
// size limit. This method should be called in sequential block number order if
// the desired behavior is that the blocks with the highest block number should
// be present in the cache.
func (b *blockCache) AddBlock(blk *gethTypes.Block) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	bInfo := blockToBlockInfo(blk)

	if err := b.hashCache.AddIfNotPresent(bInfo); err != nil {
		return err
	}
	if err := b.heightCache.AddIfNotPresent(bInfo); err != nil {
		return err
	}

	trim(b.hashCache, maxCacheSize)
	trim(b.heightCache, maxCacheSize)

	blockCacheSize.Set(float64(len(b.hashCache.ListKeys())))

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
func popProcessNoopFunc(obj interface{}) error {
	return nil
}
