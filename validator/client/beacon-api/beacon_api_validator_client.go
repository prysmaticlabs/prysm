package beacon_api

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/client/event"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
)

type ValidatorClientOpt func(*beaconApiValidatorClient)

type beaconApiValidatorClient struct {
	genesisProvider         GenesisProvider
	dutiesProvider          dutiesProvider
	stateValidatorsProvider StateValidatorsProvider
	jsonRestHandler         JsonRestHandler
	beaconBlockConverter    BeaconBlockConverter
	prysmBeaconChainCLient  iface.PrysmBeaconChainClient
	isEventStreamRunning    bool
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
		isEventStreamRunning: false,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *beaconApiValidatorClient) GetDuties(ctx context.Context, in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	return wrapInMetrics[*ethpb.DutiesResponse]("GetDuties", func() (*ethpb.DutiesResponse, error) {
		return c.getDuties(ctx, in)
	})
}

func (c *beaconApiValidatorClient) CheckDoppelGanger(ctx context.Context, in *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error) {
	return wrapInMetrics[*ethpb.DoppelGangerResponse]("CheckDoppelGanger", func() (*ethpb.DoppelGangerResponse, error) {
		return c.checkDoppelGanger(ctx, in)
	})
}

func (c *beaconApiValidatorClient) DomainData(ctx context.Context, in *ethpb.DomainRequest) (*ethpb.DomainResponse, error) {
	if len(in.Domain) != 4 {
		return nil, errors.Errorf("invalid domain type: %s", hexutil.Encode(in.Domain))
	}
	domainType := bytesutil.ToBytes4(in.Domain)

	return wrapInMetrics[*ethpb.DomainResponse]("DomainData", func() (*ethpb.DomainResponse, error) {
		return c.getDomainData(ctx, in.Epoch, domainType)
	})
}

func (c *beaconApiValidatorClient) GetAttestationData(ctx context.Context, in *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	return wrapInMetrics[*ethpb.AttestationData]("GetAttestationData", func() (*ethpb.AttestationData, error) {
		return c.getAttestationData(ctx, in.Slot, in.CommitteeIndex)
	})
}

func (c *beaconApiValidatorClient) GetBeaconBlock(ctx context.Context, in *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	return wrapInMetrics[*ethpb.GenericBeaconBlock]("GetBeaconBlock", func() (*ethpb.GenericBeaconBlock, error) {
		return c.getBeaconBlock(ctx, in.Slot, in.RandaoReveal, in.Graffiti)
	})
}

func (c *beaconApiValidatorClient) GetFeeRecipientByPubKey(_ context.Context, _ *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error) {
	return nil, nil
}

func (c *beaconApiValidatorClient) GetSyncCommitteeContribution(ctx context.Context, in *ethpb.SyncCommitteeContributionRequest) (*ethpb.SyncCommitteeContribution, error) {
	return wrapInMetrics[*ethpb.SyncCommitteeContribution]("GetSyncCommitteeContribution", func() (*ethpb.SyncCommitteeContribution, error) {
		return c.getSyncCommitteeContribution(ctx, in)
	})
}

func (c *beaconApiValidatorClient) GetSyncMessageBlockRoot(ctx context.Context, _ *empty.Empty) (*ethpb.SyncMessageBlockRootResponse, error) {
	return wrapInMetrics[*ethpb.SyncMessageBlockRootResponse]("GetSyncMessageBlockRoot", func() (*ethpb.SyncMessageBlockRootResponse, error) {
		return c.getSyncMessageBlockRoot(ctx)
	})
}

func (c *beaconApiValidatorClient) GetSyncSubcommitteeIndex(ctx context.Context, in *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexResponse, error) {
	return wrapInMetrics[*ethpb.SyncSubcommitteeIndexResponse]("GetSyncSubcommitteeIndex", func() (*ethpb.SyncSubcommitteeIndexResponse, error) {
		return c.getSyncSubcommitteeIndex(ctx, in)
	})
}

func (c *beaconApiValidatorClient) MultipleValidatorStatus(ctx context.Context, in *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error) {
	return wrapInMetrics[*ethpb.MultipleValidatorStatusResponse]("MultipleValidatorStatus", func() (*ethpb.MultipleValidatorStatusResponse, error) {
		return c.multipleValidatorStatus(ctx, in)
	})
}

func (c *beaconApiValidatorClient) PrepareBeaconProposer(ctx context.Context, in *ethpb.PrepareBeaconProposerRequest) (*empty.Empty, error) {
	return wrapInMetrics[*empty.Empty]("PrepareBeaconProposer", func() (*empty.Empty, error) {
		return new(empty.Empty), c.prepareBeaconProposer(ctx, in.Recipients)
	})
}

func (c *beaconApiValidatorClient) ProposeAttestation(ctx context.Context, in *ethpb.Attestation) (*ethpb.AttestResponse, error) {
	return wrapInMetrics[*ethpb.AttestResponse]("ProposeAttestation", func() (*ethpb.AttestResponse, error) {
		return c.proposeAttestation(ctx, in)
	})
}

func (c *beaconApiValidatorClient) ProposeBeaconBlock(ctx context.Context, in *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	return wrapInMetrics[*ethpb.ProposeResponse]("ProposeBeaconBlock", func() (*ethpb.ProposeResponse, error) {
		return c.proposeBeaconBlock(ctx, in)
	})
}

func (c *beaconApiValidatorClient) ProposeExit(ctx context.Context, in *ethpb.SignedVoluntaryExit) (*ethpb.ProposeExitResponse, error) {
	return wrapInMetrics[*ethpb.ProposeExitResponse]("ProposeExit", func() (*ethpb.ProposeExitResponse, error) {
		return c.proposeExit(ctx, in)
	})
}

func (c *beaconApiValidatorClient) StreamBlocksAltair(ctx context.Context, in *ethpb.StreamBlocksRequest) (ethpb.BeaconNodeValidator_StreamBlocksAltairClient, error) {
	return c.streamBlocks(ctx, in, time.Second), nil
}

func (c *beaconApiValidatorClient) SubmitAggregateSelectionProof(ctx context.Context, in *ethpb.AggregateSelectionRequest, index primitives.ValidatorIndex, committeeLength uint64) (*ethpb.AggregateSelectionResponse, error) {
	return wrapInMetrics[*ethpb.AggregateSelectionResponse]("SubmitAggregateSelectionProof", func() (*ethpb.AggregateSelectionResponse, error) {
		return c.submitAggregateSelectionProof(ctx, in, index, committeeLength)
	})
}

func (c *beaconApiValidatorClient) SubmitSignedAggregateSelectionProof(ctx context.Context, in *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	return wrapInMetrics[*ethpb.SignedAggregateSubmitResponse]("SubmitSignedAggregateSelectionProof", func() (*ethpb.SignedAggregateSubmitResponse, error) {
		return c.submitSignedAggregateSelectionProof(ctx, in)
	})
}

func (c *beaconApiValidatorClient) SubmitSignedContributionAndProof(ctx context.Context, in *ethpb.SignedContributionAndProof) (*empty.Empty, error) {
	return wrapInMetrics[*empty.Empty]("SubmitSignedContributionAndProof", func() (*empty.Empty, error) {
		return new(empty.Empty), c.submitSignedContributionAndProof(ctx, in)
	})
}

func (c *beaconApiValidatorClient) SubmitSyncMessage(ctx context.Context, in *ethpb.SyncCommitteeMessage) (*empty.Empty, error) {
	return wrapInMetrics[*empty.Empty]("SubmitSyncMessage", func() (*empty.Empty, error) {
		return new(empty.Empty), c.submitSyncMessage(ctx, in)
	})
}

func (c *beaconApiValidatorClient) SubmitValidatorRegistrations(ctx context.Context, in *ethpb.SignedValidatorRegistrationsV1) (*empty.Empty, error) {
	return wrapInMetrics[*empty.Empty]("SubmitValidatorRegistrations", func() (*empty.Empty, error) {
		return new(empty.Empty), c.submitValidatorRegistrations(ctx, in.Messages)
	})
}

func (c *beaconApiValidatorClient) SubscribeCommitteeSubnets(ctx context.Context, in *ethpb.CommitteeSubnetsSubscribeRequest, duties []*ethpb.DutiesResponse_Duty) (*empty.Empty, error) {
	return wrapInMetrics[*empty.Empty]("SubscribeCommitteeSubnets", func() (*empty.Empty, error) {
		return new(empty.Empty), c.subscribeCommitteeSubnets(ctx, in, duties)
	})
}

func (c *beaconApiValidatorClient) ValidatorIndex(ctx context.Context, in *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	return wrapInMetrics[*ethpb.ValidatorIndexResponse]("ValidatorIndex", func() (*ethpb.ValidatorIndexResponse, error) {
		return c.validatorIndex(ctx, in)
	})
}

func (c *beaconApiValidatorClient) ValidatorStatus(ctx context.Context, in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	return c.validatorStatus(ctx, in)
}

func (c *beaconApiValidatorClient) WaitForActivation(ctx context.Context, in *ethpb.ValidatorActivationRequest) (ethpb.BeaconNodeValidator_WaitForActivationClient, error) {
	return c.waitForActivation(ctx, in)
}

// Deprecated: Do not use.
func (c *beaconApiValidatorClient) WaitForChainStart(ctx context.Context, _ *empty.Empty) (*ethpb.ChainStartResponse, error) {
	return c.waitForChainStart(ctx)
}

func (c *beaconApiValidatorClient) StartEventStream(ctx context.Context, topics []string, eventsChannel chan<- *event.Event) {
	client := &http.Client{} // event stream should not be subject to the same settings as other api calls, so we won't use c.jsonRestHandler.HttpClient()
	eventStream, err := event.NewEventStream(ctx, client, c.jsonRestHandler.Host(), topics)
	if err != nil {
		eventsChannel <- &event.Event{
			EventType: event.EventError,
			Data:      []byte(errors.Wrap(err, "failed to start event stream").Error()),
		}
		return
	}
	c.isEventStreamRunning = true
	eventStream.Subscribe(eventsChannel)
	c.isEventStreamRunning = false
}

func (c *beaconApiValidatorClient) EventStreamIsRunning() bool {
	return c.isEventStreamRunning
}

func (c *beaconApiValidatorClient) GetAggregatedSelections(ctx context.Context, selections []iface.BeaconCommitteeSelection) ([]iface.BeaconCommitteeSelection, error) {
	return c.getAggregatedSelection(ctx, selections)
}

func (c *beaconApiValidatorClient) GetAggregatedSyncSelections(ctx context.Context, selections []iface.SyncCommitteeSelection) ([]iface.SyncCommitteeSelection, error) {
	return c.getAggregatedSyncSelections(ctx, selections)
}

func wrapInMetrics[Resp any](action string, f func() (Resp, error)) (Resp, error) {
	now := time.Now()
	resp, err := f()
	httpActionCount.WithLabelValues(action).Inc()
	if err == nil {
		httpActionLatency.WithLabelValues(action).Observe(time.Since(now).Seconds())
	} else {
		failedHTTPActionCount.WithLabelValues(action).Inc()
	}
	return resp, err
}
