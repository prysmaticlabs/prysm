package state_gen

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// This replays the input blocks on the input state until the target slot is reached.
func replayBlocks(ctx context.Context, state *state.BeaconState, blocks []*ethpb.BeaconBlock, targetSlot uint64) (*state.BeaconState, error) {
	return nil, nil
}
