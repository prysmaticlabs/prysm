package cache

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
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
type (
	CheckpointHash                                = string
	CheckpointStateCache[K CheckpointHash, V any] struct {
		lru                         *lru.Cache[K, V]
		promCacheMiss, promCacheHit prometheus.Counter
	}
)

// NewCheckpointStateCache creates a new effective balance cache for storing/accessing total balance by epoch.
func NewCheckpointStateCache[K CheckpointHash, V any]() (*CheckpointStateCache[K, V], error) {
	cache, err := lru.New[K, V](maxCheckpointStateSize)
	if err != nil {
		return nil, ErrCacheCannotBeNil
	}

	if checkpointStateMiss == nil || checkpointStateHit == nil {
		return nil, ErrCacheMetricsCannotBeNil
	}

	return &CheckpointStateCache[K, V]{
		lru:           cache,
		promCacheMiss: checkpointStateMiss,
		promCacheHit:  checkpointStateHit,
	}, nil
}

func (c *CheckpointStateCache[K, V]) get() *lru.Cache[K, V] {
	return c.lru
}

func (c *CheckpointStateCache[K, V]) hitCache() {
	c.promCacheHit.Inc()
}

func (c *CheckpointStateCache[K, V]) missCache() {
	c.promCacheMiss.Inc()
}

// StateByCheckpoint fetches state by checkpoint. Returns true with a
// reference to the CheckpointState info, if exists. Otherwise returns false, nil.
func (c *CheckpointStateCache[K, V]) StateByCheckpoint(cp *ethpb.Checkpoint) (state.BeaconState, error) {
	key, err := checkpointStateKey[K](cp)
	if err != nil {
		return nil, err
	}

	item, err := get[K, V](c, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	switch beaconState := any(item).(type) {
	case state.BeaconState:
		// Copy here is unnecessary since the return will only be used to verify attestation signature.
		return beaconState, nil
	}

	return nil, errors.Wrapf(ErrCastingFailed, "%s -> %s", "state.ReadOnlyBeaconState", "state.BeaconState")
}

// AddCheckpointState adds CheckpointState object to the cache. This method also trims the least
// recently added CheckpointState object if the cache size has ready the max cache size limit.
func (c *CheckpointStateCache[K, V]) AddCheckpointState(cp *ethpb.Checkpoint, s V) error {
	key, err := checkpointStateKey[K](cp)
	if err != nil {
		return err
	}

	return add[K, V](c, key, s)
}

func (c *CheckpointStateCache[K, V]) Keys() []K {
	return keys[K, V](c)
}

func (c *CheckpointStateCache[K, V]) Clear() {
	purge[K, V](c)
}

func checkpointStateKey[K CheckpointHash](cp *ethpb.Checkpoint) (K, error) {
	var noKey K
	h, err := hash.Proto(cp)
	if err != nil {
		return noKey, err
	}
	return K(fmt.Sprintf("%s", h)), nil
}
