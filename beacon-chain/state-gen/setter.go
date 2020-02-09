package state_gen

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// SaveState saves the state in the DB.
// It knows which cold and hot state section the input state belongs to.
func SaveState(ctx context.Context, state *state.BeaconState) error {
	return nil
}
