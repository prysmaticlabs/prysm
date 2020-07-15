package stategen

import (
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
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

	slot := state.Slot()
	if slot%s.slotsPerArchivedPoint != 0 {
		return nil
	}

	if err := s.beaconDB.SaveState(ctx, state, blockRoot); err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"slot":      slot,
		"blockRoot": hex.EncodeToString(bytesutil.Trunc(blockRoot[:]))}).Info("Saved full state on archived point")

	return nil
}

// This loads the cold state by block root.
func (s *State) loadColdStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdStateByRoot")
	defer span.End()

	summary, err := s.stateSummary(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state summary")
	}

	return s.loadColdStateBySlot(ctx, summary.Slot)
}

// This loads a cold state by slot.
func (s *State) loadColdStateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdStateBySlot")
	defer span.End()

	if slot == 0 {
		return s.beaconDB.GenesisState(ctx)
	}

	archivedState, err := s.archivedState(ctx, slot)
	if err != nil {
		return nil, err
	}
	if archivedState == nil {
		archivedState, err = s.beaconDB.GenesisState(ctx)
		if err != nil {
			return nil, err
		}
	}

	return s.processStateUpTo(ctx, archivedState, slot)
}
