package beacon_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type dutiesProvider interface {
	GetAttesterDuties(epoch types.Epoch, validatorIndices []types.ValidatorIndex) ([]*apimiddleware.AttesterDutyJson, error)
	GetProposerDuties(epoch types.Epoch) ([]*apimiddleware.ProposerDutyJson, error)
	GetSyncDuties(epoch types.Epoch, validatorIndices []types.ValidatorIndex) ([]*apimiddleware.SyncCommitteeDuty, error)
}

type beaconApiDutiesProvider struct {
	jsonRestHandler jsonRestHandler
}

func (c beaconApiValidatorClient) getDuties(in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	multipleValidatorStatus, err := c.multipleValidatorStatus(&ethpb.MultipleValidatorStatusRequest{PublicKeys: in.PublicKeys})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator status")
	}

	validatorStatusMapping := make(map[types.ValidatorIndex]*ethpb.ValidatorStatusResponse)
	for index, validatorStatus := range multipleValidatorStatus.Statuses {
		validatorIndex := multipleValidatorStatus.Indices[index]
		validatorStatusMapping[validatorIndex] = validatorStatus
	}

	currentEpochDuties, err := c.getDutiesForEpoch(in.Epoch, multipleValidatorStatus.Indices, validatorStatusMapping)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get duties for current epoch `%d`", in.Epoch)
	}

	nextEpochDuties, err := c.getDutiesForEpoch(in.Epoch+1, multipleValidatorStatus.Indices, validatorStatusMapping)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get duties for next epoch `%d`", in.Epoch+1)
	}

	return &ethpb.DutiesResponse{
		Duties:             currentEpochDuties,
		CurrentEpochDuties: currentEpochDuties,
		NextEpochDuties:    nextEpochDuties,
	}, nil
}

func (c beaconApiValidatorClient) getDutiesForEpoch(
	epoch types.Epoch,
	validatorIndices []types.ValidatorIndex,
	validatorStatusMapping map[types.ValidatorIndex]*ethpb.ValidatorStatusResponse,
) ([]*ethpb.DutiesResponse_Duty, error) {
	attesterDuties, err := c.dutiesProvider.GetAttesterDuties(epoch, validatorIndices)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get attester duties for epoch `%d`", epoch)
	}

	proposerDuties, err := c.dutiesProvider.GetProposerDuties(epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get proposer duties for epoch `%d`", epoch)
	}

	var syncDuties []*apimiddleware.SyncCommitteeDuty
	if epoch >= params.BeaconConfig().AltairForkEpoch {
		if syncDuties, err = c.dutiesProvider.GetSyncDuties(epoch, validatorIndices); err != nil {
			return nil, errors.Wrapf(err, "failed to get sync duties for epoch `%d`", epoch)
		}
	}

	committees, err := c.getCommittees(epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get committees for epoch `%d`", epoch)
	}

	proposerDutySlots := make(map[types.ValidatorIndex][]types.Slot)
	for _, proposerDuty := range proposerDuties {
		validatorIndex, err := strconv.ParseUint(proposerDuty.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse validator index `%s`", proposerDuty.ValidatorIndex)
		}

		slot, err := strconv.ParseUint(proposerDuty.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse proposer slot `%s`", proposerDuty.Slot)
		}

		proposerDutySlots[types.ValidatorIndex(validatorIndex)] = append(proposerDutySlots[types.ValidatorIndex(validatorIndex)], types.Slot(slot))
	}

	syncDutiesMapping := make(map[types.ValidatorIndex]*apimiddleware.SyncCommitteeDuty)
	for _, syncDuty := range syncDuties {
		validatorIndex, err := strconv.ParseUint(syncDuty.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse validator index `%s`", syncDuty.ValidatorIndex)
		}

		syncDutiesMapping[types.ValidatorIndex(validatorIndex)] = syncDuty
	}

	committeeMapping := make(map[types.CommitteeIndex][]types.ValidatorIndex)
	for _, committee := range committees {
		committeeIndex, err := strconv.ParseUint(committee.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse committee index `%s`", committee.Index)
		}

		validatorIndices := make([]types.ValidatorIndex, len(committee.Validators))
		for index, validatorIndexString := range committee.Validators {
			validatorIndex, err := strconv.ParseUint(validatorIndexString, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse validator index `%s`", validatorIndexString)
			}
			validatorIndices[index] = types.ValidatorIndex(validatorIndex)
		}

		committeeMapping[types.CommitteeIndex(committeeIndex)] = validatorIndices
	}

	duties := make([]*ethpb.DutiesResponse_Duty, len(attesterDuties))
	for index, attesterDuty := range attesterDuties {
		attesterSlot, err := strconv.ParseUint(attesterDuty.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse attester slot `%s`", attesterDuty.Slot)
		}

		validatorIndex, err := strconv.ParseUint(attesterDuty.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse validator index `%s`", attesterDuty.ValidatorIndex)
		}

		committeeIndex, err := strconv.ParseUint(attesterDuty.CommitteeIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse committee index `%s`", attesterDuty.CommitteeIndex)
		}

		_, isSyncCommittee := syncDutiesMapping[types.ValidatorIndex(validatorIndex)]

		committeeValidatorIndices, ok := committeeMapping[types.CommitteeIndex(committeeIndex)]
		if !ok {
			return nil, errors.Wrapf(err, "failed to find validators for committee index `%d`", committeeIndex)
		}

		pubkey, err := hexutil.Decode(attesterDuty.Pubkey)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode pubkey `%s`", attesterDuty.Pubkey)
		}

		validatorStatus, ok := validatorStatusMapping[types.ValidatorIndex(validatorIndex)]
		if !ok {
			return nil, errors.Errorf("failed to find status validator index `%d`", validatorIndex)
		}

		duties[index] = &ethpb.DutiesResponse_Duty{
			Committee:       committeeValidatorIndices,
			CommitteeIndex:  types.CommitteeIndex(committeeIndex),
			AttesterSlot:    types.Slot(attesterSlot),
			ProposerSlots:   proposerDutySlots[types.ValidatorIndex(validatorIndex)],
			PublicKey:       pubkey,
			Status:          validatorStatus.Status,
			ValidatorIndex:  types.ValidatorIndex(validatorIndex),
			IsSyncCommittee: isSyncCommittee,
		}
	}

	return duties, nil
}

func (c beaconApiValidatorClient) getCommittees(epoch types.Epoch) ([]*apimiddleware.CommitteeJson, error) {
	committeeParams := url.Values{}
	committeeParams.Add("epoch", strconv.FormatUint(uint64(epoch), 10))
	committeesRequest := buildURL("/eth/v1/beacon/states/head/committees", committeeParams)

	var stateCommittees apimiddleware.StateCommitteesResponseJson
	if _, err := c.jsonRestHandler.GetRestJsonResponse(committeesRequest, &stateCommittees); err != nil {
		return nil, errors.Wrapf(err, "failed to query committees for epoch `%d`", epoch)
	}

	if stateCommittees.Data == nil {
		return nil, errors.New("state committees data is nil")
	}

	return stateCommittees.Data, nil
}

func (c beaconApiDutiesProvider) GetAttesterDuties(epoch types.Epoch, validatorIndices []types.ValidatorIndex) ([]*apimiddleware.AttesterDutyJson, error) {

	jsonValidatorIndices := make([]string, len(validatorIndices))
	for index, validatorIndex := range validatorIndices {
		jsonValidatorIndices[index] = strconv.FormatUint(uint64(validatorIndex), 10)
	}

	validatorIndicesBytes, err := json.Marshal(jsonValidatorIndices)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal validator indices")
	}

	attesterDuties := &apimiddleware.AttesterDutiesResponseJson{}
	if _, err := c.jsonRestHandler.PostRestJson(fmt.Sprintf("/eth/v1/validator/duties/attester/%d", epoch), nil, bytes.NewBuffer(validatorIndicesBytes), attesterDuties); err != nil {
		return nil, errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	for index, attesterDuty := range attesterDuties.Data {
		if attesterDuty == nil {
			return nil, errors.Errorf("attester duty at index `%d` is nil", index)
		}
	}

	return attesterDuties.Data, nil
}

func (c beaconApiDutiesProvider) GetProposerDuties(epoch types.Epoch) ([]*apimiddleware.ProposerDutyJson, error) {
	proposerDuties := apimiddleware.ProposerDutiesResponseJson{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse(fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch), &proposerDuties); err != nil {
		return nil, errors.Wrapf(err, "failed to query proposer duties for epoch `%d`", epoch)
	}

	for index, proposerDuty := range proposerDuties.Data {
		if proposerDuty == nil {
			return nil, errors.Errorf("proposer duty at index `%d` is nil", index)
		}
	}

	return proposerDuties.Data, nil
}

func (c beaconApiDutiesProvider) GetSyncDuties(epoch types.Epoch, validatorIndices []types.ValidatorIndex) ([]*apimiddleware.SyncCommitteeDuty, error) {
	jsonValidatorIndices := make([]string, len(validatorIndices))
	for index, validatorIndex := range validatorIndices {
		jsonValidatorIndices[index] = strconv.FormatUint(uint64(validatorIndex), 10)
	}

	validatorIndicesBytes, err := json.Marshal(jsonValidatorIndices)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal validator indices")
	}

	syncDuties := apimiddleware.SyncCommitteeDutiesResponseJson{}
	if _, err := c.jsonRestHandler.PostRestJson(fmt.Sprintf("/eth/v1/validator/duties/sync/%d", epoch), nil, bytes.NewBuffer(validatorIndicesBytes), &syncDuties); err != nil {
		return nil, errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	for index, syncDuty := range syncDuties.Data {
		if syncDuty == nil {
			return nil, errors.Errorf("sync duty at index `%d` is nil", index)
		}
	}

	return syncDuties.Data, nil
}
