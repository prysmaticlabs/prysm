// Package stategen defines functions to regenerate beacon chain states
// by replaying blocks from a stored state checkpoint, useful for
// optimization and reducing a beacon node's resource consumption.
package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB              db.NoHeadAccessDatabase
	slotsPerArchivedPoint uint64
	hotStateCache         *cache.HotStateCache
	finalizedInfo         *finalizedSlotRoot
	stateSummaryCache     *cache.StateSummaryCache
}

// This tracks the finalized point. It's also the point where slot and the block root of
// cold and hot sections of the DB splits.
type finalizedSlotRoot struct {
	slot uint64
	root [32]byte
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase, stateSummaryCache *cache.StateSummaryCache) *State {
	return &State{
		beaconDB:              db,
		hotStateCache:         cache.NewHotStateCache(),
		finalizedInfo:         &finalizedSlotRoot{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint: params.BeaconConfig().SlotsPerArchivedPoint,
		stateSummaryCache:     stateSummaryCache,
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

	// Resume as genesis state if there's no last archived state.
	if lastArchivedState == nil {
		return s.beaconDB.GenesisState(ctx)
	}

	s.finalizedInfo = &finalizedSlotRoot{slot: lastArchivedState.Slot(), root: lastArchivedRoot}

	return lastArchivedState, nil
}
