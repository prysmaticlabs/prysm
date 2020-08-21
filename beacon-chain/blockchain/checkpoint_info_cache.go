package blockchain

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	// This defines the max number of checkpoint info this cache can store.
	// Each cache is calculated at 3MB(30K validators), the total cache size is around 100MB.
	// Due to reorgs and long finality, it's good to keep the old cache around for quickly switch over.
	maxInfoSize = 32

	// This tracks the number of check point info requests that aren't present in the cache.
	infoMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "check_point_info_cache_miss",
		Help: "The number of check point info requests that aren't present in the cache.",
	})
	// This tracks the number of check point info requests that are in the cache.
	infoHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "check_point_info_cache_hit",
		Help: "The number of check point info requests that are present in the cache.",
	})
)

// checkPtInfoCache is a struct with 1 LRU cache for looking up check point info by checkpoint.
type checkPtInfoCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// newCheckPointInfoCache creates a new checkpoint info cache for storing/accessing processed check point info object.
func newCheckPointInfoCache() *checkPtInfoCache {
	cache, err := lru.New(maxInfoSize)
	if err != nil {
		panic(err)
	}
	return &checkPtInfoCache{
		cache: cache,
	}
}

// get fetches check point info by check point. Returns the reference of the CheckPtInfo, nil if doesn't exist.
func (c *checkPtInfoCache) get(cp *ethpb.Checkpoint) (*pb.CheckPtInfo, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	h, err := hashutil.HashProto(cp)
	if err != nil {
		return nil, err
	}

	item, exists := c.cache.Get(h)

	if exists && item != nil {
		infoHit.Inc()
		// Copy here is unnecessary since the returned check point info object
		// will only be used to verify attestation signature.
		return item.(*pb.CheckPtInfo), nil
	}

	infoMiss.Inc()
	return nil, nil
}

// put adds CheckPtInfo object to the cache. This method also trims the least
// recently added CheckPtInfo object if the cache size has ready the max cache size limit.
func (c *checkPtInfoCache) put(cp *ethpb.Checkpoint, info *pb.CheckPtInfo) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	h, err := hashutil.HashProto(cp)
	if err != nil {
		return err
	}

	c.cache.Add(h, info)
	return nil
}
