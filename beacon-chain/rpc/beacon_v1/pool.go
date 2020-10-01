package beacon_v1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func ListPoolAttestations(ctx context.Context, req *ethpb.AttestationsPoolRequest) (*ethpb.AttestationsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

func SubmitAttestation(ctx context.Context, req *ethpb.Attestation) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

func ListPoolAttesterSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.AttesterSlashingsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

func SubmitAttesterSlashing(ctx context.Context, req *ethpb.AttesterSlashing) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

func ListPoolProposerSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashingPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

func SubmitProposerSlashing(ctx context.Context, req *ethpb.ProposerSlashing) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

func ListPoolVoluntaryExits(ctx context.Context, req *ptypes.Empty) (*ethpb.VoluntaryExitsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

func SubmitVoluntaryExit(ctx context.Context, req *ethpb.SignedVoluntaryExit) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}
