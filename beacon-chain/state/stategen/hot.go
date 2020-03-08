package stategen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

// This loads a post finalized beacon state from the hot section of the DB. If necessary it will
// replay blocks starting from the nearest epoch boundary. It returns the beacon state that
// corresponds to the input block root.
func (s *State) loadHotStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadHotStateByRoot")
	defer span.End()

	// Load the hot state cache.
	cachedState := s.hotStateCache.Get(blockRoot)
	if cachedState != nil {
		return cachedState, nil
	}

	summary, err := s.beaconDB.StateSummary(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, errUnknownStateSummary
	}
	boundaryState, err := s.beaconDB.State(ctx, bytesutil.ToBytes32(summary.BoundaryRoot))
	if err != nil {
		return nil, err
	}
	if boundaryState == nil {
		return nil, errUnknownBoundaryState
	}

	// Don't need to replay the blocks if we're already on an epoch boundary,
	// the target slot is the same as the state slot.
	var hotState *state.BeaconState
	targetSlot := summary.Slot
	if targetSlot == boundaryState.Slot() {
		hotState = boundaryState
	} else {
		blks, err := s.LoadBlocks(ctx, boundaryState.Slot()+1, targetSlot, bytesutil.ToBytes32(summary.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not load blocks for hot state using root")
		}
		hotState, err = s.ReplayBlocks(ctx, boundaryState, blks, targetSlot)
		if err != nil {
			return nil, errors.Wrap(err, "could not replay blocks for hot state using root")
		}
	}

	// Save the copied state because the reference also returned in the end.
	s.hotStateCache.Put(blockRoot, hotState.Copy())

	return hotState, nil
}
