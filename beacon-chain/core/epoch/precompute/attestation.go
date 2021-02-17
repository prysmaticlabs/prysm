package precompute

import (
	"context"

	"go.opencensus.io/trace"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessAttestations process the attestations in state and update individual validator's pre computes,
// it also tracks and updates epoch attesting balances.
func ProcessAttestations(
	ctx context.Context,
	state *stateTrie.BeaconState,
	vp []*Validator,
	pBal *Balance,
) ([]*Validator, *Balance, error) {
	ctx, span := trace.StartSpan(ctx, "precomputeEpoch.ProcessAttestations")
	defer span.End()

	v := &Validator{}
	cp := state.CurrentEpochParticipation()
	for i, b := range cp {
		if helpers.HasValidatorFlag(b, params.BeaconConfig().TimelyTargetFlag) {
			v.IsCurrentEpochTargetAttester = true
		}
		vp[i] = v
	}
	pp := state.PreviousEpochParticipation()
	for i, b := range pp {
		v = vp[i]
		if helpers.HasValidatorFlag(b, params.BeaconConfig().TimelySourceFlag) {
			v.IsPrevEpochSourceAttester = true
		}
		if helpers.HasValidatorFlag(b, params.BeaconConfig().TimelyTargetFlag) {
			v.IsPrevEpochTargetAttester = true
		}
		if helpers.HasValidatorFlag(b, params.BeaconConfig().TimelyHeadFlag) {
			v.IsPrevEpochHeadAttester = true
		}
		vp[i] = v
	}

	pBal = UpdateBalance(vp, pBal)

	return vp, pBal, nil
}

// UpdateBalance updates pre computed balance store.
func UpdateBalance(vp []*Validator, bBal *Balance) *Balance {
	for _, v := range vp {
		if !v.IsSlashed {
			if v.IsCurrentEpochTargetAttester && !v.IsSlashed {
				bBal.CurrentEpochTargetAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochSourceAttester && !v.IsSlashed {
				bBal.PrevEpochSourceAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochTargetAttester && !v.IsSlashed {
				bBal.PrevEpochTargetAttested += v.CurrentEpochEffectiveBalance
			}
			if v.IsPrevEpochHeadAttester && !v.IsSlashed {
				bBal.PrevEpochHeadAttested += v.CurrentEpochEffectiveBalance
			}
		}
	}

	return EnsureBalancesLowerBound(bBal)
}

// EnsureBalancesLowerBound ensures all the balances such as active current epoch, active previous epoch and more
// have EffectiveBalanceIncrement(1 eth) as a lower bound.
func EnsureBalancesLowerBound(bBal *Balance) *Balance {
	ebi := params.BeaconConfig().EffectiveBalanceIncrement
	if ebi > bBal.ActiveCurrentEpoch {
		bBal.ActiveCurrentEpoch = ebi
	}
	if ebi > bBal.ActivePrevEpoch {
		bBal.ActivePrevEpoch = ebi
	}
	if ebi > bBal.CurrentEpochTargetAttested {
		bBal.CurrentEpochTargetAttested = ebi
	}
	if ebi > bBal.PrevEpochSourceAttested {
		bBal.PrevEpochSourceAttested = ebi
	}
	if ebi > bBal.PrevEpochTargetAttested {
		bBal.PrevEpochTargetAttested = ebi
	}
	if ebi > bBal.PrevEpochHeadAttested {
		bBal.PrevEpochHeadAttested = ebi
	}
	return bBal
}
