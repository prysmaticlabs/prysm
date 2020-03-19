package rpc

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	SlasherDB *kv.Store
	ctx       context.Context
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	return nil, errors.New("unimplemented")
}

// IsSlashableBlock returns an proposer slashing if the block submitted
// is a double proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, req *ethpb.BeaconBlock) (*slashpb.ProposerSlashingResponse, error) {
	return nil, errors.New("unimplemented")
}
