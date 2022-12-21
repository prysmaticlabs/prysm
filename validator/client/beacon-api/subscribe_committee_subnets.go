package beacon_api

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "powchain")

func (c beaconApiValidatorClient) subscribeCommitteeSubnets(in *ethpb.CommitteeSubnetsSubscribeRequest, validatorIndices []types.ValidatorIndex, currentEpoch types.Epoch) error {
	if in == nil {
		return errors.New("committee subnets subscribe request is nil")
	}

	if len(in.CommitteeIds) != len(in.Slots) || len(in.CommitteeIds) != len(in.IsAggregator) || len(in.CommitteeIds) != len(validatorIndices) {
		return errors.New("arrays `in.CommitteeIds`, `in.Slots`, `in.IsAggregator` and `validatorIndices` don't have the same length")
	}

	log.Errorf("****************************CURRENT EPOCH: %d", currentEpoch)
	log.Errorf("****************************SUBSCRIBE SLOTS: %v", in.Slots)
	currentEpochDuties, err := c.getAttesterDuties(currentEpoch, validatorIndices)
	if err != nil {
		return errors.Wrapf(err, "failed to get duties for epoch `%d`", currentEpoch)
	}

	nextEpoch := currentEpoch + 1
	nextEpochDuties, err := c.getAttesterDuties(nextEpoch, validatorIndices)
	if err != nil {
		return errors.Wrapf(err, "failed to get duties for next epoch `%d`", nextEpoch)
	}

	duties := append(currentEpochDuties.Data, nextEpochDuties.Data...)

	slotToCommitteesAtSlotMap := make(map[types.Slot]uint64)
	for _, duty := range duties {
		dutySlot, err := strconv.ParseUint(duty.Slot, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse slot `%s`", duty.Slot)
		}

		committeesAtSlot, err := strconv.ParseUint(duty.CommitteesAtSlot, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse committees at slot `%s`", duty.CommitteesAtSlot)
		}

		slotToCommitteesAtSlotMap[types.Slot(dutySlot)] = committeesAtSlot
	}

	jsonCommitteeSubscriptions := make([]*apimiddleware.BeaconCommitteeSubscribeJson, len(in.CommitteeIds))
	for index := range in.CommitteeIds {
		subscribeSlot := in.Slots[index]
		subscribeCommitteeId := in.CommitteeIds[index]
		subscribeIsAggregator := in.IsAggregator[index]
		subscribeValidatorIndex := validatorIndices[index]

		committeesAtSlot, foundSlot := slotToCommitteesAtSlotMap[subscribeSlot]
		if !foundSlot {
			return errors.Errorf("couldn't find committees for subscription slot `%d`", subscribeSlot)
		}

		jsonCommitteeSubscriptions[index] = &apimiddleware.BeaconCommitteeSubscribeJson{
			CommitteeIndex:   strconv.FormatUint(uint64(subscribeCommitteeId), 10),
			CommitteesAtSlot: strconv.FormatUint(committeesAtSlot, 10),
			Slot:             strconv.FormatUint(uint64(subscribeSlot), 10),
			IsAggregator:     subscribeIsAggregator,
			ValidatorIndex:   strconv.FormatUint(uint64(subscribeValidatorIndex), 10),
		}
	}

	committeeSubscriptionsBytes, err := json.Marshal(jsonCommitteeSubscriptions)
	if err != nil {
		return errors.Wrap(err, "failed to marshal committees subscriptions")
	}

	if _, err := c.jsonRestHandler.PostRestJson("/eth/v1/validator/beacon_committee_subscriptions", nil, bytes.NewBuffer(committeeSubscriptionsBytes), nil); err != nil {
		return errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	return nil
}
