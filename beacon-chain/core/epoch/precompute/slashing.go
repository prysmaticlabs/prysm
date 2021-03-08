package precompute

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessSlashingsPrecompute processes the slashed validators during epoch processing.
// This is an optimized version by passing in precomputed total epoch balances.
func ProcessSlashingsPrecompute(state iface.BeaconState, pBal *Balance) error {
	currentEpoch := helpers.CurrentEpoch(state)
	exitLength := params.BeaconConfig().EpochsPerSlashingsVector

	// Compute the sum of state slashings
	slashings := state.Slashings()
	totalSlashing := uint64(0)
	for _, slashing := range slashings {
		totalSlashing += slashing
	}

	minSlashing := mathutil.Min(totalSlashing*params.BeaconConfig().ProportionalSlashingMultiplier, pBal.ActiveCurrentEpoch)
	epochToWithdraw := currentEpoch + exitLength/2

	var hasSlashing bool
	// Iterate through validator list in state, stop until a validator satisfies slashing condition of current epoch.
	err := state.ReadFromEveryValidator(func(idx int, val iface.ReadOnlyValidator) error {
		correctEpoch := epochToWithdraw == val.WithdrawableEpoch()
		if val.Slashed() && correctEpoch {
			hasSlashing = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Exit early if there's no meaningful slashing to process.
	if !hasSlashing {
		return nil
	}

	increment := params.BeaconConfig().EffectiveBalanceIncrement
	validatorFunc := func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error) {
		correctEpoch := epochToWithdraw == val.WithdrawableEpoch
		if val.Slashed && correctEpoch {
			penaltyNumerator := val.EffectiveBalance / increment * minSlashing
			penalty := penaltyNumerator / pBal.ActiveCurrentEpoch * increment
			if err := helpers.DecreaseBalance(state, types.ValidatorIndex(idx), penalty); err != nil {
				return false, val, err
			}
			return true, val, nil
		}
		return false, val, nil
	}

	return state.ApplyToEveryValidator(validatorFunc)
}
