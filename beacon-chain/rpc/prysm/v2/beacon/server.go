package beacon

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	v1alpha1Server "github.com/prysmaticlabs/prysm/beacon-chain/rpc/prysm/v1alpha1/beacon"
	v2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1alpha1Srv v1alpha1Server.Server
}

func (s Server) ListAttestations(ctx context.Context, request *v2.ListAttestationsRequest) (*v2.ListAttestationsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListIndexedAttestations(ctx context.Context, request *v2.ListIndexedAttestationsRequest) (*v2.ListIndexedAttestationsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) StreamAttestations(empty *empty.Empty, server v2.BeaconChain_StreamAttestationsServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) StreamIndexedAttestations(empty *empty.Empty, server v2.BeaconChain_StreamIndexedAttestationsServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) AttestationPool(ctx context.Context, request *v2.AttestationPoolRequest) (*v2.AttestationPoolResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListBlocks(ctx context.Context, request *v2.ListBlocksRequest) (*v2.ListBlocksResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) StreamChainHead(empty *empty.Empty, server v2.BeaconChain_StreamChainHeadServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetChainHead(ctx context.Context, empty *empty.Empty) (*v2.ChainHead, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetWeakSubjectivityCheckpoint(ctx context.Context, empty *empty.Empty) (*v2.WeakSubjectivityCheckpoint, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListBeaconCommittees(ctx context.Context, request *v2.ListCommitteesRequest) (*v2.BeaconCommittees, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListValidatorBalances(ctx context.Context, request *v2.ListValidatorBalancesRequest) (*v2.ValidatorBalances, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListValidators(ctx context.Context, request *v2.ListValidatorsRequest) (*v2.Validators, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetValidator(ctx context.Context, request *v2.GetValidatorRequest) (*v2.Validator, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetValidatorActiveSetChanges(ctx context.Context, request *v2.GetValidatorActiveSetChangesRequest) (*v2.ActiveSetChanges, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetValidatorQueue(ctx context.Context, empty *empty.Empty) (*v2.ValidatorQueue, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetValidatorPerformance(ctx context.Context, request *v2.ValidatorPerformanceRequest) (*v2.ValidatorPerformanceResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) ListValidatorAssignments(ctx context.Context, request *v2.ListValidatorAssignmentsRequest) (*v2.ValidatorAssignments, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetValidatorParticipation(ctx context.Context, request *v2.GetValidatorParticipationRequest) (*v2.ValidatorParticipationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetBeaconConfig(ctx context.Context, empty *empty.Empty) (*v2.BeaconConfig, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) StreamValidatorsInfo(server v2.BeaconChain_StreamValidatorsInfoServer) error {
	return status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubmitAttesterSlashing(ctx context.Context, slashing *v2.AttesterSlashing) (*v2.SubmitSlashingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) SubmitProposerSlashing(ctx context.Context, slashing *v2.ProposerSlashing) (*v2.SubmitSlashingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s Server) GetIndividualVotes(ctx context.Context, request *v2.IndividualVotesRequest) (*v2.IndividualVotesRespond, error) {
	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}
