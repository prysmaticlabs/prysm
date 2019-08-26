package slasher

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stateFeedListener interface {
	StateInitializedFeed() *event.Feed
}

// SlasherServer defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type SlasherServer struct {
	beaconDB db.Database
	ctx      context.Context
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *SlasherServer) IsSlashableAttestation(ctx context.Context, req *ethpb.Attestation) (*ethpb.AttesterSlashing, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// IsSlashableBlock returns a proposer slashing if the block header submitted is
// a slashable proposal.
func (ss *SlasherServer) IsSlashableBlock(ctx context.Context, req *ethpb.BeaconBlockHeader) (*ethpb.ProposerSlashing, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// SlashableProposals is a subscription to receive all slashable proposer slashing events found by the watchtower.
func (ss *SlasherServer) SlashableProposals(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashing, error) {
	// TODO this should be a stream
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// SlashableAttestations is a subscription to receive all slashable attester slashing events found by the watchtower.
func (ss *SlasherServer) SlashableAttestations(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashing, error) {
	// TODO this should be a stream
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
