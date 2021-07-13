package validator

import (
	"context"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
)

// GetAttesterDuties requests the beacon node to provide a set of attestation duties,
// which should be performed by validators, for a particular epoch.
func (vs *Server) GetAttesterDuties(ctx context.Context, req *ethpb.AttesterDutiesRequest) (*ethpb.AttesterDutiesResponse, error) {
	return nil, errors.New("Unimplemented")
}

// GetProposerDuties requests beacon node to provide all validators that are scheduled to propose a block in the given epoch.
func (vs *Server) GetProposerDuties(ctx context.Context, req *ethpb.ProposerDutiesRequest) (*ethpb.ProposerDutiesResponse, error) {
	return nil, errors.New("Unimplemented")
}

// ProduceBlock requests the beacon node to produce a valid unsigned beacon block, which can then be signed by a proposer and submitted.
func (vs *Server) ProduceBlock(ctx context.Context, req *ethpb.ProduceBlockRequest) (*ethpb.ProduceBlockResponse, error) {
	return nil, errors.New("Unimplemented")
}

// ProduceAttestationData requests that the beacon node produces attestation data for
// the requested committee index and slot based on the nodes current head.
func (vs *Server) ProduceAttestationData(ctx context.Context, req *ethpb.ProduceAttestationDataRequest) (*ethpb.ProduceAttestationDataResponse, error) {
	return nil, errors.New("Unimplemented")
}

// GetAggregateAttestation aggregates all attestations matching the given attestation data root and slot, returning the aggregated result.
func (vs *Server) GetAggregateAttestation(ctx context.Context, req *ethpb.AggregateAttestationRequest) (*ethpb.AggregateAttestationResponse, error) {
	return nil, errors.New("Unimplemented")
}

// SubmitAggregateAndProofs verifies given aggregate and proofs and publishes them on appropriate gossipsub topic.
func (vs *Server) SubmitAggregateAndProofs(ctx context.Context, req *ethpb.SubmitAggregateAndProofsRequest) (*emptypb.Empty, error) {
	return nil, errors.New("Unimplemented")
}

// SubmitBeaconCommitteeSubscription searches using discv5 for peers related to the provided subnet information
// and replaces current peers with those ones if necessary.
func (vs *Server) SubmitBeaconCommitteeSubscription(ctx context.Context, req *ethpb.SubmitBeaconCommitteeSubscriptionsRequest) (*emptypb.Empty, error) {
	return nil, errors.New("Unimplemented")
}
