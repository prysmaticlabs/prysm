package streaming

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// StreamDuties consumes a server-side stream of validator duties from a beacon node
// for a set of validating keys passed in as a request type. New duties will be
// sent over the stream upon a new epoch being reached or from a a chain reorg happening
// across epochs in the beacon node.
func (v *validator) StreamDuties(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.StreamDuties")
	defer span.End()

	validatingKeys, err := v.keyManager.FetchValidatingKeys()
	if err != nil {
		return err
	}
	numValidatingKeys := len(validatingKeys)
	req := &ethpb.DutiesRequest{
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
	}
	stream, err := v.validatorClient.StreamDuties(ctx, req)
	if err != nil {
		return errors.Wrap(err, "Could not setup validator duties streaming client")
	}
	for {
		res, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			return errors.Wrap(ctx.Err(), "context has been canceled so shutting down the loop")
		}
		if err != nil {
			return errors.Wrap(err, "Could not receive duties from stream")
		}
		// Updates validator duties and requests the beacon node to subscribe
		// to attestation subnets in advance.
		v.updateDuties(ctx, res, numValidatingKeys)
		if err := v.requestSubnetSubscriptions(ctx, res, numValidatingKeys); err != nil {
			log.WithError(err).Error("Could not request beacon node to subscribe to subnets")
		}
	}
	return nil
}

// RolesAt slot returns the validator roles at the given slot. Returns nil if the
// validator is known to not have a roles at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid validatorRole map.
func (v *validator) RolesAt(ctx context.Context, slot uint64) (map[[48]byte][]validatorRole, error) {
	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	rolesAt := make(map[[48]byte][]validatorRole)
	v.dutiesLock.RLock()
	duty, ok := v.dutiesByEpoch[epoch]
	if !ok {
		v.dutiesLock.RUnlock()
		log.Debugf("No assigned duties yet for epoch %d", epoch)
		return rolesAt, nil
	}
	v.dutiesLock.RUnlock()
	for _, dt := range duty {
		var roles []validatorRole

		if dt == nil {
			continue
		}
		if len(dt.ProposerSlots) > 0 {
			for _, proposerSlot := range dt.ProposerSlots {
				if proposerSlot != 0 && proposerSlot == slot {
					roles = append(roles, roleProposer)
					break
				}
			}
		}
		if dt.AttesterSlot == slot {
			roles = append(roles, roleAttester)

			aggregator, err := v.isAggregator(ctx, dt.Committee, slot, bytesutil.ToBytes48(dt.PublicKey))
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
		copy(pubKey[:], dt.PublicKey)
		rolesAt[pubKey] = roles
	}
	return rolesAt, nil
}

// Update duties sets the received validator duties in-memory for the validator client
// and determines which validating keys were selected as attestation aggregators
// for the epoch. Additionally, this function uses that information to notify
// the beacon node it should subscribe the assigned attestation p2p subnets.
func (v *validator) updateDuties(ctx context.Context, dutiesResp *ethpb.DutiesResponse, numKeys int) {
	ctx, span := trace.StartSpan(ctx, "validator.updateDuties")
	defer span.End()
	currentSlot := v.CurrentSlot()
	currentEpoch := currentSlot / params.BeaconConfig().SlotsPerEpoch

	v.dutiesLock.Lock()
	v.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty, 2)
	v.dutiesByEpoch[currentEpoch] = dutiesResp.CurrentEpochDuties
	v.dutiesByEpoch[currentEpoch+1] = dutiesResp.NextEpochDuties
	v.dutiesLock.Unlock()

	v.logDuties(currentSlot, dutiesResp.CurrentEpochDuties)
	v.logDuties(currentSlot+params.BeaconConfig().SlotsPerEpoch, dutiesResp.NextEpochDuties)
}

// Given the validator public key and an epoch, this gets the validator assignment.
func (v *validator) duty(pubKey [48]byte, epoch uint64) (*ethpb.DutiesResponse_Duty, error) {
	v.dutiesLock.RLock()
	defer v.dutiesLock.RUnlock()
	duty, ok := v.dutiesByEpoch[epoch]
	if !ok {
		return nil, fmt.Errorf("no duty found for epoch %d", epoch)
	}
	for _, d := range duty {
		if bytes.Equal(pubKey[:], d.PublicKey) {
			return d, nil
		}
	}
	return nil, fmt.Errorf("pubkey %#x not in duties", bytesutil.Trunc(pubKey[:]))
}

func (v *validator) requestSubnetSubscriptions(ctx context.Context, dutiesResp *ethpb.DutiesResponse, numKeys int) error {
	subscribeSlots := make([]uint64, 0, numKeys)
	subscribeCommitteeIDs := make([]uint64, 0, numKeys)
	subscribeIsAggregator := make([]bool, 0, numKeys)
	alreadySubscribed := make(map[[64]byte]bool)
	for _, duty := range dutiesResp.CurrentEpochDuties {
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

	for _, duty := range dutiesResp.NextEpochDuties {
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

	_, err := v.validatorClient.SubscribeCommitteeSubnets(ctx, &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        subscribeSlots,
		CommitteeIds: subscribeCommitteeIDs,
		IsAggregator: subscribeIsAggregator,
	})
	return err
}
