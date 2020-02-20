package precompute

import (
	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessRewardsAndPenaltiesPrecompute processes the rewards and penalties of individual validator.
// This is an optimized version by passing in precomputed validator attesting records and and total epoch balances.
func ProcessRewardsAndPenaltiesPrecompute(
	state *stateTrie.BeaconState,
	bp *Balance,
	vp []*Validator,
) (*stateTrie.BeaconState, error) {
	// Can't process rewards and penalties in genesis epoch.
	if helpers.CurrentEpoch(state) == 0 {
		return state, nil
	}

	numOfVals := state.NumValidators()
	// Guard against an out-of-bounds using validator balance precompute.
	if len(vp) != numOfVals || len(vp) != state.BalancesLength() {
		return state, errors.New("precomputed registries not the same length as state registries")
	}

	attsRewards, attsPenalties, err := attestationDeltas(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	proposerRewards, err := proposerDeltaPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	for i := 0; i < numOfVals; i++ {
		vp[i].BeforeEpochTransitionBalance, err = state.BalanceAtIndex(uint64(i))
		if err != nil {
			return nil, errors.Wrap(err, "could not get validator balance before epoch")
		}

		if err := helpers.IncreaseBalance(state, uint64(i), attsRewards[i]+proposerRewards[i]); err != nil {
			return nil, err
		}
		if err := helpers.DecreaseBalance(state, uint64(i), attsPenalties[i]); err != nil {
			return nil, err
		}

		vp[i].AfterEpochTransitionBalance, err = state.BalanceAtIndex(uint64(i))
		if err != nil {
			return nil, errors.Wrap(err, "could not get validator balance after epoch")
		}
	}

	return state, nil
}

// This computes the rewards and penalties differences for individual validators based on the
// voting records.
func attestationDeltas(state *stateTrie.BeaconState, bp *Balance, vp []*Validator) ([]uint64, []uint64, error) {
	numOfVals := state.NumValidators()
	rewards := make([]uint64, numOfVals)
	penalties := make([]uint64, numOfVals)

	for i, v := range vp {
		rewards[i], penalties[i] = attestationDelta(state, bp, v)
	}
	return rewards, penalties, nil
}

func attestationDelta(state *stateTrie.BeaconState, bp *Balance, v *Validator) (uint64, uint64) {
	eligible := v.IsActivePrevEpoch || (v.IsSlashed && !v.IsWithdrawableCurrentEpoch)
	if !eligible {
		return 0, 0
	}

	e := helpers.PrevEpoch(state)
	vb := v.CurrentEpochEffectiveBalance
	br := vb * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(bp.CurrentEpoch) / params.BeaconConfig().BaseRewardsPerEpoch
	r, p := uint64(0), uint64(0)

	// Process source reward / penalty
	if v.IsPrevEpochAttester && !v.IsSlashed {
		r += br * bp.PrevEpochAttesters / bp.CurrentEpoch
		proposerReward := br / params.BeaconConfig().ProposerRewardQuotient
		maxAtteserReward := br - proposerReward
		r += maxAtteserReward / v.InclusionDistance
	} else {
		p += br
	}

	// Process target reward / penalty
	if v.IsPrevEpochTargetAttester && !v.IsSlashed {
		r += br * bp.PrevEpochTargetAttesters / bp.CurrentEpoch
	} else {
		p += br
	}

	// Process head reward / penalty
	if v.IsPrevEpochHeadAttester && !v.IsSlashed {
		r += br * bp.PrevEpochHeadAttesters / bp.CurrentEpoch
	} else {
		p += br
	}

	// Process finality delay penalty
	finalizedEpoch := state.FinalizedCheckpointEpoch()
	finalityDelay := e - finalizedEpoch
	if finalityDelay > params.BeaconConfig().MinEpochsToInactivityPenalty {
		p += params.BeaconConfig().BaseRewardsPerEpoch * br
		if !v.IsPrevEpochTargetAttester {
			p += vb * finalityDelay / params.BeaconConfig().InactivityPenaltyQuotient
		}
	}
	return r, p
}

// This computes the rewards and penalties differences for individual validators based on the
// proposer inclusion records.
func proposerDeltaPrecompute(state *stateTrie.BeaconState, bp *Balance, vp []*Validator) ([]uint64, error) {
	numofVals := state.NumValidators()
	rewards := make([]uint64, numofVals)

	totalBalance := bp.CurrentEpoch

	for _, v := range vp {
		if v.IsPrevEpochAttester {
			vBalance := v.CurrentEpochEffectiveBalance
			baseReward := vBalance * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(totalBalance) / params.BeaconConfig().BaseRewardsPerEpoch
			proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient
			rewards[v.ProposerIndex] += proposerReward
		}
	}
	return rewards, nil
}
