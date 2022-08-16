package v2

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func computeFieldRoots(ctx context.Context, state *ethpb.BeaconStateAltair) ([][]byte, error) {
	return stateutil.ComputeFieldRootsWithHasherAltair(ctx, state)
}
