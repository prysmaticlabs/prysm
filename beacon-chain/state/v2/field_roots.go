package v2

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
)

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func computeFieldRoots(ctx context.Context, state *BeaconState) ([][]byte, error) {
	if features.Get().EnableSSZCache {
		return stateutil.CachedHasher.ComputeFieldRootsWithHasherAltair(ctx, state)
	}
	return stateutil.NocachedHasher.ComputeFieldRootsWithHasherAltair(ctx, state)
}
