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
