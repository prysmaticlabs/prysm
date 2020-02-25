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

	if state.Slot()%s.slotsPerArchivePoint != 0 {
		return errNonArchivedPoint
	}

	archivedPointIndex := state.Slot() / s.slotsPerArchivePoint
	if err := s.beaconDB.SaveArchivedPointState(ctx, state, archivedPointIndex); err != nil {
		return err
	}
	if err := s.beaconDB.SaveArchivedPointRoot(ctx, blockRoot, archivedPointIndex); err != nil {
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
		archivedPoint := summary.Slot / s.slotsPerArchivePoint
		s, err := s.loadColdStateByArchivalPoint(ctx, archivedPoint)
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
func (s *State) loadColdStateByArchivalPoint(ctx context.Context, archivedPoint uint64) (*state.BeaconState, error) {
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
	lowArchivedPointIdx := slot / s.slotsPerArchivePoint
	highArchivedPointIdx := lowArchivedPointIdx + 1

	// Acquire the read lock so the split can't change while this is happening.
	lowArchivedPointState, err := s.loadArchivedPointByIndex(ctx, lowArchivedPointIdx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get low archived point by index")
	}
	if lowArchivedPointState == nil {
		return nil, errUnknownArchivedState
	}

	// If the slot of the high archive point lies outside of the freezer cut off, use the split state
	// as the upper archive point.
	var highArchivedPointRoot [32]byte
	highArchivedPointSlot := highArchivedPointIdx * s.slotsPerArchivePoint
	if highArchivedPointSlot >= s.splitInfo.slot {
		highArchivedPointRoot = s.splitInfo.root
		highArchivedPointSlot = s.splitInfo.slot
	} else {
		if _, err := s.loadArchivedPointByIndex(ctx, highArchivedPointSlot); err != nil {
			return nil, errors.Wrap(err, "could not get high archived point by index")
		}
		highArchivedPointRoot = s.beaconDB.ArchivedPointRoot(ctx, highArchivedPointIdx)
		slot, err := s.loadColdStateSlot(ctx, highArchivedPointRoot)
		if err != nil {
			return nil, errors.Wrap(err, "could not get high archived point slot")
		}
		highArchivedPointSlot = slot
	}

	replayBlks, err := s.LoadBlocks(ctx, lowArchivedPointState.Slot()+1, highArchivedPointSlot, highArchivedPointRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks to replay for cold intermediate state with slot")
	}

	return s.ReplayBlocks(ctx, lowArchivedPointState, replayBlks, slot)
}

// Given the archive index, this returns the archived cold state in the DB.
// If the archived state does not exist in the state, it'll compute it and save it.
func (s *State) loadArchivedPointByIndex(ctx context.Context, archiveIndex uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadArchivedPointByIndex")
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

// Given the block root, this returns the slot of the block root using cold state summary look up in DB.
// If cold state summary in DB is empty, this will save to the DB.
func (s *State) loadColdStateSlot(ctx context.Context, blockRoot [32]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdStateSlot")
	defer span.End()

	if s.beaconDB.HasColdStateSummary(ctx, blockRoot) {
		summary, err := s.beaconDB.ColdStateSummary(ctx, blockRoot)
		if err != nil {
			return 0, nil
		}
		if summary == nil {
			return 0, errUnknownColdSummary
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
	if err := s.beaconDB.SaveColdStateSummary(ctx, blockRoot, &pb.ColdStateSummary{Slot: b.Block.Slot}); err != nil {
		return 0, err
	}
	return b.Block.Slot, nil
}
