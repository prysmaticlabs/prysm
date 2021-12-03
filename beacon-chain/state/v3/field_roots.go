package v3

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
//nolint:deadcode
func computeFieldRoots(ctx context.Context, state *ethpb.BeaconStateMerge) ([][]byte, error) {
	if features.Get().EnableSSZCache {
		return stateutil.CachedHasher.ComputeFieldRootsWithHasherMerge(ctx, state)
	}
	return stateutil.NocachedHasher.ComputeFieldRootsWithHasherMerge(ctx, state)
}
