package beacon_api

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
)

type beaconApiBeaconChainClient struct {
	fallbackClient  iface.BeaconChainClient
	jsonRestHandler jsonRestHandler
}

func (c beaconApiBeaconChainClient) ListAttestations(ctx context.Context, in *ethpb.ListAttestationsRequest) (*ethpb.ListAttestationsResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListAttestations(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListAttestations is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) ListIndexedAttestations(ctx context.Context, in *ethpb.ListIndexedAttestationsRequest) (*ethpb.ListIndexedAttestationsResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListIndexedAttestations(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListIndexedAttestations is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

// Deprecated: Do not use.
func (c beaconApiBeaconChainClient) StreamAttestations(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamAttestationsClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamAttestations(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.StreamAttestations is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

// Deprecated: Do not use.
func (c beaconApiBeaconChainClient) StreamIndexedAttestations(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamIndexedAttestationsClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamIndexedAttestations(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.StreamIndexedAttestations is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) AttestationPool(ctx context.Context, in *ethpb.AttestationPoolRequest) (*ethpb.AttestationPoolResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.AttestationPool(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.AttestationPool is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) ListBeaconBlocks(ctx context.Context, in *ethpb.ListBlocksRequest) (*ethpb.ListBeaconBlocksResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListBeaconBlocks(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListBeaconBlocks is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

// Deprecated: Do not use.
func (c beaconApiBeaconChainClient) StreamBlocks(ctx context.Context, in *ethpb.StreamBlocksRequest) (ethpb.BeaconChain_StreamBlocksClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamBlocks(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.StreamBlocks is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

// Deprecated: Do not use.
func (c beaconApiBeaconChainClient) StreamChainHead(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamChainHeadClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamChainHead(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.StreamChainHead is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetChainHead(ctx context.Context, in *empty.Empty) (*ethpb.ChainHead, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetChainHead(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetChainHead is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) ListBeaconCommittees(ctx context.Context, in *ethpb.ListCommitteesRequest) (*ethpb.BeaconCommittees, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListBeaconCommittees(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListBeaconCommittees is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) ListValidatorBalances(ctx context.Context, in *ethpb.ListValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListValidatorBalances(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListValidatorBalances is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) ListValidators(ctx context.Context, in *ethpb.ListValidatorsRequest) (*ethpb.Validators, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListValidators(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListValidators is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetValidator(ctx context.Context, in *ethpb.GetValidatorRequest) (*ethpb.Validator, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetValidator(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetValidator is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetValidatorActiveSetChanges(ctx context.Context, in *ethpb.GetValidatorActiveSetChangesRequest) (*ethpb.ActiveSetChanges, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetValidatorActiveSetChanges(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetValidatorActiveSetChanges is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetValidatorQueue(ctx context.Context, in *empty.Empty) (*ethpb.ValidatorQueue, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetValidatorQueue(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetValidatorQueue is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetValidatorPerformance(ctx context.Context, in *ethpb.ValidatorPerformanceRequest) (*ethpb.ValidatorPerformanceResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetValidatorPerformance(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetValidatorPerformance is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) ListValidatorAssignments(ctx context.Context, in *ethpb.ListValidatorAssignmentsRequest) (*ethpb.ValidatorAssignments, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListValidatorAssignments(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.ListValidatorAssignments is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetValidatorParticipation(ctx context.Context, in *ethpb.GetValidatorParticipationRequest) (*ethpb.ValidatorParticipationResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetValidatorParticipation(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetValidatorParticipation is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetBeaconConfig(ctx context.Context, in *empty.Empty) (*ethpb.BeaconConfig, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetBeaconConfig(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetBeaconConfig is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

// Deprecated: Do not use.
func (c beaconApiBeaconChainClient) StreamValidatorsInfo(ctx context.Context) (ethpb.BeaconChain_StreamValidatorsInfoClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamValidatorsInfo(ctx)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.StreamValidatorsInfo is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) SubmitAttesterSlashing(ctx context.Context, in *ethpb.AttesterSlashing) (*ethpb.SubmitSlashingResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitAttesterSlashing(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.SubmitAttesterSlashing is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) SubmitProposerSlashing(ctx context.Context, in *ethpb.ProposerSlashing) (*ethpb.SubmitSlashingResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitProposerSlashing(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.SubmitProposerSlashing is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func (c beaconApiBeaconChainClient) GetIndividualVotes(ctx context.Context, in *ethpb.IndividualVotesRequest) (*ethpb.IndividualVotesRespond, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetIndividualVotes(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiBeaconChainClient.GetIndividualVotes is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiBeaconChainClientWithFallback.")
}

func NewBeaconApiBeaconChainClientWithFallback(host string, timeout time.Duration, fallbackClient iface.BeaconChainClient) iface.BeaconChainClient {
	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: timeout},
		host:       host,
	}

	return &beaconApiBeaconChainClient{
		jsonRestHandler: jsonRestHandler,
		fallbackClient:  fallbackClient,
	}
}
