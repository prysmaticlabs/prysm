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
	pBal *Balance,
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

	attsRewards, attsPenalties, err := AttestationsDelta(state, pBal, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	proposerRewards, err := ProposersDelta(state, pBal, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	validatorBals := state.Balances()
	for i := 0; i < numOfVals; i++ {
		vp[i].BeforeEpochTransitionBalance = validatorBals[i]

		// Compute the post balance of the validator after accounting for the
		// attester and proposer rewards and penalties.
		validatorBals[i] = helpers.IncreaseBalanceWithVal(validatorBals[i], attsRewards[i]+proposerRewards[i])
		validatorBals[i] = helpers.DecreaseBalanceWithVal(validatorBals[i], attsPenalties[i])

		vp[i].AfterEpochTransitionBalance = validatorBals[i]
	}

	if err := state.SetBalances(validatorBals); err != nil {
		return nil, errors.Wrap(err, "could not set validator balances")
	}

	return state, nil
}

// AttestationsDelta computes and returns the rewards and penalties differences for individual validators based on the
// voting records.
func AttestationsDelta(state *stateTrie.BeaconState, pBal *Balance, vp []*Validator) ([]uint64, []uint64, error) {
	numOfVals := state.NumValidators()
	rewards := make([]uint64, numOfVals)
	penalties := make([]uint64, numOfVals)
	prevEpoch := helpers.PrevEpoch(state)
	finalizedEpoch := state.FinalizedCheckpointEpoch()

	for i, v := range vp {
		rewards[i], penalties[i] = attestationDelta(pBal, v, prevEpoch, finalizedEpoch)
	}
	return rewards, penalties, nil
}

func attestationDelta(pBal *Balance, v *Validator, prevEpoch uint64, finalizedEpoch uint64) (uint64, uint64) {
	eligible := v.IsActivePrevEpoch || (v.IsSlashed && !v.IsWithdrawableCurrentEpoch)
	if !eligible || pBal.ActiveCurrentEpoch == 0 {
		return 0, 0
	}

	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch
	effectiveBalanceIncrement := params.BeaconConfig().EffectiveBalanceIncrement
	vb := v.CurrentEpochEffectiveBalance
	br := vb * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(pBal.ActiveCurrentEpoch) / baseRewardsPerEpoch
	r, p := uint64(0), uint64(0)
	currentEpochBalance := pBal.ActiveCurrentEpoch / effectiveBalanceIncrement

	// Process source reward / penalty
	if v.IsPrevEpochAttester && !v.IsSlashed {
		proposerReward := br / params.BeaconConfig().ProposerRewardQuotient
		maxAttesterReward := br - proposerReward
		r += maxAttesterReward / v.InclusionDistance

		if isInInactivityLeak(prevEpoch, finalizedEpoch) {
			// Since full base reward will be canceled out by inactivity penalty deltas,
			// optimal participation receives full base reward compensation here.
			r += br
		} else {
			rewardNumerator := br * (pBal.PrevEpochAttested / effectiveBalanceIncrement)
			r += rewardNumerator / currentEpochBalance

		}
	} else {
		p += br
	}

	// Process target reward / penalty
	if v.IsPrevEpochTargetAttester && !v.IsSlashed {
		if isInInactivityLeak(prevEpoch, finalizedEpoch) {
			// Since full base reward will be canceled out by inactivity penalty deltas,
			// optimal participation receives full base reward compensation here.
			r += br
		} else {
			rewardNumerator := br * (pBal.PrevEpochTargetAttested / effectiveBalanceIncrement)
			r += rewardNumerator / currentEpochBalance
		}
	} else {
		p += br
	}

	// Process head reward / penalty
	if v.IsPrevEpochHeadAttester && !v.IsSlashed {
		if isInInactivityLeak(prevEpoch, finalizedEpoch) {
			// Since full base reward will be canceled out by inactivity penalty deltas,
			// optimal participation receives full base reward compensation here.
			r += br
		} else {
			rewardNumerator := br * (pBal.PrevEpochHeadAttested / effectiveBalanceIncrement)
			r += rewardNumerator / currentEpochBalance
		}
	} else {
		p += br
	}

	// Process finality delay penalty
	finalityDelay := finalityDelay(prevEpoch, finalizedEpoch)

	if isInInactivityLeak(prevEpoch, finalizedEpoch) {
		// If validator is performing optimally, this cancels all rewards for a neutral balance.
		proposerReward := br / params.BeaconConfig().ProposerRewardQuotient
		p += baseRewardsPerEpoch*br - proposerReward
		// Apply an additional penalty to validators that did not vote on the correct target or has been slashed.
		// Equivalent to the following condition from the spec:
		// `index not in get_unslashed_attesting_indices(state, matching_target_attestations)`
		if !v.IsPrevEpochTargetAttester || v.IsSlashed {
			p += vb * finalityDelay / params.BeaconConfig().InactivityPenaltyQuotient
		}
	}
	return r, p
}

// ProposersDelta computes and returns the rewards and penalties differences for individual validators based on the
// proposer inclusion records.
func ProposersDelta(state *stateTrie.BeaconState, pBal *Balance, vp []*Validator) ([]uint64, error) {
	numofVals := state.NumValidators()
	rewards := make([]uint64, numofVals)

	totalBalance := pBal.ActiveCurrentEpoch
	balanceSqrt := mathutil.IntegerSquareRoot(totalBalance)
	// Balance square root cannot be 0, this prevents division by 0.
	if balanceSqrt == 0 {
		balanceSqrt = 1
	}

	baseRewardFactor := params.BeaconConfig().BaseRewardFactor
	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch
	proposerRewardQuotient := params.BeaconConfig().ProposerRewardQuotient
	for _, v := range vp {
		if v.ProposerIndex > uint64(len(rewards)) {
			// This should never happen with a valid state / validator.
			return nil, errors.New("proposer index out of range")
		}
		// Only apply inclusion rewards to proposer only if the attested hasn't been slashed.
		if v.IsPrevEpochAttester && !v.IsSlashed {
			vBalance := v.CurrentEpochEffectiveBalance
			baseReward := vBalance * baseRewardFactor / balanceSqrt / baseRewardsPerEpoch
			proposerReward := baseReward / proposerRewardQuotient
			rewards[v.ProposerIndex] += proposerReward
		}
	}
	return rewards, nil
}

// isInInactivityLeak returns true if the state is experiencing inactivity leak.
//
// Spec code:
// def is_in_inactivity_leak(state: BeaconState) -> bool:
//    return get_finality_delay(state) > MIN_EPOCHS_TO_INACTIVITY_PENALTY
func isInInactivityLeak(prevEpoch uint64, finalizedEpoch uint64) bool {
	return finalityDelay(prevEpoch, finalizedEpoch) > params.BeaconConfig().MinEpochsToInactivityPenalty
}

// finalityDelay returns the finality delay using the beacon state.
//
// Spec code:
// def get_finality_delay(state: BeaconState) -> uint64:
//    return get_previous_epoch(state) - state.finalized_checkpoint.epoch
func finalityDelay(prevEpoch uint64, finalizedEpoch uint64) uint64 {
	return prevEpoch - finalizedEpoch
}
