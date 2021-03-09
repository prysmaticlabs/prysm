package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	// maxCheckpointStateSize defines the max number of entries check point to state cache can contain.
	// Choosing 10 to account for multiple forks, this allows 5 forks per epoch boundary with 2 epochs
	// window to accept attestation based on latest spec.
	maxCheckpointStateSize = 10

	// Metrics.
	checkpointStateMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "check_point_state_cache_miss",
		Help: "The number of check point state requests that aren't present in the cache.",
	})
	checkpointStateHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "check_point_state_cache_hit",
		Help: "The number of check point state requests that are present in the cache.",
	})
)

// CheckpointStateCache is a struct with 1 queue for looking up state by checkpoint.
type CheckpointStateCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewCheckpointStateCache creates a new checkpoint state cache for storing/accessing processed state.
func NewCheckpointStateCache() *CheckpointStateCache {
	cache, err := lru.New(maxCheckpointStateSize)
	if err != nil {
		panic(err)
	}
	return &CheckpointStateCache{
		cache: cache,
	}
}

// StateByCheckpoint fetches state by checkpoint. Returns true with a
// reference to the CheckpointState info, if exists. Otherwise returns false, nil.
func (c *CheckpointStateCache) StateByCheckpoint(cp *ethpb.Checkpoint) (iface.BeaconState, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	h, err := hashutil.HashProto(cp)
	if err != nil {
		return nil, err
	}

	item, exists := c.cache.Get(h)

	if exists && item != nil {
		checkpointStateHit.Inc()
		// Copy here is unnecessary since the return will only be used to verify attestation signature.
		return item.(iface.BeaconState), nil
	}

	checkpointStateMiss.Inc()
	return nil, nil
}

// AddCheckpointState adds CheckpointState object to the cache. This method also trims the least
// recently added CheckpointState object if the cache size has ready the max cache size limit.
func (c *CheckpointStateCache) AddCheckpointState(cp *ethpb.Checkpoint, s iface.ReadOnlyBeaconState) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	h, err := hashutil.HashProto(cp)
	if err != nil {
		return err
	}
	c.cache.Add(h, s)
	return nil
}
