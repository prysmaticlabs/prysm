package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"go.opencensus.io/trace"
)

// This loads the cold state by block root, it chooses whether to load from archive point (faster) or
// somewhere between archive points (slower) since it requires replaying blocks. This is more efficient
// than load cold state by slot.
func (s *State) loadColdStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadColdStateByRoot")
	defer span.End()

	summary, err := s.beaconDB.StateSummary(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, errUnknownStateSummary
	}

	// Use the archived point state if the slot lies on top of the archived point.
	if summary.Slot%s.slotsPerArchivePoint == 0 {
		archivedPoint := summary.Slot / s.slotsPerArchivePoint
		s, err := s.loadColdStateByArchivedPoint(ctx, archivedPoint)
		if err != nil {
			return nil, errors.Wrap(err, "could not get cold state using archived index")
		}
		if s == nil {
			return nil, errUnknownArchivedState
		}
		return s, nil
	}

	return s.loadColdIntermediateStateWithRoot(ctx, summary.Slot, blockRoot)
}
