package stategen

import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
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

type PersistenceMode int

const (
	// PersistenceModeMemory means the hot state cache does write to the database
	PersistenceModeMemory PersistenceMode = iota
	PersistenceModeSnapshot
)

func (m PersistenceMode) String() string {
	switch m {
	case PersistenceModeMemory:
		return "memory"
	case PersistenceModeSnapshot:
		return "snapshot"
	default:
		return "unknown"
	}
}

func NewHotStateSaver(d db.NoHeadAccessDatabase, fc FinalizedCheckpointer, cs CurrentSlotter) *hotStateSaver {
	return &hotStateSaver{
		snapshotInterval: DefaultSnapshotInterval,
		db:               d,
		fc:               fc,
		cs:               cs,
	}
}

// This tracks the config in the event of long non-finality,
// how often does the node save hot states to db? what are
// the saved hot states in db?... etc
type hotStateSaver struct {
	m                PersistenceMode
	lock             sync.RWMutex
	snapshotInterval types.Slot
	savedRoots       [][32]byte
	db               db.NoHeadAccessDatabase
	fc               FinalizedCheckpointer
	cs               CurrentSlotter
}

var _ Saver = &hotStateSaver{}

// enable/disable hot state saving m depending on
// whether the size of the gap between finalized and current epochs
// is greater than the threshold.
func (s *hotStateSaver) refreshMode(ctx context.Context) (PersistenceMode, error) {
	current := slots.ToEpoch(s.cs.CurrentSlot())
	fcp := s.fc.FinalizedCheckpoint()
	if fcp == nil {
		return PersistenceModeMemory, errForkchoiceFinalizedNil
	}
	// don't allow underflow
	if fcp.Epoch > current {
		return PersistenceModeMemory, errCurrentEpochBehindFinalized
	}
	if current-fcp.Epoch >= hotStateSaveThreshold {
		s.enableSnapshots()
		return PersistenceModeSnapshot, nil
	}

	return PersistenceModeMemory, s.disableSnapshots(ctx)
}

func (s *hotStateSaver) mode() PersistenceMode {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.m
}

// enableHotStateSaving enters the m that saves hot beacon state to the DB.
// This usually gets triggered when there's long duration since finality.
func (s *hotStateSaver) enableSnapshots() {
	if s.mode() == PersistenceModeSnapshot {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.m = PersistenceModeSnapshot
	log.WithFields(logrus.Fields{
		"mode":             s.m.String(),
		"slotsPerSnapshot": s.snapshotInterval,
	}).Warn("Enabling state cache db snapshots")
}

// disableHotStateSaving exits the m that saves beacon state to DB for the hot states.
// This usually gets triggered once there's finality after long duration since finality.
func (s *hotStateSaver) disableSnapshots(ctx context.Context) error {
	if s.mode() == PersistenceModeMemory {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	log.WithFields(logrus.Fields{
		"mode":             PersistenceModeMemory.String(),
		"slotsPerSnapshot": s.snapshotInterval,
	}).Warn("Disabling state cache db snapshots and removing saved snapshots")

	// we have a recent-enough finalized state, so time to clean up the state cache snapshots
	if err := s.db.DeleteStates(ctx, s.savedRoots); err != nil {
		return err
	}
	s.savedRoots = nil
	s.m = PersistenceModeMemory

	return nil
}

func shouldSave(m PersistenceMode, interval types.Slot, st state.BeaconState) bool {
	if m != PersistenceModeSnapshot {
		// only write full states to the db when in snapshot mode
		return false
	}
	// divide by zero guard
	if interval == 0 {
		return false
	}
	// only saving every s.duration slots - typically every 128 slots
	// checking this first avoids holding the lock if we aren't on a slot that should be saved
	if st.Slot().ModSlot(interval) != 0 {
		return false
	}
	return true
}

func (s *hotStateSaver) Save(ctx context.Context, blockRoot [32]byte, st state.BeaconState) error {
	mode, err := s.refreshMode(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to make hot state saving decision")
	}
	err = s.db.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: st.Slot(), Root: blockRoot[:]})
	if err != nil {
		return err
	}
	if !shouldSave(mode, s.snapshotInterval, st) {
		return nil
	}

	// we need the update to savedRoots to be in the same critical section as db.SaveState
	// because in Preserve we need the state bucket to be consistent with the list of saved roots
	// so that we can safely confirm the state is present and remove it from the root cleanup list.
	s.lock.Lock()
	defer s.lock.Unlock()
	s.savedRoots = append(s.savedRoots, blockRoot)
	log.WithFields(logrus.Fields{
		"slot":               st.Slot(),
		"totalStatesWritten": len(s.savedRoots),
	}).Info("Saving hot state to DB")
	return s.db.SaveState(ctx, st, blockRoot)
}

// Preserve ensures that the given state is permanently saved in the db. If the state already exists
// and the state saver is in snapshot mode, the block root will be removed from the list of roots to
// clean up when exiting snapshot mode to ensure it won't be deleted in the cleanup procedure.
func (s *hotStateSaver) Preserve(ctx context.Context, root [32]byte, st state.BeaconState) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	exists := s.db.HasState(ctx, root)
	if !exists {
		if err := s.db.SaveState(ctx, st, root); err != nil {
			return err
		}
	}
	// the state exists, and we aren't in snapshot mode, so we shouldn't have to do anything
	if s.m != PersistenceModeSnapshot {
		return nil
	}

	// slice the preserved root out of the list of states to delete once the node exists snapshot mode.
	for i := 0; i < len(s.savedRoots); i++ {
		if s.savedRoots[i] == root {
			s.savedRoots = append(s.savedRoots[:i], s.savedRoots[i+1:]...)
			return nil
		}
	}
	return nil
}
