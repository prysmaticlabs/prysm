package cache

import (
	"context"
	"math"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
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
	cache      *lru.Cache
	lock       sync.RWMutex
	inProgress map[string]bool
}

// NewCheckpointStateCache creates a new checkpoint state cache for storing/accessing processed state.
func NewCheckpointStateCache() *CheckpointStateCache {
	cache, err := lru.New(maxCheckpointStateSize)
	if err != nil {
		panic(err)
	}
	return &CheckpointStateCache{
		cache:      cache,
		inProgress: map[string]bool{},
	}
}

// StateByCheckpoint fetches state by checkpoint. Returns true with a
// reference to the CheckpointState info, if exists. Otherwise returns false, nil.
func (c *CheckpointStateCache) StateByCheckpoint(ctx context.Context, cp *ethpb.Checkpoint) (*stateTrie.BeaconState, error) {
	k, err := checkpointKey(cp)
	if err != nil {
		return nil, err
	}

	delay := minDelay

	// Another identical request may be in progress already. Let's wait until
	// any in progress request resolves or our timeout is exceeded.
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		c.lock.RLock()
		if !c.inProgress[k] {
			c.lock.RUnlock()
			break
		}
		c.lock.RUnlock()

		// This increasing backoff is to decrease the CPU cycles while waiting
		// for the in progress boolean to flip to false.
		time.Sleep(time.Duration(delay) * time.Nanosecond)
		delay *= delayFactor
		delay = math.Min(delay, maxDelay)
	}

	item, exists := c.cache.Get(k)
	if exists && item != nil {
		checkpointStateHit.Inc()
		// Copy here is unnecessary since the return will only be used to verify attestation signature.
		return item.(*stateTrie.BeaconState), nil
	}

	checkpointStateMiss.Inc()
	return nil, nil
}

// AddCheckpointState adds CheckpointState object to the cache. This method also trims the least
// recently added CheckpointState object if the cache size has ready the max cache size limit.
func (c *CheckpointStateCache) AddCheckpointState(cp *ethpb.Checkpoint, s *stateTrie.BeaconState) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	k, err := checkpointKey(cp)
	if err != nil {
		return err
	}
	c.cache.Add(k, s)
	return nil
}

// MarkInProgress a request so that any other similar requests will block on
// Get until MarkNotInProgress is called.
func (c *CheckpointStateCache) MarkInProgress(cp *ethpb.Checkpoint) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	k, err := checkpointKey(cp)
	if err != nil {
		return err
	}
	if c.inProgress[k] {
		return ErrAlreadyInProgress
	}
	c.inProgress[k] = true
	return nil
}

// MarkNotInProgress will release the lock on a given request. This should be
// called after put.
func (c *CheckpointStateCache) MarkNotInProgress(cp *ethpb.Checkpoint) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	k, err := checkpointKey(cp)
	if err != nil {
		return err
	}
	delete(c.inProgress, k)
	return nil
}

func checkpointKey(cp *ethpb.Checkpoint) (string, error) {
	h, err := hashutil.HashProto(cp)
	if err != nil {
		return "", err
	}
	return string(h[:]), err
}
