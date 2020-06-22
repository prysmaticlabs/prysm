package cache

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotCheckpointState will be returned when a cache object is not a pointer to
	// a CheckpointState struct.
	ErrNotCheckpointState = errors.New("object is not a state by check point struct")

	// maxCheckpointStateSize defines the max number of entries check point to state cache can contain.
	// Choosing 10 to account for multiple forks, this allows 5 forks per epoch boundary with 2 epochs
	// window to accept attestation based on latest spec.
	maxCheckpointStateSize = uint64(10)

	// Metrics.
	checkpointStateMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "check_point_statecache_miss",
		Help: "The number of check point state requests that aren't present in the cache.",
	})
	checkpointStateHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "check_point_state_cache_hit",
		Help: "The number of check point state requests that are present in the cache.",
	})
)

// CheckpointState defines the active validator indices per epoch.
type CheckpointState struct {
	Checkpoint *ethpb.Checkpoint
	State      *stateTrie.BeaconState
}

// CheckpointStateCache is a struct with 1 queue for looking up state by checkpoint.
type CheckpointStateCache struct {
	cache *cache.FIFO
	lock  sync.RWMutex
}

// checkpointState takes the checkpoint as the key of the resulting state.
func checkpointState(obj interface{}) (string, error) {
	info, ok := obj.(*CheckpointState)
	if !ok {
		return "", ErrNotCheckpointState
	}

	h, err := hashutil.HashProto(info.Checkpoint)
	if err != nil {
		return "", err
	}
	return string(h[:]), nil
}

// NewCheckpointStateCache creates a new checkpoint state cache for storing/accessing processed state.
func NewCheckpointStateCache() *CheckpointStateCache {
	return &CheckpointStateCache{
		cache: cache.NewFIFO(checkpointState),
	}
}

// StateByCheckpoint fetches state by checkpoint. Returns true with a
// reference to the CheckpointState info, if exists. Otherwise returns false, nil.
func (c *CheckpointStateCache) StateByCheckpoint(cp *ethpb.Checkpoint) (*stateTrie.BeaconState, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	h, err := hashutil.HashProto(cp)
	if err != nil {
		return nil, err
	}

	obj, exists, err := c.cache.GetByKey(string(h[:]))
	if err != nil {
		return nil, err
	}

	if exists {
		checkpointStateHit.Inc()
	} else {
		checkpointStateMiss.Inc()
		return nil, nil
	}

	info, ok := obj.(*CheckpointState)
	if !ok {
		return nil, ErrNotCheckpointState
	}

	// Copy here is unnecessary since the return will only be used to verify attestation signature.
	return info.State, nil
}

// AddCheckpointState adds CheckpointState object to the cache. This method also trims the least
// recently added CheckpointState object if the cache size has ready the max cache size limit.
func (c *CheckpointStateCache) AddCheckpointState(cp *CheckpointState) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.cache.AddIfNotPresent(&CheckpointState{
		Checkpoint: stateTrie.CopyCheckpoint(cp.Checkpoint),
		State:      cp.State.Copy(),
	}); err != nil {
		return err
	}

	trim(c.cache, maxCheckpointStateSize)
	return nil
}

// CheckpointStateKeys returns the keys of the state in cache.
func (c *CheckpointStateCache) CheckpointStateKeys() []string {
	return c.cache.ListKeys()
}
