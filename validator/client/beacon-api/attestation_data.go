//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) getAttestationData(
	reqSlot types.Slot,
	reqCommitteeIndex types.CommitteeIndex,
) (*ethpb.AttestationData, error) {
	params := url.Values{}
	params.Add("slot", strconv.FormatUint(uint64(reqSlot), 10))
	params.Add("committee_index", strconv.FormatUint(uint64(reqCommitteeIndex), 10))

	query := buildURL(c.url, "eth/v1/validator/attestation_data", params)
	produceAttestationDataResponseJson := rpcmiddleware.ProduceAttestationDataResponseJson{}
	_, err := getRestJsonResponse(c.httpClient, query, &produceAttestationDataResponseJson)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get json response")
	}

	if produceAttestationDataResponseJson.Data == nil {
		return nil, errors.New("attestation data is nil")
	}

	attestationData := produceAttestationDataResponseJson.Data
	committeeIndex, err := strconv.ParseUint(attestationData.CommitteeIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse attestation committee index: %s", attestationData.CommitteeIndex)
	}

	if !validRoot(attestationData.BeaconBlockRoot) {
		return nil, errors.Errorf("invalid beacon block root: %s", attestationData.BeaconBlockRoot)
	}

	beaconBlockRoot, err := hexutil.Decode(attestationData.BeaconBlockRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode beacon block root: %s", attestationData.BeaconBlockRoot)
	}

	slot, err := strconv.ParseUint(attestationData.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse attestation slot: %s", attestationData.Slot)
	}

	if attestationData.Source == nil {
		return nil, errors.New("attestation source is nil")
	}

	sourceEpoch, err := strconv.ParseUint(attestationData.Source.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse attestation source epoch: %s", attestationData.Source.Epoch)
	}

	if !validRoot(attestationData.Source.Root) {
		return nil, errors.Errorf("invalid attestation source root: %s", attestationData.Source.Root)
	}

	sourceRoot, err := hexutil.Decode(attestationData.Source.Root)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode attestation source root: %s", attestationData.Source.Root)
	}

	if attestationData.Target == nil {
		return nil, errors.New("attestation target is nil")
	}

	targetEpoch, err := strconv.ParseUint(attestationData.Target.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse attestation target epoch: %s", attestationData.Target.Epoch)
	}

	if !validRoot(attestationData.Target.Root) {
		return nil, errors.Errorf("invalid attestation target root: %s", attestationData.Target.Root)
	}

	targetRoot, err := hexutil.Decode(attestationData.Target.Root)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode attestation target root: %s", attestationData.Target.Root)
	}

	response := &ethpb.AttestationData{
		BeaconBlockRoot: beaconBlockRoot,
		CommitteeIndex:  types.CommitteeIndex(committeeIndex),
		Slot:            types.Slot(slot),
		Source: &ethpb.Checkpoint{
			Epoch: types.Epoch(sourceEpoch),
			Root:  sourceRoot,
		},
		Target: &ethpb.Checkpoint{
			Epoch: types.Epoch(targetEpoch),
			Root:  targetRoot,
		},
	}

	return response, nil
}
