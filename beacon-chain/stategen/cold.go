package stategen

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
)

// This saves a pre finalized beacon state in the cold section of the DB. The returns an error
// and not store anything if the state does not lie on an archive point boundary.
func (s *State) saveColdState(ctx context.Context, blockRoot [32]byte, state *state.BeaconState) error {
	if state.Slot()%s.slotsPerArchivePoint != 0 {
		return errors.New("unable to store non archive point state in cold")
	}

	if err := s.beaconDB.SaveState(ctx, state, blockRoot); err != nil {
		return err
	}

	archivePointIndex := state.Slot() / s.slotsPerArchivePoint
	if err := s.beaconDB.SaveArchivePoint(ctx, blockRoot, archivePointIndex); err != nil {
		return err
	}
	archivePointSaved.Inc()

	log.WithFields(logrus.Fields{
		"slot":      state.Slot(),
		"blockRoot": hex.EncodeToString(bytesutil.Trunc(blockRoot[:]))}).Info("Saved full state on archive point")

	return nil
}

// This loads the cold state by block root, it chooses whether to load from archive point (faster) or
// somewhere between archive points (slower) since it requires replaying blocks.
func (s *State) loadColdStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	summary, err := s.beaconDB.ColdStateSummary(ctx, blockRoot)
	if err != nil {
		return nil, err
	}

	if summary.Slot%s.slotsPerArchivePoint == 0 {
		archivePoint := summary.Slot / s.slotsPerArchivePoint
		s, err := s.loadColdStateByArchivalPoint(ctx, archivePoint)
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	return s.loadColdIntermediateStateWithRoot(ctx, summary.Slot, blockRoot)
}

// This loads the cold state for the certain archive point.
func (s *State) loadColdStateByArchivalPoint(ctx context.Context, archivePoint uint64) (*state.BeaconState, error) {
	root := s.beaconDB.ArchivePoint(ctx, archivePoint)
	return s.beaconDB.State(ctx, root)
}

// This loads a cold state by slot and block root which lies between the archive point.
// This is a faster implementation with block root provided.
func (s *State) loadColdIntermediateStateWithRoot(ctx context.Context, slot uint64, blockRoot [32]byte) (*state.BeaconState, error) {
	// Load the archive point for lower side of the intermediate state.
	lowArchivePointIdx := slot / s.slotsPerArchivePoint

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivePointState, err := s.loadArchivePointByIndex(ctx, lowArchivePointIdx)
	if err != nil {
		return nil, err
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
	// Load the archive point for lower and high side of the intermediate state.
	lowArchivePointIdx := slot / s.slotsPerArchivePoint
	highArchivePointIdx := lowArchivePointIdx + 1

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivePointState, err := s.loadArchivePointByIndex(ctx, lowArchivePointIdx)
	if err != nil {
		return nil, err
	}

	// If the slot of the high archive point lies outside of the freezer cut off, use the split state
	// as the upper archive point.
	var highArchivePointRoot [32]byte
	highArchivePointSlot := highArchivePointIdx * s.slotsPerArchivePoint
	if highArchivePointSlot >= s.splitInfo.slot {
		highArchivePointRoot = s.splitInfo.root
		highArchivePointSlot = s.splitInfo.slot
	} else {
		highArchivePointRoot = s.beaconDB.ArchivePoint(ctx, highArchivePointIdx)
	}

	replayBlks, err := s.LoadBlocks(ctx, lowArchivePointState.Slot()+1, highArchivePointSlot, highArchivePointRoot)
	if err != nil {
		return nil, err
	}

	return s.ReplayBlocks(ctx, lowArchivePointState, replayBlks, slot)
}

// Given the archive index, this returns the state in the DB.
func (s *State) loadArchivePointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	blockRoot := s.beaconDB.ArchivePoint(ctx, archiveIndex)
	return s.beaconDB.State(ctx, blockRoot)
}
