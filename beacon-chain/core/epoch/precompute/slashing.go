package precompute

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessSlashingsPrecompute processes the slashed validators during epoch processing.
// This is an optimized version by passing in precomputed total epoch balances.
func ProcessSlashingsPrecompute(state *stateTrie.BeaconState, validators []*ethpb.Validator, p *Balance) error {
	currentEpoch := helpers.CurrentEpoch(state)
	exitLength := params.BeaconConfig().EpochsPerSlashingsVector

	// Compute the sum of state slashings
	slashings := state.Slashings()
	totalSlashing := uint64(0)
	for _, slashing := range slashings {
		totalSlashing += slashing
	}

	// Compute slashing for each validator.
	for index, validator := range validators {
		correctEpoch := (currentEpoch + exitLength/2) == validator.WithdrawableEpoch
		if validator.Slashed && correctEpoch {
			minSlashing := mathutil.Min(totalSlashing*3, p.CurrentEpoch)
			increment := params.BeaconConfig().EffectiveBalanceIncrement
			penaltyNumerator := validator.EffectiveBalance / increment * minSlashing
			penalty := penaltyNumerator / p.CurrentEpoch * increment
			if err := helpers.DecreaseBalance(state, uint64(index), penalty); err != nil {
				return err
			}
		}
	}
	return nil
}
