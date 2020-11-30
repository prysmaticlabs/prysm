package stategen

import (
	"context"
	"math"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// SaveState saves the state in the cache and/or DB.
func (s *State) SaveState(ctx context.Context, root [32]byte, state *state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.SaveState")
	defer span.End()

	return s.saveStateByRoot(ctx, root, state)
}

// ForceCheckpoint initiates a cold state save of the given state. This method does not update the
// "last archived state" but simply saves the specified state from the root argument into the DB.
func (s *State) ForceCheckpoint(ctx context.Context, root []byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.ForceCheckpoint")
	defer span.End()

	root32 := bytesutil.ToBytes32(root)
	// Before the first finalized check point, the finalized root is zero hash.
	// Return early if there hasn't been a finalized check point.
	if root32 == params.BeaconConfig().ZeroHash {
		return nil
	}

	fs, err := s.loadStateByRoot(ctx, root32)
	if err != nil {
		return err
	}

	return s.beaconDB.SaveState(ctx, fs, root32)
}

// SaveStateSummariesToDB saves cached state summaries to DB.
func (s *State) SaveStateSummariesToDB(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.SaveStateSummariesToDB")
	defer span.End()
	if err := s.beaconDB.SaveStateSummaries(ctx, s.stateSummaryCache.GetAll()); err != nil {
		return err
	}
	s.stateSummaryCache.Clear()
	return nil
}

// SaveStateSummary saves the relevant state summary for a block and its corresponding state slot in the
// state summary cache.
func (s *State) SaveStateSummary(_ context.Context, blk *ethpb.SignedBeaconBlock, blockRoot [32]byte) {
	// Save State summary
	s.stateSummaryCache.Put(blockRoot, &pb.StateSummary{
		Slot: blk.Block.Slot,
		Root: blockRoot[:],
	})
}

// This saves a post beacon state. On the epoch boundary,
// it saves a full state. On an intermediate slot, it saves a back pointer to the
// nearest epoch boundary state.
func (s *State) saveStateByRoot(ctx context.Context, blockRoot [32]byte, state *state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.saveStateByRoot")
	defer span.End()

	// Duration can't be 0 to prevent panic for division.
	duration := uint64(math.Max(float64(s.saveHotStateDB.duration), 1))

	s.saveHotStateDB.lock.Lock()
	if s.saveHotStateDB.enabled && state.Slot()%duration == 0 {
		if err := s.beaconDB.SaveState(ctx, state, blockRoot); err != nil {
			s.saveHotStateDB.lock.Unlock()
			return err
		}
		s.saveHotStateDB.savedStateRoots = append(s.saveHotStateDB.savedStateRoots, blockRoot)

		log.WithFields(logrus.Fields{
			"slot":                   state.Slot(),
			"totalHotStateSavedInDB": len(s.saveHotStateDB.savedStateRoots),
		}).Info("Saving hot state to DB")
	}
	s.saveHotStateDB.lock.Unlock()

	// If the hot state is already in cache, one can be sure the state was processed and in the DB.
	if s.hotStateCache.Has(blockRoot) {
		return nil
	}

	// Only on an epoch boundary slot, saves epoch boundary state in epoch boundary root state cache.
	if helpers.IsEpochStart(state.Slot()) {
		if err := s.epochBoundaryStateCache.put(blockRoot, state); err != nil {
			return err
		}
	}

	// On an intermediate slots, save the hot state summary.
	s.stateSummaryCache.Put(blockRoot, &pb.StateSummary{
		Slot: state.Slot(),
		Root: blockRoot[:],
	})

	// Store the copied state in the hot state cache.
	s.hotStateCache.Put(blockRoot, state)

	return nil
}

// EnableSaveHotStateToDB enters the mode that saves hot beacon state to the DB.
// This usually gets triggered when there's long duration since finality.
func (s *State) EnableSaveHotStateToDB(_ context.Context) {
	s.saveHotStateDB.lock.Lock()
	defer s.saveHotStateDB.lock.Unlock()
	if s.saveHotStateDB.enabled {
		return
	}

	s.saveHotStateDB.enabled = true

	log.WithFields(logrus.Fields{
		"enabled":       s.saveHotStateDB.enabled,
		"slotsInterval": s.saveHotStateDB.duration,
	}).Warn("Entering mode to save hot states in DB")
}

// DisableSaveHotStateToDB exits the mode that saves beacon state to DB for the hot states.
// This usually gets triggered once there's finality after long duration since finality.
func (s *State) DisableSaveHotStateToDB(ctx context.Context) error {
	s.saveHotStateDB.lock.Lock()
	defer s.saveHotStateDB.lock.Unlock()
	if !s.saveHotStateDB.enabled {
		return nil
	}

	log.WithFields(logrus.Fields{
		"enabled":          s.saveHotStateDB.enabled,
		"deletedHotStates": len(s.saveHotStateDB.savedStateRoots),
	}).Warn("Exiting mode to save hot states in DB")

	// Delete previous saved states in DB as we are turning this mode off.
	s.saveHotStateDB.enabled = false
	if err := s.beaconDB.DeleteStates(ctx, s.saveHotStateDB.savedStateRoots); err != nil {
		return err
	}
	s.saveHotStateDB.savedStateRoots = nil

	return nil
}
