package stategen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"go.opencensus.io/trace"
)

// This loads the cold state by block root, it chooses whether to load from archive point (faster) or
// somewhere between archive points (slower) since it requires replaying blocks. This is more efficient
// than load cold state by slot.
func (s *State) loadColdStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdStateByRoot")
	defer span.End()

	summary, err := s.beaconDB.StateSummary(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, errUnknownStateSummary
	}

	// Use the archived point state if the slot lies on top of the archived point.
	if summary.Slot%s.slotsPerArchivePoint == 0 {
		archivedPoint := summary.Slot / s.slotsPerArchivePoint
		s, err := s.loadColdStateByArchivedPoint(ctx, archivedPoint)
		if err != nil {
			return nil, errors.Wrap(err, "could not get cold state using archived index")
		}
		if s == nil {
			return nil, errUnknownArchivedState
		}
		return s, nil
	}

	return s.loadColdIntermediateStateWithRoot(ctx, summary.Slot, blockRoot)
}

// This loads the cold state for the input archived point.
func (s *State) loadColdStateByArchivedPoint(ctx context.Context, archivedPoint uint64) (*state.BeaconState, error) {
	return s.beaconDB.ArchivedPointState(ctx, archivedPoint)
}

// This loads a cold state by slot and block root which lies between the archive point.
// This is a faster implementation given the block root is provided.
func (s *State) loadColdIntermediateStateWithRoot(ctx context.Context, slot uint64, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdIntermediateStateWithRoot")
	defer span.End()

	// Load the archive point for lower side of the intermediate state.
	lowArchivePointIdx := slot / s.slotsPerArchivePoint

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivePointState, err := s.loadArchivedPointByIndex(ctx, lowArchivePointIdx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get lower bound archived state using index")
	}
	if lowArchivePointState == nil {
		return nil, errUnknownArchivedState
	}

	replayBlks, err := s.LoadBlocks(ctx, lowArchivePointState.Slot()+1, slot, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get load blocks for cold state using slot")
	}

	return s.ReplayBlocks(ctx, lowArchivePointState, replayBlks, slot)
}
