package beacon_api

import (
	"context"
	"net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func (c *beaconApiValidatorClient) submitAggregateSelectionProof(ctx context.Context, in *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error) {
	validatorIndexResponse, err := c.validatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: in.PublicKey})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator index")
	}

	attesterDuties, err := c.dutiesProvider.GetAttesterDuties(ctx, slots.ToEpoch(in.Slot), []primitives.ValidatorIndex{validatorIndexResponse.Index})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attester duties")
	}

	if len(attesterDuties) == 0 {
		return nil, errors.Errorf("no attester duty for the given slot %d", in.Slot)
	}

	// First attester duty is required since we requested attester duties for one validator index.
	attesterDuty := attesterDuties[0]

	committeeLen, err := strconv.ParseUint(attesterDuty.CommitteeLength, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse committee length")
	}

	isAggregator, err := helpers.IsAggregator(committeeLen, in.SlotSignature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get aggregator status")
	}
	if !isAggregator {
		return nil, errors.New("validator is not an aggregator")
	}

	attestationData, err := c.getAttestationData(ctx, in.Slot, in.CommitteeIndex)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get attestation data for slot=%d and committee_index=%d", in.Slot, in.CommitteeIndex)
	}

	attestationDataRoot, err := attestationData.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate attestation data root")
	}

	params := url.Values{}
	params.Add("slot", strconv.FormatUint(uint64(in.Slot), 10))
	params.Add("attestation_data_root", hexutil.Encode(attestationDataRoot[:]))
	endpoint := buildURL("/eth/v1/validator/aggregate_attestation", params)

	var aggregateAttestationResponse apimiddleware.AggregateAttestationResponseJson
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, endpoint, &aggregateAttestationResponse); err != nil {
		return nil, errors.Wrap(err, "failed to get aggregate attestation")
	}

	aggregatedAttestation, err := convertAttestationToProto(aggregateAttestationResponse.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert aggregate attestation json to proto")
	}

	return &ethpb.AggregateSelectionResponse{
		AggregateAndProof: &ethpb.AggregateAttestationAndProof{
			AggregatorIndex: validatorIndexResponse.Index,
			Aggregate:       aggregatedAttestation,
			SelectionProof:  in.SlotSignature,
		},
	}, nil
}
