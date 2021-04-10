package altair

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// InitializeEpochValidators gets called at the beginning of process epoch cycle to return
// pre computed instances of validators attesting records and total
// balances attested in an epoch.
func InitializeEpochValidators(ctx context.Context, state iface.BeaconState) ([]*precompute.Validator, *precompute.Balance, error) {
	ctx, span := trace.StartSpan(ctx, "altair.InitializeEpochValidators")
	defer span.End()
	pValidators := make([]*precompute.Validator, state.NumValidators())
	pBal := &precompute.Balance{}

	prevEpoch := helpers.PrevEpoch(state)

	if err := state.ReadFromEveryValidator(func(idx int, val iface.ReadOnlyValidator) error {
		// Was validator withdrawable or slashed
		withdrawable := prevEpoch+1 >= val.WithdrawableEpoch()
		pVal := &precompute.Validator{
			IsSlashed:                    val.Slashed(),
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: val.EffectiveBalance(),
		}
		// Was validator active previous epoch
		if helpers.IsActiveValidatorUsingTrie(val, prevEpoch) {
			pVal.IsActivePrevEpoch = true
			pBal.ActivePrevEpoch += val.EffectiveBalance()
		}

		pValidators[idx] = pVal
		return nil
	}); err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize precompute")
	}
	return pValidators, pBal, nil
}

// ProcessEpochParticipation processes the epoch participation in state and update individual validator's pre computes,
// it also tracks and updates epoch attesting balances.
func ProcessEpochParticipation(
	ctx context.Context,
	state iface.BeaconState,
	vp []*precompute.Validator,
	pBal *precompute.Balance,
) ([]*precompute.Validator, *precompute.Balance, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessEpochParticipation")
	defer span.End()

	v := &precompute.Validator{}
	cp, err := state.CurrentEpochParticipation()
	if err != nil {
		return nil, nil, err
	}
	for i, b := range cp {
		if HasValidatorFlag(b, params.BeaconConfig().TimelyTargetFlagIndex) {
			v.IsCurrentEpochTargetAttester = true
		}
		vp[i] = v
	}
	pp, err := state.PreviousEpochParticipation()
	if err != nil {
		return nil, nil, err
	}
	for i, b := range pp {
		v = vp[i]
		if HasValidatorFlag(b, params.BeaconConfig().TimelySourceFlagIndex) {
			v.IsPrevEpochAttester = true
		}
		if HasValidatorFlag(b, params.BeaconConfig().TimelyTargetFlagIndex) {
			v.IsPrevEpochTargetAttester = true
		}
		if HasValidatorFlag(b, params.BeaconConfig().TimelyHeadFlagIndex) {
			v.IsPrevEpochHeadAttester = true
		}
		vp[i] = v
	}

	pBal = precompute.UpdateBalance(vp, pBal)

	return vp, pBal, nil
}
