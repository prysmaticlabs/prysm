package rpc

import (
	"context"
	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	types "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	SlasherDb *db.Store
	ctx       context.Context
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.Attestation) (*ethpb.AttesterSlashing, error) {
	//TODO(3133): implement attestation validation after attestation store will be merged.
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// IsSlashableBlock returns a proposer slashing if the block header submitted is
// a slashable proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, psr *ethpb.ProposerSlashingRequest) (*ethpb.ProposerSlashingResponse, error) {
	epoch := helpers.SlotToEpoch(psr.BlockHeader.Slot)
	blockHeaders, err := ss.SlasherDb.BlockHeader(epoch, psr.ValidatorIndex)
	if err != nil {
		return nil, errors.Wrap(err, "slasher service error while trying to retrieve blocks")
	}
	pSlashingsResponse := &ethpb.ProposerSlashingResponse{}
	presentInDb := false
	for _, bh := range blockHeaders {
		if proto.Equal(bh, psr.BlockHeader) {
			presentInDb = true
			continue
		}
		pSlashingsResponse.ProposerSlashing = append(pSlashingsResponse.ProposerSlashing, &ethpb.ProposerSlashing{ProposerIndex: psr.ValidatorIndex, Header_1: psr.BlockHeader, Header_2: bh})
	}
	if len(pSlashingsResponse.ProposerSlashing) == 0 && !presentInDb {
		err = ss.SlasherDb.SaveBlockHeader(epoch, psr.ValidatorIndex, psr.BlockHeader)
		if err != nil {
			return nil, err
		}
	}
	return pSlashingsResponse, nil
}

// SlashableProposals is a subscription to receive all slashable proposer slashing events found by the watchtower.
func (ss *Server) SlashableProposals(req *types.Empty, server ethpb.Slasher_SlashableProposalsServer) error {
	//TODO(3133): implement stream provider for newly discovered listening to slashable proposals.
	return status.Error(codes.Unimplemented, "not implemented")
}

// SlashableAttestations is a subscription to receive all slashable attester slashing events found by the watchtower.
func (ss *Server) SlashableAttestations(req *types.Empty, server ethpb.Slasher_SlashableAttestationsServer) error {
	//TODO(3133): implement stream provider for newly discovered listening to slashable attestation.
	return status.Error(codes.Unimplemented, "not implemented")
}
