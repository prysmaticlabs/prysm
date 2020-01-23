package precompute

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// New gets called at the beginning of process epoch cycle to return
// pre computed instances of validators attesting records and total
// balances attested in an epoch.
func New(ctx context.Context, state *stateTrie.BeaconState, validators []*ethpb.Validator) ([]*Validator, *Balance) {
	ctx, span := trace.StartSpan(ctx, "precomputeEpoch.New")
	defer span.End()

	vp := make([]*Validator, len(validators))
	bp := &Balance{}

	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	for i, v := range validators {
		// Was validator withdrawable or slashed
		withdrawable := currentEpoch >= v.WithdrawableEpoch
		p := &Validator{
			IsSlashed:                    v.Slashed,
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: v.EffectiveBalance,
		}
		// Was validator active current epoch
		if helpers.IsActiveValidator(v, currentEpoch) {
			p.IsActiveCurrentEpoch = true
			bp.CurrentEpoch += v.EffectiveBalance
		}
		// Was validator active previous epoch
		if helpers.IsActiveValidator(v, prevEpoch) {
			p.IsActivePrevEpoch = true
			bp.PrevEpoch += v.EffectiveBalance
		}
		// Set inclusion slot and inclusion distance to be max, they will be compared and replaced
		// with the lower values
		p.InclusionSlot = params.BeaconConfig().FarFutureEpoch
		p.InclusionDistance = params.BeaconConfig().FarFutureEpoch

		vp[i] = p
	}
	return vp, bp
}
