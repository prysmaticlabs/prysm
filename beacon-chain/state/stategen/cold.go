package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// This saves a pre finalized beacon state in the cold section of the DB. The returns an error
// and not store anything if the state does not lie on an archive point boundary.
func (s *State) saveColdState(ctx context.Context, blockRoot [32]byte, state *state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.saveColdState")
	defer span.End()

	if state.Slot()%s.slotsPerArchivePoint != 0 {
		return errNonArchivedPoint
	}

	archivePointIndex := state.Slot() / s.slotsPerArchivePoint
	if err := s.beaconDB.SaveArchivedPointState(ctx, state, archivePointIndex); err != nil {
		return err
	}
	if err := s.beaconDB.SaveArchivedPointRoot(ctx, blockRoot, archivePointIndex); err != nil {
		return err
	}
	archivePointSaved.Inc()

	log.WithFields(logrus.Fields{
		"slot":      state.Slot(),
		"blockRoot": hex.EncodeToString(bytesutil.Trunc(blockRoot[:]))}).Info("Saved full state on archive point")

	return nil
}

// This loads the cold state by block root, it chooses whether to load from archive point (faster) or
// somewhere between archive points (slower) since it requires replaying blocks. This is more efficient
// than load cold state by slot.
func (s *State) loadColdStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdStateByRoot")
	defer span.End()

	summary, err := s.beaconDB.ColdStateSummary(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, errUnknownColdSummary
	}

	// Use the archived point state if the slot lies on top of the archived point.
	if summary.Slot%s.slotsPerArchivePoint == 0 {
		archivePoint := summary.Slot / s.slotsPerArchivePoint
		s, err := s.loadColdStateByArchivalPoint(ctx, archivePoint)
		if err != nil {
			return nil, err
		}
		if s == nil {
			return nil, errUnknownArchivedState
		}
		return s, nil
	}

	return s.loadColdIntermediateStateWithRoot(ctx, summary.Slot, blockRoot)
}

// This loads the cold state for the input archived point.
func (s *State) loadColdStateByArchivalPoint(ctx context.Context, archivePoint uint64) (*state.BeaconState, error) {
	return s.beaconDB.ArchivedPointState(ctx, archivePoint)
}

// This loads a cold state by slot and block root which lies between the archive point.
// This is a faster implementation given the block root is provided.
func (s *State) loadColdIntermediateStateWithRoot(ctx context.Context, slot uint64, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdIntermediateStateWithRoot")
	defer span.End()

	// Load the archive point for lower side of the intermediate state.
	lowArchivePointIdx := slot / s.slotsPerArchivePoint

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivePointState, err := s.loadArchivePointByIndex(ctx, lowArchivePointIdx)
	if err != nil {
		return nil, err
	}
	if lowArchivePointState == nil {
		return nil, errUnknownArchivedState
	}

	replayBlks, err := s.LoadBlocks(ctx, lowArchivePointState.Slot()+1, slot, blockRoot)
	if err != nil {
		return nil, err
	}

	return s.ReplayBlocks(ctx, lowArchivePointState, replayBlks, slot)
}

// This loads a cold state by slot only where the slot lies between the archive point.
// This is a slower implementation given slot is the only argument. It require fetching
// all the blocks between the archival points.
func (s *State) loadColdIntermediateStateWithSlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdIntermediateStateWithSlot")
	defer span.End()

	// Load the archive point for lower and high side of the intermediate state.
	lowArchivePointIdx := slot / s.slotsPerArchivePoint
	highArchivePointIdx := lowArchivePointIdx + 1

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivePointState, err := s.loadArchivePointByIndex(ctx, lowArchivePointIdx)
	if err != nil {
		return nil, err
	}
	if lowArchivePointState == nil {
		return nil, errUnknownArchivedState
	}


	// If the slot of the high archive point lies outside of the freezer cut off, use the split state
	// as the upper archive point.
	var highArchivePointRoot [32]byte
	highArchivePointSlot := highArchivePointIdx * s.slotsPerArchivePoint
	if highArchivePointSlot >= s.splitInfo.slot {
		log.Info("Debugging: entering cold state case 1")
		highArchivePointRoot = s.splitInfo.root
		highArchivePointSlot = s.splitInfo.slot
	} else {
		log.Info("Debugging: entering cold state case 2")
		highArchivePointRoot = s.beaconDB.ArchivedPointRoot(ctx, highArchivePointIdx)
		summary, err := s.beaconDB.ColdStateSummary(ctx, highArchivePointRoot)
		if err != nil {
			return nil, err
		}
		if summary == nil {
			return nil, errUnknownColdSummary
		}
		highArchivePointSlot = summary.Slot
	}

	replayBlks, err := s.LoadBlocks(ctx, lowArchivePointState.Slot()+1, highArchivePointSlot, highArchivePointRoot)
	if err != nil {
		return nil, err
	}

	return s.ReplayBlocks(ctx, lowArchivePointState, replayBlks, slot)
}

// Given the archive index, this returns the archived cold state in the DB.
// If the archived state does not exist in the state, it'll compute it and save it.
func (s *State) loadArchivePointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadArchivePointByIndex")
	defer span.End()

	if s.beaconDB.HasArchivedPoint(ctx, archiveIndex) {
		return s.beaconDB.ArchivedPointState(ctx, archiveIndex)
	}

	archivedSlot := archiveIndex * s.slotsPerArchivePoint
	archivedState, err := s.ComputeStateUpToSlot(ctx, archivedSlot)
	if err != nil {
		return nil, err
	}
	if archivedState == nil {
		return nil, errUnknownArchivedState
	}
	lastRoot, _, err := s.getLastValidBlock(ctx, archivedSlot)
	if err != nil {
		return nil, err
	}

	if err := s.beaconDB.SaveArchivedPointRoot(ctx, lastRoot, archiveIndex); err != nil {
		return nil, err
	}
	if err := s.beaconDB.SaveArchivedPointState(ctx, archivedState, archiveIndex); err != nil {
		return nil, err
	}

	return archivedState, nil
}
