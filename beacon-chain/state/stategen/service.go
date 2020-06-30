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
	epochStateCache         *epochStateCache
}

// This tracks the split point. The point where slot and the block root of
// cold and hot sections of the DB splits.
type splitSlotAndRoot struct {
	slot uint64
	root [32]byte
}

// New returns a new state management object.
func New(ctx context.Context, db db.NoHeadAccessDatabase, stateSummaryCache *cache.StateSummaryCache) (*State, error) {
	f, err := db.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	fRoot := bytesutil.ToBytes32(f.Root)
	fState, err := db.State(ctx, fRoot)
	if err != nil {
		return nil, err
	}
	return &State{
		beaconDB:                db,
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
		hotStateCache:           cache.NewHotStateCache(),
		splitInfo:               &splitSlotAndRoot{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint:   params.BeaconConfig().SlotsPerArchivedPoint,
		stateSummaryCache:       stateSummaryCache,
		epochStateCache: newEpochStateCache(fRoot, fState),
	}, nil
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

	if featureconfig.Get().SkipRegenHistoricalStates {
		// If a node doesn't want to regen historical states, the node would
		// start from last finalized check point.
		cp, err := s.beaconDB.FinalizedCheckpoint(ctx)
		if err != nil {
			return nil, err
		}
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
