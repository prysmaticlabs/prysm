package precompute

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// New gets called at the beginning of process epoch cycle to return
// pre computed instances of validators attesting records and total
// balances attested in an epoch.
func New(ctx context.Context, state *pb.BeaconState) ([]*Validator, *Balance) {
	ctx, span := trace.StartSpan(ctx, "precomputeEpoch.New")
	defer span.End()

	vp := make([]*Validator, len(state.Validators))
	bp := &Balance{}

	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)

	for i, v := range state.Validators {
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
		vp[i] = p
	}
	return vp, bp
}
