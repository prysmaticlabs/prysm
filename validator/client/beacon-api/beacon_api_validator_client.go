package beacon_api

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
)

type ValidatorClientOpt func(*beaconApiValidatorClient)

func WithEventHandler(h *EventHandler) ValidatorClientOpt {
	return func(c *beaconApiValidatorClient) {
		c.eventHandler = h
	}
}

type beaconApiValidatorClient struct {
	genesisProvider         GenesisProvider
	dutiesProvider          dutiesProvider
	stateValidatorsProvider StateValidatorsProvider
	jsonRestHandler         JsonRestHandler
	eventHandler            *EventHandler
	beaconBlockConverter    BeaconBlockConverter
	prysmBeaconChainCLient  iface.PrysmBeaconChainClient
}

func NewBeaconApiValidatorClient(jsonRestHandler JsonRestHandler, opts ...ValidatorClientOpt) iface.ValidatorClient {
	c := &beaconApiValidatorClient{
		genesisProvider:         beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler},
		dutiesProvider:          beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler},
		stateValidatorsProvider: beaconApiStateValidatorsProvider{jsonRestHandler: jsonRestHandler},
		jsonRestHandler:         jsonRestHandler,
		beaconBlockConverter:    beaconApiBeaconBlockConverter{},
		prysmBeaconChainCLient: prysmBeaconChainClient{
			nodeClient:      &beaconApiNodeClient{jsonRestHandler: jsonRestHandler},
			jsonRestHandler: jsonRestHandler,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *beaconApiValidatorClient) GetDuties(ctx context.Context, in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	const action = "GetDuties"
	now := time.Now()
	resp, err := c.getDuties(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) CheckDoppelGanger(ctx context.Context, in *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error) {
	const action = "CheckDoppelGanger"
	now := time.Now()
	resp, err := c.checkDoppelGanger(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) DomainData(ctx context.Context, in *ethpb.DomainRequest) (*ethpb.DomainResponse, error) {
	if len(in.Domain) != 4 {
		return nil, errors.Errorf("invalid domain type: %s", hexutil.Encode(in.Domain))
	}
	domainType := bytesutil.ToBytes4(in.Domain)

	const action = "DomainData"
	now := time.Now()
	resp, err := c.getDomainData(ctx, in.Epoch, domainType)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) GetAttestationData(ctx context.Context, in *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	const action = "GetAttestationData"
	now := time.Now()
	resp, err := c.getAttestationData(ctx, in.Slot, in.CommitteeIndex)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		x := float64(time.Since(now).Milliseconds())
		httpActionLatency.WithLabelValues(action).Observe(x)
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) GetBeaconBlock(ctx context.Context, in *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	const action = "GetBeaconBlock"
	now := time.Now()
	resp, err := c.getBeaconBlock(ctx, in.Slot, in.RandaoReveal, in.Graffiti)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) GetFeeRecipientByPubKey(_ context.Context, _ *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error) {
	return nil, nil
}

func (c *beaconApiValidatorClient) GetSyncCommitteeContribution(ctx context.Context, in *ethpb.SyncCommitteeContributionRequest) (*ethpb.SyncCommitteeContribution, error) {
	const action = "GetSyncCommitteeContribution"
	now := time.Now()
	resp, err := c.getSyncCommitteeContribution(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) GetSyncMessageBlockRoot(ctx context.Context, _ *empty.Empty) (*ethpb.SyncMessageBlockRootResponse, error) {
	const action = "GetSyncMessageBlockRoot"
	now := time.Now()
	resp, err := c.getSyncMessageBlockRoot(ctx)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) GetSyncSubcommitteeIndex(ctx context.Context, in *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexResponse, error) {
	const action = "GetSyncSubcommitteeIndex"
	now := time.Now()
	resp, err := c.getSyncSubcommitteeIndex(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) MultipleValidatorStatus(ctx context.Context, in *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error) {
	const action = "MultipleValidatorStatus"
	now := time.Now()
	resp, err := c.multipleValidatorStatus(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) PrepareBeaconProposer(ctx context.Context, in *ethpb.PrepareBeaconProposerRequest) (*empty.Empty, error) {
	const action = "PrepareBeaconProposer"
	now := time.Now()
	err := c.prepareBeaconProposer(ctx, in.Recipients)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return new(empty.Empty), err
}

func (c *beaconApiValidatorClient) ProposeAttestation(ctx context.Context, in *ethpb.Attestation) (*ethpb.AttestResponse, error) {
	const action = "ProposeAttestation"
	now := time.Now()
	resp, err := c.proposeAttestation(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) ProposeBeaconBlock(ctx context.Context, in *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	const action = "ProposeBeaconBlock"
	now := time.Now()
	resp, err := c.proposeBeaconBlock(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) ProposeExit(ctx context.Context, in *ethpb.SignedVoluntaryExit) (*ethpb.ProposeExitResponse, error) {
	const action = "ProposeExit"
	now := time.Now()
	resp, err := c.proposeExit(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) StreamSlots(ctx context.Context, in *ethpb.StreamSlotsRequest) (ethpb.BeaconNodeValidator_StreamSlotsClient, error) {
	return c.streamSlots(ctx, in, time.Second), nil
}

func (c *beaconApiValidatorClient) StreamBlocksAltair(ctx context.Context, in *ethpb.StreamBlocksRequest) (ethpb.BeaconNodeValidator_StreamBlocksAltairClient, error) {
	return c.streamBlocks(ctx, in, time.Second), nil
}

func (c *beaconApiValidatorClient) SubmitAggregateSelectionProof(ctx context.Context, in *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error) {
	const action = "SubmitAggregateSelectionProof"
	now := time.Now()
	resp, err := c.submitAggregateSelectionProof(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) SubmitSignedAggregateSelectionProof(ctx context.Context, in *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	const action = "SubmitSignedAggregateSelectionProof"
	now := time.Now()
	resp, err := c.submitSignedAggregateSelectionProof(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) SubmitSignedContributionAndProof(ctx context.Context, in *ethpb.SignedContributionAndProof) (*empty.Empty, error) {
	const action = "SubmitSignedContributionAndProof"
	now := time.Now()
	err := c.submitSignedContributionAndProof(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return new(empty.Empty), err
}

func (c *beaconApiValidatorClient) SubmitSyncMessage(ctx context.Context, in *ethpb.SyncCommitteeMessage) (*empty.Empty, error) {
	const action = "SubmitSyncMessage"
	now := time.Now()
	err := c.submitSyncMessage(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return new(empty.Empty), err
}

func (c *beaconApiValidatorClient) SubmitValidatorRegistrations(ctx context.Context, in *ethpb.SignedValidatorRegistrationsV1) (*empty.Empty, error) {
	const action = "SubmitValidatorRegistrations"
	now := time.Now()
	err := c.submitValidatorRegistrations(ctx, in.Messages)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return new(empty.Empty), err
}

func (c *beaconApiValidatorClient) SubscribeCommitteeSubnets(ctx context.Context, in *ethpb.CommitteeSubnetsSubscribeRequest, validatorIndices []primitives.ValidatorIndex) (*empty.Empty, error) {
	const action = "SubscribeCommitteeSubnets"
	now := time.Now()
	err := c.subscribeCommitteeSubnets(ctx, in, validatorIndices)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return new(empty.Empty), err
}

func (c *beaconApiValidatorClient) ValidatorIndex(ctx context.Context, in *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	const action = "ValidatorIndex"
	now := time.Now()
	resp, err := c.validatorIndex(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) ValidatorStatus(ctx context.Context, in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	const action = "ValidatorStatus"
	now := time.Now()
	resp, err := c.validatorStatus(ctx, in)
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(float64(time.Since(now).Milliseconds()))
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}

func (c *beaconApiValidatorClient) WaitForActivation(ctx context.Context, in *ethpb.ValidatorActivationRequest) (ethpb.BeaconNodeValidator_WaitForActivationClient, error) {
	return c.waitForActivation(ctx, in)
}

// Deprecated: Do not use.
func (c *beaconApiValidatorClient) WaitForChainStart(ctx context.Context, _ *empty.Empty) (*ethpb.ChainStartResponse, error) {
	return c.waitForChainStart(ctx)
}

func (c *beaconApiValidatorClient) StartEventStream(ctx context.Context) error {
	if c.eventHandler != nil {
		if err := c.eventHandler.get(ctx, []string{"head"}); err != nil {
			return errors.Wrapf(err, "could not invoke event handler")
		}
	}
	return nil
}

func (c *beaconApiValidatorClient) EventStreamIsRunning() bool {
	return c.eventHandler.running
}

func (c *beaconApiValidatorClient) GetAggregatedSelections(ctx context.Context, selections []iface.BeaconCommitteeSelection) ([]iface.BeaconCommitteeSelection, error) {
	return c.getAggregatedSelection(ctx, selections)
}
