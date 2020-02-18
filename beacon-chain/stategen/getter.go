package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// StateByRoot retrieves the state from DB using input root.
// It retrieves state from the cold section if the cold state
// summary exists in DB by default.
func (s *State) StateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	if s.beaconDB.HasColdStateSummary(ctx, blockRoot) {
		return s.loadColdState(ctx, blockRoot)
	}

	return s.loadHotState(ctx, blockRoot)
}
