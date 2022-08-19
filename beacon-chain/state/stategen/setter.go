package stategen

import (
	"context"
	"math"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	errCurrentEpochBehindFinalized = errors.New("finalized epoch must always be before the current epoch")
	errForkchoiceFinalizedNil      = errors.New("forkchoice store finalized checkpoint is nil")
)
var hotStateSaveThreshold = types.Epoch(100)

// SaveState saves the state in the cache and/or DB.
func (s *State) SaveState(ctx context.Context, blockRoot [32]byte, st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.SaveState")
	defer span.End()

	if err := s.hotStateStatus.refresh(ctx); err != nil {
		return errors.Wrap(err, "stategen is unable to make hot state saving decision")
	}

	return s.saveStateByRoot(ctx, blockRoot, st)
}

// ForceCheckpoint initiates a cold state save of the given block root's state. This method does not update the
// "last archived state" but simply saves the specified state from the root argument into the DB.
//
// The name "Checkpoint" isn't referring to checkpoint in the sense of our consensus type, but checkpoint for our historical states.
func (s *State) ForceCheckpoint(ctx context.Context, blockRoot []byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.ForceCheckpoint")
	defer span.End()

	root32 := bytesutil.ToBytes32(blockRoot)
	// Before the first finalized checkpoint, the finalized root is zero hash.
	// Return early if there hasn't been a finalized checkpoint.
	if root32 == params.BeaconConfig().ZeroHash {
		return nil
	}

	fs, err := s.loadStateByRoot(ctx, root32)
	if err != nil {
		return err
	}

	return s.beaconDB.SaveState(ctx, fs, root32)
}

// This saves a post beacon state. On the epoch boundary,
// it saves a full state. On an intermediate slot, it saves a back pointer to the
// nearest epoch boundary state.
func (s *State) saveStateByRoot(ctx context.Context, blockRoot [32]byte, st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.saveStateByRoot")
	defer span.End()

	// Duration can't be 0 to prevent panic for division.
	duration := uint64(math.Max(float64(s.hotStateStatus.duration), 1))

	s.hotStateStatus.lock.Lock()
	if s.hotStateStatus.enabled && st.Slot().Mod(duration) == 0 {
		if err := s.beaconDB.SaveState(ctx, st, blockRoot); err != nil {
			s.hotStateStatus.lock.Unlock()
			return err
		}
		s.hotStateStatus.blockRootsOfSavedStates = append(s.hotStateStatus.blockRootsOfSavedStates, blockRoot)

		log.WithFields(logrus.Fields{
			"slot":                   st.Slot(),
			"totalHotStateSavedInDB": len(s.hotStateStatus.blockRootsOfSavedStates),
		}).Info("Saving hot state to DB")
	}
	s.hotStateStatus.lock.Unlock()

	// If the hot state is already in cache, one can be sure the state was processed and in the DB.
	if s.hotStateCache.has(blockRoot) {
		return nil
	}

	// Only on an epoch boundary slot, save epoch boundary state in epoch boundary root state cache.
	if slots.IsEpochStart(st.Slot()) {
		if err := s.epochBoundaryStateCache.put(blockRoot, st); err != nil {
			return err
		}
	}

	// On an intermediate slot, save state summary.
	if err := s.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: st.Slot(),
		Root: blockRoot[:],
	}); err != nil {
		return err
	}

	// Store the copied state in the hot state cache.
	s.hotStateCache.put(blockRoot, st)

	return nil
}