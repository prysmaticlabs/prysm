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
	coldStateSaved.Inc()

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

// This loads the cold state by deciding whether to load from archive point (faster) or
// somewhere between archive points (slower) since it requires replaying blocks.
func (s *State) loadColdState(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
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

	return s.loadColdIntermediateState(ctx, summary.Slot, blockRoot)
}

// This loads the cold state for the certain archive point.
func (s *State) loadColdStateByArchivalPoint(ctx context.Context, archivePoint uint64) (*state.BeaconState, error) {
	root := s.beaconDB.ArchivePoint(ctx, archivePoint)
	return s.beaconDB.State(ctx, root)
}

// This loads a cold state by slot and block root that lies between the archive point.
func (s *State) loadColdIntermediateState(ctx context.Context, slot uint64, blockRoot [32]byte) (*state.BeaconState, error) {
	// Load the archive point for lower and high side of the intermediate state.
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

// Given the archive index, this returns the state in the DB.
func (s *State) loadArchivePointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	blockRoot := s.beaconDB.ArchivePoint(ctx, archiveIndex)
	return s.beaconDB.State(ctx, blockRoot)
}
