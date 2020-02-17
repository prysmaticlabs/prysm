package stategen

import (
	"context"

	"github.com/pkg/errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB db.NoHeadAccessDatabase
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB: db,
	}
}

// GenerateState generates a state from the provided pre-state till the provided block.
func (s *State) GenerateState(ctx context.Context, preState *stateTrie.BeaconState, endBlock *ethpb.SignedBeaconBlock) (*stateTrie.BeaconState, error) {
	preState = preState.Copy()
	root, err := ssz.HashTreeRoot(endBlock.Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the root of the block")
	}
	blocks, err := s.loadBlocks(ctx, preState.Slot()+1, endBlock.Block.Slot, root)
	if err != nil {
		return nil, errors.Wrap(err, "could not load the required blocks")
	}
	postState, err := s.replayBlocks(ctx, preState, blocks, endBlock.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay the blocks to generate the resultant state")
	}
	return postState, nil
}
