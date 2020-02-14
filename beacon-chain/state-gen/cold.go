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
	if state.Slot() % s.slotsPerArchivePoint != 0 {
		return errors.New("unable to store non archive point state in cold")
	}

	if err := s.beaconDB.SaveState(ctx, state, blockRoot); err != nil {
		return err
	}

	archivePointIndex := state.Slot() / s.slotsPerArchivePoint
	if err := s.beaconDB.SaveArchivePoint(ctx, blockRoot, archivePointIndex); err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"slot":      state.Slot(),
		"blockRoot": hex.EncodeToString(bytesutil.Trunc(blockRoot[:]))}).Debug("Saved full state on archive point")

	return nil
}

// This loads a cold state that lies between the archive point.
func (s *State) loadColdIntermediateState(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	// Load the archive point for lower and high side of the intermediate state.
	lowArchivePointIdx := slot / s.slotsPerArchivePoint
	highArchivePointIdx := lowArchivePointIdx + 1

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivePointState, err := s.loadArchivePointByIndex(ctx, lowArchivePointIdx)
	if err != nil {
		return nil, err
	}

	// If the high archive point is outside of the split, it uses the split state as the
	// high archive point.
	if highArchivePointIdx * s.slotsPerArchivePoint >= s.splitSlot {

	} else {

	}

	// Load the blocks from low archive point to input slot.
	s.beaconDB.Col

	return nil
}

// Given the archive index, this returns the state in the DB.
func (s *State) loadArchivePointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	blockRoot := s.beaconDB.ArchivePoint(ctx, archiveIndex)
	return s.beaconDB.State(ctx, blockRoot)
}
