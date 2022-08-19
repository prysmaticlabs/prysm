package stategen

import (
	"context"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
)

var (
	// hotStateCacheSize defines the max number of hot state this can cache.
	hotStateCacheSize = 32
	// Metrics
	hotStateCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hot_state_cache_hit",
		Help: "The total number of cache hits on the hot state cache.",
	})
	hotStateCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hot_state_cache_miss",
		Help: "The total number of cache misses on the hot state cache.",
	})
)

// hotStateCache is used to store the processed beacon state after finalized check point.
type hotStateCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// newHotStateCache initializes the map and underlying cache.
func newHotStateCache() *hotStateCache {
	return &hotStateCache{
		cache: lruwrpr.New(hotStateCacheSize),
	}
}

// Get returns a cached response via input block root, if any.
// The response is copied by default.
func (c *hotStateCache) get(blockRoot [32]byte) state.BeaconState {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(blockRoot)

	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(state.BeaconState).Copy()
	}
	hotStateCacheMiss.Inc()
	return nil
}

func (c *hotStateCache) ByBlockRoot(r [32]byte) (state.BeaconState, error) {
	st := c.get(r)
	if st == nil {
		return nil, ErrNotInCache
	}
	return st, nil
}

// GetWithoutCopy returns a non-copied cached response via input block root.
func (c *hotStateCache) getWithoutCopy(blockRoot [32]byte) state.BeaconState {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(blockRoot)
	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(state.BeaconState)
	}
	hotStateCacheMiss.Inc()
	return nil
}

// put the response in the cache.
func (c *hotStateCache) put(blockRoot [32]byte, state state.BeaconState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(blockRoot, state)
}

// has returns true if the key exists in the cache.
func (c *hotStateCache) has(blockRoot [32]byte) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cache.Contains(blockRoot)
}

// delete deletes the key exists in the cache.
func (c *hotStateCache) delete(blockRoot [32]byte) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.cache.Remove(blockRoot)
}

// FinalizedCheckpointer describes the forkchoice methods needed by the stategen service
type FinalizedCheckpointer interface {
	FinalizedCheckpoint() *forkchoicetypes.Checkpoint
}

// This tracks the config in the event of long non-finality,
// how often does the node save hot states to db? what are
// the saved hot states in db?... etc
type hotStateStatus struct {
	enabled                 bool
	lock                    sync.Mutex
	duration                types.Slot
	blockRootsOfSavedStates [][32]byte
	fc                      FinalizedCheckpointer
	cs                      CurrentSlotter
	db                      db.NoHeadAccessDatabase
}

// This checks whether it's time to start saving hot state to DB.
func (s *hotStateStatus) refresh(ctx context.Context) error {
	current := slots.ToEpoch(s.cs.CurrentSlot())
	fcp := s.fc.FinalizedCheckpoint()
	if fcp == nil {
		return errForkchoiceFinalizedNil
	}
	// don't allow underflow
	if fcp.Epoch > current {
		return errCurrentEpochBehindFinalized
	}

	if current-fcp.Epoch >= hotStateSaveThreshold {
		s.enableSaving()
		return nil
	}

	return s.disableSaving(ctx)
}

// enableHotStateSaving enters the mode that saves hot beacon state to the DB.
// This usually gets triggered when there's long duration since finality.
func (s *hotStateStatus) enableSaving() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.enabled {
		return
	}
	s.enabled = true

	log.WithFields(logrus.Fields{
		"enabled":       s.enabled,
		"slotsInterval": s.duration,
	}).Warn("Enabling hot state db persistence mode")
}

// disableHotStateSaving exits the mode that saves beacon state to DB for the hot states.
// This usually gets triggered once there's finality after long duration since finality.
func (s *hotStateStatus) disableSaving(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.enabled {
		return nil
	}

	log.WithFields(logrus.Fields{
		"enabled":          s.enabled,
		"deletedHotStates": len(s.blockRootsOfSavedStates),
	}).Warn("Disabling hot state db persistence mode")

	// Delete previous saved states in DB as we are turning this mode off.
	s.enabled = false
	if err := s.db.DeleteStates(ctx, s.blockRootsOfSavedStates); err != nil {
		return err
	}
	s.blockRootsOfSavedStates = nil

	return nil
}