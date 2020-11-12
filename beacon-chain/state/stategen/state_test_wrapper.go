package stategen

import (
	"context"
	"errors"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// stateTestWrapper is a wrapper for the real State struct. The wrapper is meant to be used in testing instead
// of the real StateBase implementation.
type stateTestWrapper struct {
	State
}

// newStateTestWrapper returns a new instance of StateTestWrapper.
func newStateTestWrapper(db db.NoHeadAccessDatabase, stateSummaryCache *cache.StateSummaryCache) (*stateTestWrapper, error) {
	state := New(db, stateSummaryCache)
	state.LoadBlocks = wrapLoadBlocks(state)
	return &stateTestWrapper{State: *state}, nil

}

// wrapLoadBlocks modifies the behavior of State.LoadBlocks, so that an error is not returned when an invalid
// range is provided. This simplifies test setup.
func wrapLoadBlocks(state *State) func(context.Context, uint64, uint64, [32]byte) ([]*eth.SignedBeaconBlock, error) {
	originalLoadBlocks := state.LoadBlocks
	return func(ctx context.Context, startSlot, endSlot uint64, endBlockRoot [32]byte) ([]*eth.SignedBeaconBlock, error) {
		blocks, err := originalLoadBlocks(ctx, startSlot, endSlot, endBlockRoot)
		if errors.Is(err, errInvalidRange) {
			return nil, nil
		}
		return blocks, err
	}
}
