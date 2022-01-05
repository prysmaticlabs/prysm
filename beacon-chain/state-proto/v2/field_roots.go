package v2

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func computeFieldRoots(ctx context.Context, state *BeaconState) ([][]byte, error) {
	protoState, ok := state.toProtoNoLock().(*eth.BeaconStateAltair)
	if !ok {
		return nil, errors.New("could not convert beacon state to proto")
	}
	if features.Get().EnableSSZCache {
		return stateutil.CachedHasher.ComputeFieldRootsWithHasherAltair(ctx, protoState)
	}
	return stateutil.NocachedHasher.ComputeFieldRootsWithHasherAltair(ctx, protoState)
}
