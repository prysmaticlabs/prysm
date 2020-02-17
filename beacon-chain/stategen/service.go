package stategen

import (
	"context"

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

func (s *State) GenerateState(ctx context.Context, preState *stateTrie.BeaconState, endBlock *ethpb.SignedBeaconBlock) (*stateTrie.BeaconState, error) {
	preState = preState.Copy()
	root, err := ssz.HashTreeRoot(endBlock)
	if err != nil {
		return nil, err
	}
	blocks, err := s.LoadBlocks(ctx, preState.Slot(), endBlock.Block.Slot, root)
	if err != nil {
		return nil, err
	}
	postState, err := s.ReplayBlocks(ctx, preState, blocks, endBlock.Block.Slot)
	if err != nil {
		return nil, err
	}
	return postState, nil
}
