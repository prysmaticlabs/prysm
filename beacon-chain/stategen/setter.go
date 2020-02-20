package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// SaveState saves the state in the DB.
// It knows which cold and hot state section the input state should belong to.
func (s *State) SaveState(ctx context.Context, root [32]byte, state *state.BeaconState) error {
	// State belongs to the cold section if it's below the split slot threshold.
	if state.Slot() < s.splitInfo.slot {
		if err := s.saveColdState(ctx, root, state); err != nil {
			return err
		}
		return nil
	}

	if err := s.saveHotState(ctx, root, state); err != nil {
		return err
	}

	return nil
}
