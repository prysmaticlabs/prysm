package slasher

import (
	"context"

	slashpb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsSlashableBlock returns a proposer slashing if an input
// signed beacon block header is found to be slashable.
func (s *Server) IsSlashableBlock(
	ctx context.Context, req *ethpb.SignedBeaconBlockHeader,
) (*slashpb.ProposerSlashingResponse, error) {
	proposerSlashing, err := s.SlashingChecker.IsSlashableBlock(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if block is slashable: %v", err)
	}
	return &slashpb.ProposerSlashingResponse{
		ProposerSlashing: proposerSlashing,
	}, nil
}
