package slasher

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// IsSlashableBlock returns a proposer slashing if an input
// signed beacon block header is found to be slashable.
func (s *Server) IsSlashableBlock(
	ctx context.Context, req *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashing, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}
