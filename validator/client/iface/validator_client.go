package iface

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

type BeaconCommitteeSelection struct {
	SelectionProof []byte
	Slot           primitives.Slot
	ValidatorIndex primitives.ValidatorIndex
}

type beaconCommitteeSelectionJson struct {
	SelectionProof string `json:"selection_proof"`
	Slot           string `json:"slot"`
	ValidatorIndex string `json:"validator_index"`
}

func (b BeaconCommitteeSelection) MarshalJSON() ([]byte, error) {
	return json.Marshal(beaconCommitteeSelectionJson{
		SelectionProof: hexutil.Encode(b.SelectionProof),
		Slot:           strconv.FormatUint(uint64(b.Slot), 10),
		ValidatorIndex: strconv.FormatUint(uint64(b.ValidatorIndex), 10),
	})
}

func (b *BeaconCommitteeSelection) UnmarshalJSON(input []byte) error {
	var bjson beaconCommitteeSelectionJson
	err := json.Unmarshal(input, &bjson)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal beacon committee selection")
	}

	slot, err := strconv.ParseUint(bjson.Slot, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse slot")
	}

	vIdx, err := strconv.ParseUint(bjson.ValidatorIndex, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse validator index")
	}

	selectionProof, err := hexutil.Decode(bjson.SelectionProof)
	if err != nil {
		return errors.Wrap(err, "failed to parse selection proof")
	}

	b.Slot = primitives.Slot(slot)
	b.SelectionProof = selectionProof
	b.ValidatorIndex = primitives.ValidatorIndex(vIdx)

	return nil
}

type ValidatorClient interface {
	GetDuties(ctx context.Context, in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error)
	DomainData(ctx context.Context, in *ethpb.DomainRequest) (*ethpb.DomainResponse, error)
	WaitForChainStart(ctx context.Context, in *empty.Empty) (*ethpb.ChainStartResponse, error)
	WaitForActivation(ctx context.Context, in *ethpb.ValidatorActivationRequest) (ethpb.BeaconNodeValidator_WaitForActivationClient, error)
	ValidatorIndex(ctx context.Context, in *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error)
	ValidatorStatus(ctx context.Context, in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error)
	MultipleValidatorStatus(ctx context.Context, in *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error)
	GetBeaconBlock(ctx context.Context, in *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error)
	ProposeBeaconBlock(ctx context.Context, in *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error)
	PrepareBeaconProposer(ctx context.Context, in *ethpb.PrepareBeaconProposerRequest) (*empty.Empty, error)
	GetFeeRecipientByPubKey(ctx context.Context, in *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error)
	GetAttestationData(ctx context.Context, in *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error)
	ProposeAttestation(ctx context.Context, in *ethpb.Attestation) (*ethpb.AttestResponse, error)
	SubmitAggregateSelectionProof(ctx context.Context, in *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error)
	SubmitSignedAggregateSelectionProof(ctx context.Context, in *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error)
	ProposeExit(ctx context.Context, in *ethpb.SignedVoluntaryExit) (*ethpb.ProposeExitResponse, error)
	SubscribeCommitteeSubnets(ctx context.Context, in *ethpb.CommitteeSubnetsSubscribeRequest, validatorIndices []primitives.ValidatorIndex) (*empty.Empty, error)
	CheckDoppelGanger(ctx context.Context, in *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error)
	GetSyncMessageBlockRoot(ctx context.Context, in *empty.Empty) (*ethpb.SyncMessageBlockRootResponse, error)
	SubmitSyncMessage(ctx context.Context, in *ethpb.SyncCommitteeMessage) (*empty.Empty, error)
	GetSyncSubcommitteeIndex(ctx context.Context, in *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexResponse, error)
	GetSyncCommitteeContribution(ctx context.Context, in *ethpb.SyncCommitteeContributionRequest) (*ethpb.SyncCommitteeContribution, error)
	SubmitSignedContributionAndProof(ctx context.Context, in *ethpb.SignedContributionAndProof) (*empty.Empty, error)
	StreamSlots(ctx context.Context, in *ethpb.StreamSlotsRequest) (ethpb.BeaconNodeValidator_StreamSlotsClient, error)
	SubmitValidatorRegistrations(ctx context.Context, in *ethpb.SignedValidatorRegistrationsV1) (*empty.Empty, error)
	StartEventStream(ctx context.Context) error
	EventStreamIsRunning() bool
	GetAggregatedSelections(ctx context.Context, selections []BeaconCommitteeSelection) ([]BeaconCommitteeSelection, error)
}
