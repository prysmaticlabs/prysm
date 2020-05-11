package client

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

func (v *validator) StreamDuties() error {
	ctx, span := trace.StartSpan(ctx, "validator.StreamDuties")
	defer span.End()

	validatingKeys, err := v.keyManager.FetchValidatingKeys()
	if err != nil {
		return err
	}
	req := &ethpb.DutiesRequest{
		Epoch:      slot / params.BeaconConfig().SlotsPerEpoch,
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
	}
	resp, err := v.validatorClient.StreamDuties(ctx, req)
	if err != nil {
		v.duties = nil // Clear assignments so we know to retry the request.
		log.Error(err)
		return err
	}
	_ = resp
	return nil
}

// UpdateDuties checks the slot number to determine if the validator's
// list of upcoming assignments needs to be updated. For example, at the
// beginning of a new epoch.
func (v *validator) UpdateDuties(ctx context.Context, slot uint64) error {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 && v.duties != nil {
		// Do nothing if not epoch start AND assignments already exist.
		return nil
	}
	// Set deadline to end of epoch.
	ctx, cancel := context.WithDeadline(ctx, v.SlotDeadline(helpers.StartSlot(helpers.SlotToEpoch(slot)+1)))
	defer cancel()
	ctx, span := trace.StartSpan(ctx, "validator.UpdateAssignments")
	defer span.End()

	validatingKeys, err := v.keyManager.FetchValidatingKeys()
	if err != nil {
		return err
	}
	req := &ethpb.DutiesRequest{
		Epoch:      slot / params.BeaconConfig().SlotsPerEpoch,
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
	}

	// If duties is nil it means we have had no prior duties and just started up.
	resp, err := v.validatorClient.GetDuties(ctx, req)
	if err != nil {
		v.duties = nil // Clear assignments so we know to retry the request.
		log.Error(err)
		return err
	}

	v.duties = resp
	v.logDuties(slot, v.duties.Duties)
	subscribeSlots := make([]uint64, 0, len(validatingKeys))
	subscribeCommitteeIDs := make([]uint64, 0, len(validatingKeys))
	subscribeIsAggregator := make([]bool, 0, len(validatingKeys))
	alreadySubscribed := make(map[[64]byte]bool)

	for _, duty := range v.duties.Duties {
		if duty.Status == ethpb.ValidatorStatus_ACTIVE || duty.Status == ethpb.ValidatorStatus_EXITING {
			attesterSlot := duty.AttesterSlot
			committeeIndex := duty.CommitteeIndex

			alreadySubscribedKey := validatorSubscribeKey(attesterSlot, committeeIndex)
			if _, ok := alreadySubscribed[alreadySubscribedKey]; ok {
				continue
			}

			aggregator, err := v.isAggregator(ctx, duty.Committee, attesterSlot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				alreadySubscribed[alreadySubscribedKey] = true
			}

			subscribeSlots = append(subscribeSlots, attesterSlot)
			subscribeCommitteeIDs = append(subscribeCommitteeIDs, committeeIndex)
			subscribeIsAggregator = append(subscribeIsAggregator, aggregator)
		}
	}

	// Notify beacon node to subscribe to the attester and aggregator subnets for the next epoch.
	req.Epoch++
	dutiesNextEpoch, err := v.validatorClient.GetDuties(ctx, req)
	if err != nil {
		log.Error(err)
		return err
	}
	for _, duty := range dutiesNextEpoch.Duties {
		if duty.Status == ethpb.ValidatorStatus_ACTIVE || duty.Status == ethpb.ValidatorStatus_EXITING {
			attesterSlot := duty.AttesterSlot
			committeeIndex := duty.CommitteeIndex

			alreadySubscribedKey := validatorSubscribeKey(attesterSlot, committeeIndex)
			if _, ok := alreadySubscribed[alreadySubscribedKey]; ok {
				continue
			}

			aggregator, err := v.isAggregator(ctx, duty.Committee, attesterSlot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				alreadySubscribed[alreadySubscribedKey] = true
			}

			subscribeSlots = append(subscribeSlots, attesterSlot)
			subscribeCommitteeIDs = append(subscribeCommitteeIDs, committeeIndex)
			subscribeIsAggregator = append(subscribeIsAggregator, aggregator)
		}
	}

	_, err = v.validatorClient.SubscribeCommitteeSubnets(ctx, &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        subscribeSlots,
		CommitteeIds: subscribeCommitteeIDs,
		IsAggregator: subscribeIsAggregator,
	})

	return err
}

// RolesAt slot returns the validator roles at the given slot. Returns nil if the
// validator is known to not have a roles at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid validatorRole map.
func (v *validator) RolesAt(ctx context.Context, slot uint64) (map[[48]byte][]validatorRole, error) {
	rolesAt := make(map[[48]byte][]validatorRole)
	for _, duty := range v.duties.Duties {
		var roles []validatorRole

		if duty == nil {
			continue
		}
		if len(duty.ProposerSlots) > 0 {
			for _, proposerSlot := range duty.ProposerSlots {
				if proposerSlot != 0 && proposerSlot == slot {
					roles = append(roles, roleProposer)
					break
				}
			}
		}
		if duty.AttesterSlot == slot {
			roles = append(roles, roleAttester)

			aggregator, err := v.isAggregator(ctx, duty.Committee, slot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return nil, errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				roles = append(roles, roleAggregator)
			}

		}
		if len(roles) == 0 {
			roles = append(roles, roleUnknown)
		}

		var pubKey [48]byte
		copy(pubKey[:], duty.PublicKey)
		rolesAt[pubKey] = roles
	}
	return rolesAt, nil
}
