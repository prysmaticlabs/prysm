package stategen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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

	// this is the only method that puts states in the cache, so if the state is in the cache
	// the state has already been processed by this method and there's no need to run again.
	if s.hotStateCache.has(blockRoot) {
		return nil
	}

	if err := s.saver.Save(ctx, blockRoot, st); err != nil {
		return err
	}

	// Only on an epoch boundary slot, save epoch boundary state in epoch boundary root state cache.
	if slots.IsEpochStart(st.Slot()) {
		if err := s.epochBoundaryStateCache.put(blockRoot, st); err != nil {
			return err
		}
	}

	// Store the copied state in the hot state cache.
	s.hotStateCache.put(blockRoot, st)

	return nil
}
