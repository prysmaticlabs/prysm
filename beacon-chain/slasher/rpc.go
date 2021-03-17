package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// IsSlashableBlock comapres the given block to the slasher database and returns any double block
// proposals that are considered slashable.
func (s *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashing, error) {
	dataRoot, err := block.Header.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	signedBlockWrapper := &slashertypes.SignedBlockHeaderWrapper{
		SignedBeaconBlockHeader: block,
		SigningRoot:             dataRoot,
	}
	doubleProposals, err := s.saveSafeProposals(ctx, []*slashertypes.SignedBlockHeaderWrapper{signedBlockWrapper})
	if err != nil {
		return nil, err
	}
	if len(doubleProposals) == 0 {
		return nil, nil
	}
	proposerSlashing := &ethpb.ProposerSlashing{
		Header_1: doubleProposals[0].PrevBeaconBlockWrapper.SignedBeaconBlockHeader,
		Header_2: doubleProposals[0].BeaconBlockWrapper.SignedBeaconBlockHeader,
	}
	return proposerSlashing, nil
}
