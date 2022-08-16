package helpers

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

// ValidatorStatus returns a validator's status at the given epoch.
func ValidatorStatus(validator state.ReadOnlyValidator, epoch types.Epoch) (ethpb.ValidatorStatus, error) {
	valStatus, err := ValidatorSubStatus(validator, epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get sub status")
	}
	switch valStatus {
	case ethpb.ValidatorStatus_PENDING_INITIALIZED, ethpb.ValidatorStatus_PENDING_QUEUED:
		return ethpb.ValidatorStatus_PENDING, nil
	case ethpb.ValidatorStatus_ACTIVE_ONGOING, ethpb.ValidatorStatus_ACTIVE_SLASHED, ethpb.ValidatorStatus_ACTIVE_EXITING:
		return ethpb.ValidatorStatus_ACTIVE, nil
	case ethpb.ValidatorStatus_EXITED_UNSLASHED, ethpb.ValidatorStatus_EXITED_SLASHED:
		return ethpb.ValidatorStatus_EXITED, nil
	case ethpb.ValidatorStatus_WITHDRAWAL_POSSIBLE, ethpb.ValidatorStatus_WITHDRAWAL_DONE:
		return ethpb.ValidatorStatus_WITHDRAWAL, nil
	}
	return 0, errors.New("invalid validator state")
}

// ValidatorSubStatus returns a validator's sub-status at the given epoch.
func ValidatorSubStatus(validator state.ReadOnlyValidator, epoch types.Epoch) (ethpb.ValidatorStatus, error) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	// Pending.
	if validator.ActivationEpoch() > epoch {
		if validator.ActivationEligibilityEpoch() == farFutureEpoch {
			return ethpb.ValidatorStatus_PENDING_INITIALIZED, nil
		} else if validator.ActivationEligibilityEpoch() < farFutureEpoch {
			return ethpb.ValidatorStatus_PENDING_QUEUED, nil
		}
	}

	// Active.
	if validator.ActivationEpoch() <= epoch && epoch < validator.ExitEpoch() {
		if validator.ExitEpoch() == farFutureEpoch {
			return ethpb.ValidatorStatus_ACTIVE_ONGOING, nil
		} else if validator.ExitEpoch() < farFutureEpoch {
			if validator.Slashed() {
				return ethpb.ValidatorStatus_ACTIVE_SLASHED, nil
			}
			return ethpb.ValidatorStatus_ACTIVE_EXITING, nil
		}
	}

	// Exited.
	if validator.ExitEpoch() <= epoch && epoch < validator.WithdrawableEpoch() {
		if validator.Slashed() {
			return ethpb.ValidatorStatus_EXITED_SLASHED, nil
		}
		return ethpb.ValidatorStatus_EXITED_UNSLASHED, nil
	}

	if validator.WithdrawableEpoch() <= epoch {
		if validator.EffectiveBalance() != 0 {
			return ethpb.ValidatorStatus_WITHDRAWAL_POSSIBLE, nil
		} else {
			return ethpb.ValidatorStatus_WITHDRAWAL_DONE, nil
		}
	}

	return 0, errors.New("invalid validator state")
}
