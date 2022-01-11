package v3

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/state-native/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
//nolint:deadcode
func computeFieldRoots(ctx context.Context, state *ethpb.BeaconStateBellatrix) ([][]byte, error) {
	if features.Get().EnableSSZCache {
		return stateutil.CachedHasher.ComputeFieldRootsWithHasherBellatrix(ctx, state)
	}
	return stateutil.NocachedHasher.ComputeFieldRootsWithHasherBellatrix(ctx, state)
}
