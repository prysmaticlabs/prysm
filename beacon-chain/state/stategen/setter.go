package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

// SaveState saves the state in the DB.
// It knows which cold and hot state section the input state should belong to.
func (s *State) SaveState(ctx context.Context, root [32]byte, state *state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.SaveState")
	defer span.End()

	// The state belongs to the cold section if it's below the split slot threshold.
	if state.Slot() < s.splitInfo.slot {
		return s.saveColdState(ctx, root, state)
	}

	return s.saveHotState(ctx, root, state)
}

// DeleteHotStateInCache deletes the hot state entry from the cache.
func (s *State) DeleteHotStateInCache(root [32]byte) {
	s.hotStateCache.Delete(root)
}

// ForceCheckpoint initiates a cold state save of the given state. This method does not update the
// "last archived state" but simply saves the specified state from the root argument into the DB.
func (s *State) ForceCheckpoint(ctx context.Context, root []byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.ForceCheckpoint")
	defer span.End()

	root32 := bytesutil.ToBytes32(root)
	fs, err := s.loadHotStateByRoot(ctx, root32)
	if err != nil {
		return err
	}
	if err := s.beaconDB.SaveState(ctx, fs, root32); err != nil {
		return err
	}

	return nil
}
