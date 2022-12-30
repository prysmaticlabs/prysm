package beacon_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

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
	GetCommittees(epoch types.Epoch) ([]*apimiddleware.CommitteeJson, error)
}

type beaconApiDutiesProvider struct {
	jsonRestHandler jsonRestHandler
}

type committeeMappingKey struct {
	committeeIndex types.CommitteeIndex
	slot           types.Slot
}

func (c beaconApiValidatorClient) getDuties(in *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	multipleValidatorStatus, err := c.multipleValidatorStatus(&ethpb.MultipleValidatorStatusRequest{PublicKeys: in.PublicKeys})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator status")
	}

	// Sync committees are an Altair feature
	fetchSyncDuties := in.Epoch >= params.BeaconConfig().AltairForkEpoch

	currentEpochDuties, err := c.getDutiesForEpoch(in.Epoch, multipleValidatorStatus, true, fetchSyncDuties)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get duties for current epoch `%d`", in.Epoch)
	}

	// We don't fetch proposer duties for epoch+1
	nextEpochDuties, err := c.getDutiesForEpoch(in.Epoch+1, multipleValidatorStatus, false, fetchSyncDuties)
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
	multipleValidatorStatus *ethpb.MultipleValidatorStatusResponse,
	fetchProposerDuties bool,
	fetchSyncDuties bool,
) ([]*ethpb.DutiesResponse_Duty, error) {
	attesterDuties, err := c.dutiesProvider.GetAttesterDuties(epoch, multipleValidatorStatus.Indices)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get attester duties for epoch `%d`", epoch)
	}

	var syncDuties []*apimiddleware.SyncCommitteeDuty
	if fetchSyncDuties {
		if syncDuties, err = c.dutiesProvider.GetSyncDuties(epoch, multipleValidatorStatus.Indices); err != nil {
			return nil, errors.Wrapf(err, "failed to get sync duties for epoch `%d`", epoch)
		}
	}

	var proposerDuties []*apimiddleware.ProposerDutyJson
	if fetchProposerDuties {
		if proposerDuties, err = c.dutiesProvider.GetProposerDuties(epoch); err != nil {
			return nil, errors.Wrapf(err, "failed to get proposer duties for epoch `%d`", epoch)
		}
	}

	attesterDutiesMapping := make(map[types.ValidatorIndex]*apimiddleware.AttesterDutyJson)
	for _, attesterDuty := range attesterDuties {
		validatorIndex, err := strconv.ParseUint(attesterDuty.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse validator index `%s`", attesterDuty.ValidatorIndex)
		}

		attesterDutiesMapping[types.ValidatorIndex(validatorIndex)] = attesterDuty
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

	syncDutiesMapping := make(map[types.ValidatorIndex]bool)
	for _, syncDuty := range syncDuties {
		validatorIndex, err := strconv.ParseUint(syncDuty.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse validator index `%s`", syncDuty.ValidatorIndex)
		}

		syncDutiesMapping[types.ValidatorIndex(validatorIndex)] = true
	}

	committees, err := c.dutiesProvider.GetCommittees(epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get committees for epoch `%d`", epoch)
	}

	committeeMapping := make(map[committeeMappingKey][]types.ValidatorIndex)
	for _, committee := range committees {
		committeeIndex, err := strconv.ParseUint(committee.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse committee index `%s`", committee.Index)
		}

		slot, err := strconv.ParseUint(committee.Slot, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse slot `%s`", committee.Slot)
		}

		validatorIndices := make([]types.ValidatorIndex, len(committee.Validators))
		for index, validatorIndexString := range committee.Validators {
			validatorIndex, err := strconv.ParseUint(validatorIndexString, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse validator index `%s`", validatorIndexString)
			}
			validatorIndices[index] = types.ValidatorIndex(validatorIndex)
		}

		key := committeeMappingKey{
			committeeIndex: types.CommitteeIndex(committeeIndex),
			slot:           types.Slot(slot),
		}
		committeeMapping[key] = validatorIndices
	}

	duties := make([]*ethpb.DutiesResponse_Duty, len(multipleValidatorStatus.Statuses))
	for index, validatorStatus := range multipleValidatorStatus.Statuses {
		validatorIndex := multipleValidatorStatus.Indices[index]
		pubkey := multipleValidatorStatus.PublicKeys[index]

		var attesterSlot uint64
		var committeeIndex uint64
		var committeeValidatorIndices []types.ValidatorIndex

		if attesterDuty, ok := attesterDutiesMapping[validatorIndex]; ok {
			attesterSlot, err = strconv.ParseUint(attesterDuty.Slot, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse attester slot `%s`", attesterDuty.Slot)
			}

			committeeIndex, err = strconv.ParseUint(attesterDuty.CommitteeIndex, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse committee index `%s`", attesterDuty.CommitteeIndex)
			}

			key := committeeMappingKey{
				committeeIndex: types.CommitteeIndex(committeeIndex),
				slot:           types.Slot(attesterSlot),
			}
			committeeValidatorIndices, ok = committeeMapping[key]
			if !ok {
				return nil, errors.Wrapf(err, "failed to find validators for committee index `%d` and slot %d", committeeIndex, attesterSlot)
			}
		}

		duties[index] = &ethpb.DutiesResponse_Duty{
			Committee:       committeeValidatorIndices,
			CommitteeIndex:  types.CommitteeIndex(committeeIndex),
			AttesterSlot:    types.Slot(attesterSlot),
			ProposerSlots:   proposerDutySlots[types.ValidatorIndex(validatorIndex)],
			PublicKey:       pubkey,
			Status:          validatorStatus.Status,
			ValidatorIndex:  types.ValidatorIndex(validatorIndex),
			IsSyncCommittee: syncDutiesMapping[types.ValidatorIndex(validatorIndex)],
		}
	}

	return duties, nil
}

// GetCommittees retrieves the committees for the given epoch
func (c beaconApiDutiesProvider) GetCommittees(epoch types.Epoch) ([]*apimiddleware.CommitteeJson, error) {
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

	for index, committee := range stateCommittees.Data {
		if committee == nil {
			return nil, errors.Errorf("committee at index `%d` is nil", index)
		}
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
