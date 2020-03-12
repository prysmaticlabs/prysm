package stategen

import (
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// This saves a pre finalized beacon state in the cold section of the DB. The returns an error
// and not store anything if the state does not lie on an archive point boundary.
func (s *State) saveColdState(ctx context.Context, blockRoot [32]byte, state *state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.saveColdState")
	defer span.End()

	if state.Slot()%s.slotsPerArchivedPoint != 0 {
		return errSlotNonArchivedPoint
	}

	archivedPointIndex := state.Slot() / s.slotsPerArchivedPoint
	if err := s.beaconDB.SaveArchivedPointState(ctx, state, archivedPointIndex); err != nil {
		return err
	}
	if err := s.beaconDB.SaveArchivedPointRoot(ctx, blockRoot, archivedPointIndex); err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"slot":      state.Slot(),
		"blockRoot": hex.EncodeToString(bytesutil.Trunc(blockRoot[:]))}).Info("Saved full state on archived point")

	return nil
}

// Given the archive index, this returns the archived cold state in the DB.
// If the archived state does not exist in the state, it'll compute it and save it.
func (s *State) archivedPointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadArchivedPointByIndex")
	defer span.End()
	if s.beaconDB.HasArchivedPoint(ctx, archiveIndex) {
		return s.beaconDB.ArchivedPointState(ctx, archiveIndex)
	}

	// If for certain reasons, archived point does not exist in DB,
	// a node should regenerate it and save it.
	return s.recoverArchivedPointByIndex(ctx, archiveIndex)
}

// This recovers an archived point by index. For certain reasons (ex. user toggles feature flag),
// an archived point may not be present in the DB. This regenerates the archived point state via
// playback and saves the archived root/state to the DB.
func (s *State) recoverArchivedPointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.recoverArchivedPointByIndex")
	defer span.End()

	archivedSlot := archiveIndex * s.slotsPerArchivedPoint
	archivedState, err := s.ComputeStateUpToSlot(ctx, archivedSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state up to archived index slot")
	}
	if archivedState == nil {
		return nil, errUnknownArchivedState
	}
	lastRoot, _, err := s.lastSavedBlock(ctx, archivedSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get last valid block up to archived index slot")
	}

	if err := s.beaconDB.SaveArchivedPointRoot(ctx, lastRoot, archiveIndex); err != nil {
		return nil, err
	}
	if err := s.beaconDB.SaveArchivedPointState(ctx, archivedState, archiveIndex); err != nil {
		return nil, err
	}

	return archivedState, nil
}

// Given the block root, this returns the slot of the block root using state summary look up in DB.
// If state summary does not exist in DB, it will recover the state summary and save it to the DB.
// This is used to cover corner cases where users toggle new state service's feature flag.
func (s *State) blockRootSlot(ctx context.Context, blockRoot [32]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.blockRootSlot")
	defer span.End()

	if s.beaconDB.HasStateSummary(ctx, blockRoot) {
		summary, err := s.beaconDB.StateSummary(ctx, blockRoot)
		if err != nil {
			return 0, nil
		}
		if summary == nil {
			return 0, errUnknownStateSummary
		}
		return summary.Slot, nil
	}

	// Retry with DB using block bucket.
	b, err := s.beaconDB.Block(ctx, blockRoot)
	if err != nil {
		return 0, err
	}
	if b == nil || b.Block == nil {
		return 0, errUnknownBlock
	}
	if err := s.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: blockRoot[:], Slot: b.Block.Slot}); err != nil {
		return 0, errors.Wrap(err, "could not save state summary")
	}

	return b.Block.Slot, nil
}
