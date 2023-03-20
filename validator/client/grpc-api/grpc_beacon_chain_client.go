package grpc_api

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	"google.golang.org/grpc"
)

type grpcBeaconChainClient struct {
	beaconChainClient ethpb.BeaconChainClient
}

func (c *grpcBeaconChainClient) ListAttestations(ctx context.Context, in *ethpb.ListAttestationsRequest) (*ethpb.ListAttestationsResponse, error) {
	return c.beaconChainClient.ListAttestations(ctx, in)
}

func (c *grpcBeaconChainClient) ListIndexedAttestations(ctx context.Context, in *ethpb.ListIndexedAttestationsRequest) (*ethpb.ListIndexedAttestationsResponse, error) {
	return c.beaconChainClient.ListIndexedAttestations(ctx, in)
}

// Deprecated: Do not use.
func (c *grpcBeaconChainClient) StreamAttestations(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamAttestationsClient, error) {
	return c.beaconChainClient.StreamAttestations(ctx, in)
}

// Deprecated: Do not use.
func (c *grpcBeaconChainClient) StreamIndexedAttestations(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamIndexedAttestationsClient, error) {
	return c.beaconChainClient.StreamIndexedAttestations(ctx, in)
}

func (c *grpcBeaconChainClient) AttestationPool(ctx context.Context, in *ethpb.AttestationPoolRequest) (*ethpb.AttestationPoolResponse, error) {
	return c.beaconChainClient.AttestationPool(ctx, in)
}

func (c *grpcBeaconChainClient) ListBeaconBlocks(ctx context.Context, in *ethpb.ListBlocksRequest) (*ethpb.ListBeaconBlocksResponse, error) {
	return c.beaconChainClient.ListBeaconBlocks(ctx, in)
}

// Deprecated: Do not use.
func (c *grpcBeaconChainClient) StreamBlocks(ctx context.Context, in *ethpb.StreamBlocksRequest) (ethpb.BeaconChain_StreamBlocksClient, error) {
	return c.beaconChainClient.StreamBlocks(ctx, in)
}

// Deprecated: Do not use.
func (c *grpcBeaconChainClient) StreamChainHead(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamChainHeadClient, error) {
	return c.beaconChainClient.StreamChainHead(ctx, in)
}

func (c *grpcBeaconChainClient) GetChainHead(ctx context.Context, in *empty.Empty) (*ethpb.ChainHead, error) {
	return c.beaconChainClient.GetChainHead(ctx, in)
}

func (c *grpcBeaconChainClient) ListBeaconCommittees(ctx context.Context, in *ethpb.ListCommitteesRequest) (*ethpb.BeaconCommittees, error) {
	return c.beaconChainClient.ListBeaconCommittees(ctx, in)
}

func (c *grpcBeaconChainClient) ListValidatorBalances(ctx context.Context, in *ethpb.ListValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {
	return c.beaconChainClient.ListValidatorBalances(ctx, in)
}

func (c *grpcBeaconChainClient) ListValidators(ctx context.Context, in *ethpb.ListValidatorsRequest) (*ethpb.Validators, error) {
	return c.beaconChainClient.ListValidators(ctx, in)
}

func (c *grpcBeaconChainClient) GetValidator(ctx context.Context, in *ethpb.GetValidatorRequest) (*ethpb.Validator, error) {
	return c.beaconChainClient.GetValidator(ctx, in)
}

func (c *grpcBeaconChainClient) GetValidatorActiveSetChanges(ctx context.Context, in *ethpb.GetValidatorActiveSetChangesRequest) (*ethpb.ActiveSetChanges, error) {
	return c.beaconChainClient.GetValidatorActiveSetChanges(ctx, in)
}

func (c *grpcBeaconChainClient) GetValidatorQueue(ctx context.Context, in *empty.Empty) (*ethpb.ValidatorQueue, error) {
	return c.beaconChainClient.GetValidatorQueue(ctx, in)
}

func (c *grpcBeaconChainClient) GetValidatorPerformance(ctx context.Context, in *ethpb.ValidatorPerformanceRequest) (*ethpb.ValidatorPerformanceResponse, error) {
	return c.beaconChainClient.GetValidatorPerformance(ctx, in)
}

func (c *grpcBeaconChainClient) ListValidatorAssignments(ctx context.Context, in *ethpb.ListValidatorAssignmentsRequest) (*ethpb.ValidatorAssignments, error) {
	return c.beaconChainClient.ListValidatorAssignments(ctx, in)
}

func (c *grpcBeaconChainClient) GetValidatorParticipation(ctx context.Context, in *ethpb.GetValidatorParticipationRequest) (*ethpb.ValidatorParticipationResponse, error) {
	return c.beaconChainClient.GetValidatorParticipation(ctx, in)
}

func (c *grpcBeaconChainClient) GetBeaconConfig(ctx context.Context, in *empty.Empty) (*ethpb.BeaconConfig, error) {
	return c.beaconChainClient.GetBeaconConfig(ctx, in)
}

// Deprecated: Do not use.
func (c *grpcBeaconChainClient) StreamValidatorsInfo(ctx context.Context) (ethpb.BeaconChain_StreamValidatorsInfoClient, error) {
	return c.beaconChainClient.StreamValidatorsInfo(ctx)
}

func (c *grpcBeaconChainClient) SubmitAttesterSlashing(ctx context.Context, in *ethpb.AttesterSlashing) (*ethpb.SubmitSlashingResponse, error) {
	return c.beaconChainClient.SubmitAttesterSlashing(ctx, in)
}

func (c *grpcBeaconChainClient) SubmitProposerSlashing(ctx context.Context, in *ethpb.ProposerSlashing) (*ethpb.SubmitSlashingResponse, error) {
	return c.beaconChainClient.SubmitProposerSlashing(ctx, in)
}

func (c *grpcBeaconChainClient) GetIndividualVotes(ctx context.Context, in *ethpb.IndividualVotesRequest) (*ethpb.IndividualVotesRespond, error) {
	return c.beaconChainClient.GetIndividualVotes(ctx, in)
}

func NewGrpcBeaconChainClient(cc grpc.ClientConnInterface) iface.BeaconChainClient {
	return &grpcBeaconChainClient{ethpb.NewBeaconChainClient(cc)}
}
