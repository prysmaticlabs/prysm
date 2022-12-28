package validator

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/sirupsen/logrus"
)

// Returns true if builder (ie outsourcing block construction) can be used. Both conditions have to meet:
// - Validator has registered to use builder (ie called registerBuilder API end point)
// - Circuit breaker has not been activated (ie the liveness of the chain is healthy)
func (vs *Server) canUseBuilder(ctx context.Context, slot types.Slot, idx types.ValidatorIndex) (bool, error) {
	registered, err := vs.validatorRegistered(ctx, idx)
	if err != nil {
		return false, err
	}
	if !registered {
		return false, nil
	}
	activated, err := vs.circuitBreakBuilder(slot)
	if err != nil {
		return false, err
	}
	return !activated, nil
}

// validatorRegistered returns true if validator with index `id` was previously registered in the database.
func (vs *Server) validatorRegistered(ctx context.Context, id types.ValidatorIndex) (bool, error) {
	if vs.BeaconDB == nil {
		return false, errors.New("nil beacon db")
	}
	_, err := vs.BeaconDB.RegistrationByValidatorID(ctx, id)
	switch {
	case errors.Is(err, kv.ErrNotFoundFeeRecipient):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

// circuitBreakBuilder returns true if the builder is not allowed to be used due to circuit breaker conditions.
func (vs *Server) circuitBreakBuilder(s types.Slot) (bool, error) {
	if vs.ForkFetcher == nil || vs.ForkFetcher.ForkChoicer() == nil {
		return true, errors.New("no fork choicer configured")
	}

	// Circuit breaker is active if the missing consecutive slots greater than `MaxBuilderConsecutiveMissedSlots`.
	highestReceivedSlot := vs.ForkFetcher.ForkChoicer().HighestReceivedBlockSlot()
	maxConsecutiveSkipSlotsAllowed := params.BeaconConfig().MaxBuilderConsecutiveMissedSlots
	diff, err := s.SafeSubSlot(highestReceivedSlot)
	if err != nil {
		return true, err
	}

	if diff > maxConsecutiveSkipSlotsAllowed {
		log.WithFields(logrus.Fields{
			"currentSlot":                    s,
			"highestReceivedSlot":            highestReceivedSlot,
			"maxConsecutiveSkipSlotsAllowed": maxConsecutiveSkipSlotsAllowed,
		}).Warn("Builder circuit breaker activated due to missing consecutive slot")
		return true, nil
	}

	// Not much reason to check missed slots epoch rolling window if input slot is less than epoch.
	if s < params.BeaconConfig().SlotsPerEpoch {
		return false, nil
	}

	// Circuit breaker is active if the missing slots per epoch (rolling window) greater than `MaxBuilderEpochMissedSlots`.
	receivedCount, err := vs.ForkFetcher.ForkChoicer().ReceivedBlocksLastEpoch()
	if err != nil {
		return true, err
	}
	maxEpochSkipSlotsAllowed := params.BeaconConfig().MaxBuilderEpochMissedSlots
	diff, err = params.BeaconConfig().SlotsPerEpoch.SafeSub(receivedCount)
	if err != nil {
		return true, err
	}
	if diff > maxEpochSkipSlotsAllowed {
		log.WithFields(logrus.Fields{
			"totalMissed":              diff,
			"maxEpochSkipSlotsAllowed": maxEpochSkipSlotsAllowed,
		}).Warn("Builder circuit breaker activated due to missing enough slots last epoch")
		return true, nil
	}

	return false, nil
}
