package validator

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	v1alpha1Server "github.com/prysmaticlabs/prysm/beacon-chain/rpc/prysm/v1alpha1/validator"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1alpha1Srv v1alpha1Server.Server
}

func (s Server) GetDuties(ctx context.Context, request *v2.DutiesRequest) (*v2.DutiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) StreamDuties(request *v2.DutiesRequest, server v2.BeaconNodeValidator_StreamDutiesServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) DomainData(ctx context.Context, request *v2.DomainRequest) (*v2.DomainResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) WaitForChainStart(empty *empty.Empty, server v2.BeaconNodeValidator_WaitForChainStartServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) WaitForActivation(request *v2.ValidatorActivationRequest, server v2.BeaconNodeValidator_WaitForActivationServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ValidatorIndex(ctx context.Context, request *v2.ValidatorIndexRequest) (*v2.ValidatorIndexResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ValidatorStatus(ctx context.Context, request *v2.ValidatorStatusRequest) (*v2.ValidatorStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) MultipleValidatorStatus(ctx context.Context, request *v2.MultipleValidatorStatusRequest) (*v2.MultipleValidatorStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetBlock(ctx context.Context, request *v2.BlockRequest) (*v2.BeaconBlock, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ProposeBlock(ctx context.Context, block *v2.SignedBeaconBlock) (*v2.ProposeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetAttestationData(ctx context.Context, request *v2.AttestationDataRequest) (*v2.AttestationData, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ProposeAttestation(ctx context.Context, attestation *v2.Attestation) (*v2.AttestResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubmitAggregateSelectionProof(ctx context.Context, request *v2.AggregateSelectionRequest) (*v2.AggregateSelectionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubmitSignedAggregateSelectionProof(ctx context.Context, request *v2.SignedAggregateSubmitRequest) (*v2.SignedAggregateSubmitResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ProposeExit(ctx context.Context, exit *v2.SignedVoluntaryExit) (*v2.ProposeExitResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubscribeCommitteeSubnets(ctx context.Context, request *v2.CommitteeSubnetsSubscribeRequest) (*empty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) CheckDoppelGanger(ctx context.Context, request *v2.DoppelGangerRequest) (*v2.DoppelGangerResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetBlockAltair(ctx context.Context, request *ethpb.BlockRequest) (*v2.BeaconBlockAltair, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ProposeBlockAltair(ctx context.Context, altair *v2.SignedBeaconBlockAltair) (*ethpb.ProposeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetSyncMessageBlockRoot(ctx context.Context, empty *empty.Empty) (*v2.SyncMessageBlockRootResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubmitSyncMessage(ctx context.Context, message *v2.SyncCommitteeMessage) (*empty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetSyncSubcommitteeIndex(ctx context.Context, request *v2.SyncSubcommitteeIndexRequest) (*v2.SyncSubcommitteeIndexResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetSyncCommitteeContribution(ctx context.Context, request *v2.SyncCommitteeContributionRequest) (*v2.SyncCommitteeContribution, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubmitSignedContributionAndProof(ctx context.Context, proof *v2.SignedContributionAndProof) (*empty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) StreamBlocksAltair(request *ethpb.StreamBlocksRequest, server v2.BeaconNodeValidator_StreamBlocksAltairServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}
