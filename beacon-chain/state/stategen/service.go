// Package stategen defines functions to regenerate beacon chain states
// by replaying blocks from a stored state checkpoint, useful for
// optimization and reducing a beacon node's resource consumption.
package stategen

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	slotsPerArchivedPoint   uint64
	epochBoundarySlotToRoot map[uint64][32]byte
	epochBoundaryLock       sync.RWMutex
	hotStateCache           *cache.HotStateCache
	splitInfo               *splitSlotAndRoot
	stateSummaryCache       *cache.StateSummaryCache
	finalized               *finalized
}

// This tracks the split point. The point where slot and the block root of
// cold and hot sections of the DB splits.
type splitSlotAndRoot struct {
	slot uint64
	root [32]byte
}

// This tracks the finalized state in memory. This is used to replay back hot state to a certain slot.
type finalized struct {
	state *state.BeaconState
	lock  sync.RWMutex
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase, stateSummaryCache *cache.StateSummaryCache) *State {
	f, err := db.FinalizedCheckpoint(context.Background())
	if err != nil {
		log.Error("could not get state")
	}
	fState, err := db.State(context.Background(), bytesutil.ToBytes32(f.Root))
	if err != nil {
		log.Error("could not get state")
	}
	return &State{
		beaconDB:                db,
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
		hotStateCache:           cache.NewHotStateCache(),
		splitInfo:               &splitSlotAndRoot{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint:   params.BeaconConfig().SlotsPerArchivedPoint,
		stateSummaryCache:       stateSummaryCache,
		finalized: &finalized{state: fState},
	}
}

// Resume resumes a new state management object from previously saved finalized check point in DB.
func (s *State) Resume(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.Resume")
	defer span.End()

	lastArchivedRoot := s.beaconDB.LastArchivedIndexRoot(ctx)
	lastArchivedState, err := s.beaconDB.State(ctx, lastArchivedRoot)
	if err != nil {
		return nil, err
	}

	cp, err := s.beaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	if featureconfig.Get().SkipRegenHistoricalStates {
		// If a node doesn't want to regen historical states, the node would
		// start from last finalized check point.
		lastArchivedState, err = s.beaconDB.State(ctx, bytesutil.ToBytes32(cp.Root))
		if err != nil {
			return nil, err
		}
		lastArchivedRoot = bytesutil.ToBytes32(cp.Root)
	}

	// Resume as genesis state if there's no last archived state.
	if lastArchivedState == nil {
		return s.beaconDB.GenesisState(ctx)
	}

	s.splitInfo = &splitSlotAndRoot{slot: lastArchivedState.Slot(), root: lastArchivedRoot}

	fState, err := s.beaconDB.State(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return nil, err
	}
	s.finalized.lock.Lock()
	s.finalized.state = fState
	s.finalized.lock.Unlock()

	return lastArchivedState, nil
}

// This verifies the archive point frequency is valid. It checks the interval
// is a divisor of the number of slots per epoch. This ensures we have at least one
// archive point within range of our state root history when iterating
// backwards. It also ensures the archive points align with hot state summaries
// which makes it quicker to migrate hot to cold.
func verifySlotsPerArchivePoint(slotsPerArchivePoint uint64) bool {
	return slotsPerArchivePoint > 0 &&
		slotsPerArchivePoint%params.BeaconConfig().SlotsPerEpoch == 0
}
