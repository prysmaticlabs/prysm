package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitProposerSlashing receives a proposer slashing object via
// RPC and injects it into the beacon node's operations pool.
// Submission into this pool does not guarantee inclusion into a beacon block.
func (bs *Server) SubmitProposerSlashing(
	ctx context.Context,
	req *ethpb.ProposerSlashing,
) (*ethpb.SubmitSlashingResponse, error) {
	beaconState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head state: %v", err)
	}
	if err := bs.SlashingsPool.InsertProposerSlashing(beaconState, req); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not insert proposer slashing into pool: %v", err)
	}
	return &ethpb.SubmitSlashingResponse{
		SlashedIndices: []uint64{req.ProposerIndex},
	}, nil
}

// SubmitAttesterSlashing receives an attester slashing object via
// RPC and injects it into the beacon node's operations pool.
// Submission into this pool does not guarantee inclusion into a beacon block.
func (bs *Server) SubmitAttesterSlashing(
	ctx context.Context,
	req *ethpb.AttesterSlashing,
) (*ethpb.SubmitSlashingResponse, error) {
	beaconState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head state: %v", err)
	}
	if err := bs.SlashingsPool.InsertAttesterSlashing(beaconState, req); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not insert attester slashing into pool: %v", err)
	}
	slashedIndices := sliceutil.IntersectionUint64(req.Attestation_1.AttestingIndices, req.Attestation_2.AttestingIndices)
	return &ethpb.SubmitSlashingResponse{
		SlashedIndices: slashedIndices,
	}, nil
}
