package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// IsSlashableBlock returns a proposer slashing if an input
// signed beacon block header is found to be slashable.
func (s *Server) IsSlashableBlock(
	ctx context.Context, req *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashing, error) {
	proposerSlashing, err := s.slasher.IsSlashableProposal(ctx, req)
	if err != nil {
		return nil, err
	}
	return proposerSlashing, nil
}
