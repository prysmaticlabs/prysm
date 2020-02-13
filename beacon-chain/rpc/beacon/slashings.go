package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// SubmitProposerSlashing receives a proposer slashing object via
// RPC and injects it into the beacon node's operations pool.
func (bs *Server) SubmitProposerSlashing(
	ctx context.Context,
	req *ethpb.ProposerSlashing,
) (*ethpb.SubmitSlashingResponse, error) {
	return nil, nil
}

// SubmitAttesterSlashing receives an attester slashing object via
// RPC and injects it into the beacon node's operations pool.
func (bs *Server) SubmitAttesterSlashing(
	ctx context.Context,
	req *ethpb.AttesterSlashing,
) (*ethpb.SubmitSlashingResponse, error) {
	return nil, nil
}
