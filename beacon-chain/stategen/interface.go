package stategen

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// StateGenerator defines the functions to load and replay blocks and generate state.
type StateGenerator interface {
	LoadBlocks(ctx context.Context, startSlot uint64, endSlot uint64, endBlockRoot [32]byte) ([]*ethpb.SignedBeaconBlock, error)
	ReplayBlocks(ctx context.Context, state *state.BeaconState, signed []*ethpb.SignedBeaconBlock, targetSlot uint64) (*state.BeaconState, error)
}
