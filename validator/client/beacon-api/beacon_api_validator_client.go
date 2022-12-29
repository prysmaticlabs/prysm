package beacon_api

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/v3/validator/client/iface"
)

type beaconApiValidatorClient struct {
	genesisProvider         genesisProvider
	stateValidatorsProvider stateValidatorsProvider
	dutiesProvider          dutiesProvider
	jsonRestHandler         jsonRestHandler
	fallbackClient          iface.ValidatorClient
}

func NewBeaconApiValidatorClient(host string, timeout time.Duration) iface.ValidatorClient {
	return NewBeaconApiValidatorClientWithFallback(host, timeout, nil)
}

func NewBeaconApiValidatorClientWithFallback(host string, timeout time.Duration, fallbackClient iface.ValidatorClient) iface.ValidatorClient {
	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: timeout},
		host:       host,
	}

	return &beaconApiValidatorClient{
		genesisProvider:         beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler},
		stateValidatorsProvider: beaconApiStateValidatorsProvider{jsonRestHandler: jsonRestHandler},
		dutiesProvider:          beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler},
		jsonRestHandler:         jsonRestHandler,
		fallbackClient:          fallbackClient,
	}
}

func (c *beaconApiValidatorClient) GetDuties(_ context.Context, in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	return c.getDuties(in)
}

func (c *beaconApiValidatorClient) CheckDoppelGanger(ctx context.Context, in *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.CheckDoppelGanger(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.CheckDoppelGanger is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) DomainData(_ context.Context, in *ethpb.DomainRequest) (*ethpb.DomainResponse, error) {
	if len(in.Domain) != 4 {
		return nil, errors.Errorf("invalid domain type: %s", hexutil.Encode(in.Domain))
	}

	domainType := bytesutil.ToBytes4(in.Domain)
	return c.getDomainData(in.Epoch, domainType)
}

func (c *beaconApiValidatorClient) GetAttestationData(_ context.Context, in *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	if in == nil {
		return nil, errors.New("GetAttestationData received nil argument `in`")
	}

	return c.getAttestationData(in.Slot, in.CommitteeIndex)
}

func (c *beaconApiValidatorClient) GetBeaconBlock(_ context.Context, in *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	return c.getBeaconBlock(in.Slot, in.RandaoReveal, in.Graffiti)
}

func (c *beaconApiValidatorClient) GetFeeRecipientByPubKey(ctx context.Context, in *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetFeeRecipientByPubKey(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.GetFeeRecipientByPubKey is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) GetSyncCommitteeContribution(ctx context.Context, in *ethpb.SyncCommitteeContributionRequest) (*ethpb.SyncCommitteeContribution, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetSyncCommitteeContribution(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.GetSyncCommitteeContribution is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) GetSyncMessageBlockRoot(ctx context.Context, in *empty.Empty) (*ethpb.SyncMessageBlockRootResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetSyncMessageBlockRoot(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.GetSyncMessageBlockRoot is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) GetSyncSubcommitteeIndex(ctx context.Context, in *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetSyncSubcommitteeIndex(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.GetSyncSubcommitteeIndex is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) MultipleValidatorStatus(_ context.Context, in *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error) {
	return c.multipleValidatorStatus(in)
}

func (c *beaconApiValidatorClient) PrepareBeaconProposer(_ context.Context, in *ethpb.PrepareBeaconProposerRequest) (*empty.Empty, error) {
	return new(empty.Empty), c.prepareBeaconProposer(in.Recipients)
}

func (c *beaconApiValidatorClient) ProposeAttestation(ctx context.Context, in *ethpb.Attestation) (*ethpb.AttestResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ProposeAttestation(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.ProposeAttestation is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) ProposeBeaconBlock(_ context.Context, in *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	return c.proposeBeaconBlock(in)
}

func (c *beaconApiValidatorClient) ProposeExit(_ context.Context, in *ethpb.SignedVoluntaryExit) (*ethpb.ProposeExitResponse, error) {
	return c.proposeExit(in)
}

func (c *beaconApiValidatorClient) StreamBlocksAltair(ctx context.Context, in *ethpb.StreamBlocksRequest) (ethpb.BeaconNodeValidator_StreamBlocksAltairClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamBlocksAltair(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.StreamBlocksAltair is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) StreamDuties(ctx context.Context, in *ethpb.DutiesRequest) (ethpb.BeaconNodeValidator_StreamDutiesClient, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.StreamDuties(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.StreamDuties is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) SubmitAggregateSelectionProof(ctx context.Context, in *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitAggregateSelectionProof(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitAggregateSelectionProof is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) SubmitSignedAggregateSelectionProof(ctx context.Context, in *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitSignedAggregateSelectionProof(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitSignedAggregateSelectionProof is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) SubmitSignedContributionAndProof(ctx context.Context, in *ethpb.SignedContributionAndProof) (*empty.Empty, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitSignedContributionAndProof(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitSignedContributionAndProof is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) SubmitSyncMessage(ctx context.Context, in *ethpb.SyncCommitteeMessage) (*empty.Empty, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitSyncMessage(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitSyncMessage is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) SubmitValidatorRegistrations(ctx context.Context, in *ethpb.SignedValidatorRegistrationsV1) (*empty.Empty, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.SubmitValidatorRegistrations(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitValidatorRegistrations is not implemented. To use a fallback client, create this validator with NewBeaconApiValidatorClientWithFallback instead.")
}

func (c *beaconApiValidatorClient) SubscribeCommitteeSubnets(_ context.Context, in *ethpb.CommitteeSubnetsSubscribeRequest, validatorIndices []types.ValidatorIndex) (*empty.Empty, error) {
	return new(empty.Empty), c.subscribeCommitteeSubnets(in, validatorIndices)
}

func (c *beaconApiValidatorClient) ValidatorIndex(_ context.Context, in *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	return c.validatorIndex(in)
}

func (c *beaconApiValidatorClient) ValidatorStatus(_ context.Context, in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	return c.validatorStatus(in)
}

func (c *beaconApiValidatorClient) WaitForActivation(ctx context.Context, in *ethpb.ValidatorActivationRequest) (ethpb.BeaconNodeValidator_WaitForActivationClient, error) {
	return c.waitForActivation(ctx, in)
}

// Deprecated: Do not use.
func (c *beaconApiValidatorClient) WaitForChainStart(ctx context.Context, _ *empty.Empty) (*ethpb.ChainStartResponse, error) {
	return c.waitForChainStart(ctx)
}
