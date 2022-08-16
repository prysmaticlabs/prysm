package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsSlashableBlock returns a proposer slashing if an input
// signed beacon block header is found to be slashable.
func (s *Server) IsSlashableBlock(
	ctx context.Context, req *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashingResponse, error) {
	proposerSlashing, err := s.SlashingChecker.IsSlashableBlock(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if block is slashable: %v", err)
	}
	if proposerSlashing == nil {
		return &ethpb.ProposerSlashingResponse{
			ProposerSlashings: []*ethpb.ProposerSlashing{},
		}, nil
	}
	return &ethpb.ProposerSlashingResponse{
		ProposerSlashings: []*ethpb.ProposerSlashing{proposerSlashing},
	}, nil
}
