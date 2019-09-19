package slasher

import (
	"context"
	"reflect"

	"github.com/pkg/errors"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	slasherDb db.Store
	ctx       context.Context
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.Attestation) (*ethpb.AttesterSlashing, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// IsSlashableBlock returns a proposer slashing if the block header submitted is
// a slashable proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, psr *ethpb.ProposerSlashingRequest) (*ethpb.ProposerSlashingResponse, error) {
	//TODO: add signature validation
	ep := helpers.SlotToEpoch(psr.BlockHeader.Slot)
	bha, err := ss.slasherDb.BlockHeader(ep, psr.ValidatorIndex)

	if err != nil {
		return nil, errors.Wrap(err, "slasher service error while trying to retrieve blocks")
	}
	pSlashingsResponse := &ethpb.ProposerSlashingResponse{}
	for _, bh := range bha {
		if reflect.DeepEqual(bh, psr.BlockHeader) {
			continue
		}
		pSlashingsResponse.ProposerSlashing = append(pSlashingsResponse.ProposerSlashing, &ethpb.ProposerSlashing{ProposerIndex: psr.ValidatorIndex, Header_1: psr.BlockHeader, Header_2: bh})
	}

	return pSlashingsResponse, nil
}

// SlashableProposals is a subscription to receive all slashable proposer slashing events found by the watchtower.
func (ss *Server) SlashableProposals(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashing, error) {
	// TODO this should be a stream
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// SlashableAttestations is a subscription to receive all slashable attester slashing events found by the watchtower.
func (ss *Server) SlashableAttestations(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashing, error) {
	// TODO this should be a stream
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
