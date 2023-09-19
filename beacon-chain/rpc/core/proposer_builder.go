package core

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/monitoring/tracing"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Returns true if builder (ie outsourcing block construction) can be used. Both conditions have to meet:
// - Validator has registered to use builder (ie called registerBuilder API end point)
// - Circuit breaker has not been activated (ie the liveness of the chain is healthy)
func (s *Service) canUseBuilder(ctx context.Context, slot primitives.Slot, idx primitives.ValidatorIndex) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "proposer.canUseBuilder")
	defer span.End()

	if !s.BlockBuilder.Configured() {
		return false, nil
	}
	activated, err := s.circuitBreakBuilder(slot)
	span.AddAttributes(trace.BoolAttribute("circuitBreakerActivated", activated))
	if err != nil {
		tracing.AnnotateError(span, err)
		return false, err
	}
	if activated {
		return false, nil
	}
	return s.validatorRegistered(ctx, idx)
}

// validatorRegistered returns true if validator with index `id` was previously registered in the database.
func (s *Service) validatorRegistered(ctx context.Context, id primitives.ValidatorIndex) (bool, error) {
	if s.BlockBuilder == nil {
		return false, nil
	}
	_, err := s.BlockBuilder.RegistrationByValidatorID(ctx, id)
	switch {
	case errors.Is(err, kv.ErrNotFoundFeeRecipient), errors.Is(err, cache.ErrNotFoundRegistration):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

// circuitBreakBuilder returns true if the builder is not allowed to be used due to circuit breaker conditions.
func (s *Service) circuitBreakBuilder(slot primitives.Slot) (bool, error) {
	if s.ForkchoiceFetcher == nil {
		return true, errors.New("no fork choicer configured")
	}

	// Circuit breaker is active if the missing consecutive slots greater than `MaxBuilderConsecutiveMissedSlots`.
	highestReceivedSlot := s.ForkchoiceFetcher.HighestReceivedBlockSlot()
	maxConsecutiveSkipSlotsAllowed := params.BeaconConfig().MaxBuilderConsecutiveMissedSlots
	diff, err := slot.SafeSubSlot(highestReceivedSlot)
	if err != nil {
		return true, err
	}

	if diff >= maxConsecutiveSkipSlotsAllowed {
		log.WithFields(logrus.Fields{
			"currentSlot":                    s,
			"highestReceivedSlot":            highestReceivedSlot,
			"maxConsecutiveSkipSlotsAllowed": maxConsecutiveSkipSlotsAllowed,
		}).Warn("Circuit breaker activated due to missing consecutive slot. Ignore if mev-boost is not used")
		return true, nil
	}

	// Not much reason to check missed slots epoch rolling window if input slot is less than epoch.
	if slot < params.BeaconConfig().SlotsPerEpoch {
		return false, nil
	}

	// Circuit breaker is active if the missing slots per epoch (rolling window) greater than `MaxBuilderEpochMissedSlots`.
	receivedCount, err := s.ForkchoiceFetcher.ReceivedBlocksLastEpoch()
	if err != nil {
		return true, err
	}
	maxEpochSkipSlotsAllowed := params.BeaconConfig().MaxBuilderEpochMissedSlots
	diff, err = params.BeaconConfig().SlotsPerEpoch.SafeSub(receivedCount)
	if err != nil {
		return true, err
	}
	if diff >= maxEpochSkipSlotsAllowed {
		log.WithFields(logrus.Fields{
			"totalMissed":              diff,
			"maxEpochSkipSlotsAllowed": maxEpochSkipSlotsAllowed,
		}).Warn("Circuit breaker activated due to missing enough slots last epoch. Ignore if mev-boost is not used")
		return true, nil
	}

	return false, nil
}
