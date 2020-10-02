package beacon_v1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

// ListPoolAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block.
func ListPoolAttestations(ctx context.Context, req *ethpb.AttestationsPoolRequest) (*ethpb.AttestationsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

// SubmitAttestation submits Attestation object to node. If attestation passes all validation
// constraints, node MUST publish attestation on appropriate subnet.
func SubmitAttestation(ctx context.Context, req *ethpb.Attestation) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// ListPoolAttesterSlashings retrieves attester slashings known by the node but
// not necessarily incorporated into any block.
func ListPoolAttesterSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.AttesterSlashingsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

// SubmitAttesterSlashing submits AttesterSlashing object to node's pool and
// if passes validation node MUST broadcast it to network.
func SubmitAttesterSlashing(ctx context.Context, req *ethpb.AttesterSlashing) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// ListPoolProposerSlashings retrieves proposer slashings known by the node
// but not necessarily incorporated into any block.
func ListPoolProposerSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashingPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

// SubmitProposerSlashing submits AttesterSlashing object to node's pool and if
// passes validation node MUST broadcast it to network.
func SubmitProposerSlashing(ctx context.Context, req *ethpb.ProposerSlashing) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// ListPoolVoluntaryExits retrieves voluntary exits known by the node but
// not necessarily incorporated into any block.
func ListPoolVoluntaryExits(ctx context.Context, req *ptypes.Empty) (*ethpb.VoluntaryExitsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

// SubmitVoluntaryExit submits SignedVoluntaryExit object to node's pool
// and if passes validation node MUST broadcast it to network.
func SubmitVoluntaryExit(ctx context.Context, req *ethpb.SignedVoluntaryExit) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}
