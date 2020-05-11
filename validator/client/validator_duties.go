package client

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
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
			return errors.Wrap(err, "could not receive validator duties from stream")
		}
		if err := v.updateDuties(res, len(validatingKeys)); err != nil {
			log.WithError(err).Error("Could not update duties from stream")
		}
	}

	return nil
}

func (v *validator) updateDuties(ctx context.Context, dutiesResp *ethpb.DutiesResponse, numKeys int) error {
	ctx, span := trace.StartSpan(ctx, "validator.updateDuties")
	defer span.End()
	currentSlot := slotutil.SlotsSinceGenesis(time.Unix(int64(v.genesisTime), 0))

	v.duties = dutiesResp
	v.logDuties(currentSlot, dutiesResp.CurrentEpochDuties)
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
