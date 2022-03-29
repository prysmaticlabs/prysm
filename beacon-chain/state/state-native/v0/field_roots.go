package v0

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
)

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func computeFieldRoots(ctx context.Context, state *BeaconState) ([][]byte, error) {
	return stateutil.ComputeFieldRootsWithHasher(ctx, state)
}
