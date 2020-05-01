// Package precompute provides gathering of nicely-structured
// data important to feed into epoch processing, such as attesting
// records and balances, for faster computation.
package precompute

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// New gets called at the beginning of process epoch cycle to return
// pre computed instances of validators attesting records and total
// balances attested in an epoch.
func New(ctx context.Context, state *stateTrie.BeaconState) ([]*Validator, *Balance, error) {
	ctx, span := trace.StartSpan(ctx, "precomputeEpoch.New")
	defer span.End()
	vp := make([]*Validator, state.NumValidators())
	bp := &Balance{}

	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	if err := state.ReadFromEveryValidator(func(idx int, val *stateTrie.ReadOnlyValidator) error {
		// Was validator withdrawable or slashed
		withdrawable := currentEpoch >= val.WithdrawableEpoch()
		p := &Validator{
			IsSlashed:                    val.Slashed(),
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: val.EffectiveBalance(),
		}
		// Was validator active current epoch
		if helpers.IsActiveValidatorUsingTrie(val, currentEpoch) {
			p.IsActiveCurrentEpoch = true
			bp.ActiveCurrentEpoch += val.EffectiveBalance()
		}
		// Was validator active previous epoch
		if helpers.IsActiveValidatorUsingTrie(val, prevEpoch) {
			p.IsActivePrevEpoch = true
			bp.ActivePrevEpoch += val.EffectiveBalance()
		}
		// Set inclusion slot and inclusion distance to be max, they will be compared and replaced
		// with the lower values
		p.InclusionSlot = params.BeaconConfig().FarFutureEpoch
		p.InclusionDistance = params.BeaconConfig().FarFutureEpoch

		vp[idx] = p
		return nil
	}); err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize precompute")
	}
	return vp, bp, nil
}
