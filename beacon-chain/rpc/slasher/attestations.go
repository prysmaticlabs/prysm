package slasher

import (
	"context"

	slashpb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsSlashableAttestation returns an attester slashing if an input
// attestation is found to be slashable.
func (s *Server) IsSlashableAttestation(
	ctx context.Context, req *ethpb.IndexedAttestation,
) (*slashpb.AttesterSlashingResponse, error) {
	attesterSlashings, err := s.SlashingChecker.IsSlashableAttestation(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if attestation is slashable: %v", err)
	}
	if len(attesterSlashings) > 0 {
		return &slashpb.AttesterSlashingResponse{
			AttesterSlashing: attesterSlashings[0],
		}, nil
	}
	return &slashpb.AttesterSlashingResponse{}, nil
}

// HighestAttestations returns the highest source and target epochs attested for
// validator indices that have been observed by slasher.
func (s *Server) HighestAttestations(
	ctx context.Context, req *slashpb.HighestAttestationRequest,
) (*slashpb.HighestAttestationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}
