package beacon_v1

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func ListPoolAttestations(ctx context.Context, req *ethpb.AttestationsPoolRequest) (*ethpb.AttestationsPoolResponse, error) {
	return &ethpb.AttestationsPoolResponse{}, nil
}

func SubmitAttestation(ctx context.Context, req *ethpb.Attestation) (*ptypes.Empty, error) {
	return &ptypes.Empty{}, nil
}

func ListPoolAttesterSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.AttesterSlashingsPoolResponse, error) {
	return &ethpb.AttesterSlashingsPoolResponse{}, nil
}

func SubmitAttesterSlashing(ctx context.Context, req *ethpb.AttesterSlashing) (*ptypes.Empty, error) {
	return &ptypes.Empty{}, nil
}

func ListPoolProposerSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashingPoolResponse, error) {
	return &ethpb.ProposerSlashingPoolResponse{}, nil
}

func SubmitProposerSlashing(ctx context.Context, req *ethpb.ProposerSlashing) (*ptypes.Empty, error) {
	return &ptypes.Empty{}, nil
}

func ListPoolVoluntaryExits(ctx context.Context, req *ptypes.Empty) (*ethpb.VoluntaryExitsPoolResponse, error) {
	return &ethpb.VoluntaryExitsPoolResponse{}, nil
}

func SubmitVoluntaryExit(ctx context.Context, req *ethpb.SignedVoluntaryExit) (*ptypes.Empty, error) {
	return &ptypes.Empty{}, nil
}
