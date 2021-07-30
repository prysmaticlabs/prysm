package altair

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// InitializeEpochValidators gets called at the beginning of process epoch cycle to return
// pre computed instances of validators attesting records and total
// balances attested in an epoch.
func InitializeEpochValidators(ctx context.Context, st state.BeaconStateAltair) ([]*precompute.Validator, *precompute.Balance, error) {
	ctx, span := trace.StartSpan(ctx, "altair.InitializeEpochValidators")
	defer span.End()
	pValidators := make([]*precompute.Validator, st.NumValidators())
	bal := &precompute.Balance{}
	prevEpoch := helpers.PrevEpoch(st)

	inactivityScores, err := st.InactivityScores()
	if err != nil {
		return nil, nil, err
	}

	// This shouldn't happen with a correct beacon state,
	// but rather be safe to defend against index out of bound panics.
	if st.NumValidators() > len(inactivityScores) {
		return nil, nil, errors.New("num of validators can't be greater than length of inactivity scores")
	}
	if err := st.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		// Was validator withdrawable or slashed
		withdrawable := prevEpoch+1 >= val.WithdrawableEpoch()
		pVal := &precompute.Validator{
			IsSlashed:                    val.Slashed(),
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: val.EffectiveBalance(),
			InactivityScore:              inactivityScores[idx],
		}
		// Validator active current epoch
		if helpers.IsActiveValidatorUsingTrie(val, helpers.CurrentEpoch(st)) {
			pVal.IsActiveCurrentEpoch = true
			bal.ActiveCurrentEpoch += val.EffectiveBalance()
		}
		// Validator active previous epoch
		if helpers.IsActiveValidatorUsingTrie(val, prevEpoch) {
			pVal.IsActivePrevEpoch = true
			bal.ActivePrevEpoch += val.EffectiveBalance()
		}

		pValidators[idx] = pVal
		return nil
	}); err != nil {
		return nil, nil, errors.Wrap(err, "could not initialize epoch validator")
	}
	return pValidators, bal, nil
}

// ProcessInactivityScores of beacon chain. This updates inactivity scores of beacon chain and
// updates the precompute validator struct for later processing.
func ProcessInactivityScores(
	ctx context.Context,
	state state.BeaconState,
	vals []*precompute.Validator,
) (state.BeaconState, []*precompute.Validator, error) {
	cfg := params.BeaconConfig()
	if helpers.CurrentEpoch(state) == cfg.GenesisEpoch {
		return state, vals, nil
	}

	inactivityScores, err := state.InactivityScores()
	if err != nil {
		return nil, nil, err
	}

	bias := cfg.InactivityScoreBias
	recoveryRate := cfg.InactivityScoreRecoveryRate
	for i, v := range vals {
		if v.IsPrevEpochTargetAttester && !v.IsSlashed {
			// Decrease inactivity score when validator gets target correct.
			if v.InactivityScore > 0 {
				score := uint64(1)
				// Prevents underflow below 0.
				if score > v.InactivityScore {
					score = v.InactivityScore
				}
				v.InactivityScore -= score
			}
		} else {
			v.InactivityScore += bias
		}

		if !helpers.IsInInactivityLeak(helpers.PrevEpoch(state), state.FinalizedCheckpointEpoch()) {
			score := recoveryRate
			// Prevents underflow below 0.
			if score > v.InactivityScore {
				score = v.InactivityScore
			}
			v.InactivityScore -= score
		}
		inactivityScores[i] = v.InactivityScore
	}

	if err := state.SetInactivityScores(inactivityScores); err != nil {
		return nil, nil, err
	}

	return state, vals, nil
}

// ProcessEpochParticipation processes the epoch participation in state and updates individual validator's pre computes,
// it also tracks and updates epoch attesting balances.
func ProcessEpochParticipation(
	ctx context.Context,
	state state.BeaconState,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) ([]*precompute.Validator, *precompute.Balance, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessEpochParticipation")
	defer span.End()

	cp, err := state.CurrentEpochParticipation()
	if err != nil {
		return nil, nil, err
	}
	cfg := params.BeaconConfig()
	targetIdx := cfg.TimelyTargetFlagIndex
	sourceIdx := cfg.TimelySourceFlagIndex
	headIdx := cfg.TimelyHeadFlagIndex
	for i, b := range cp {
		if HasValidatorFlag(b, targetIdx) {
			vals[i].IsCurrentEpochTargetAttester = true
		}
	}
	pp, err := state.PreviousEpochParticipation()
	if err != nil {
		return nil, nil, err
	}
	for i, b := range pp {
		if HasValidatorFlag(b, sourceIdx) {
			vals[i].IsPrevEpochAttester = true
		}
		if HasValidatorFlag(b, targetIdx) {
			vals[i].IsPrevEpochTargetAttester = true
		}
		if HasValidatorFlag(b, headIdx) {
			vals[i].IsPrevEpochHeadAttester = true
		}
	}
	bal = precompute.UpdateBalance(vals, bal)
	return vals, bal, nil
}

// ProcessRewardsAndPenaltiesPrecompute processes the rewards and penalties of individual validator.
// This is an optimized version by passing in precomputed validator attesting records and and total epoch balances.
func ProcessRewardsAndPenaltiesPrecompute(
	state state.BeaconStateAltair,
	bal *precompute.Balance,
	vals []*precompute.Validator,
) (state.BeaconStateAltair, error) {
	// Don't process rewards and penalties in genesis epoch.
	if helpers.CurrentEpoch(state) == 0 {
		return state, nil
	}

	numOfVals := state.NumValidators()
	// Guard against an out-of-bounds using validator balance precompute.
	if len(vals) != numOfVals || len(vals) != state.BalancesLength() {
		return state, errors.New("validator registries not the same length as state's validator registries")
	}

	attsRewards, attsPenalties, err := AttestationsDelta(state, bal, vals)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}

	balances := state.Balances()
	for i := 0; i < numOfVals; i++ {
		vals[i].BeforeEpochTransitionBalance = balances[i]

		// Compute the post balance of the validator after accounting for the
		// attester and proposer rewards and penalties.
		balances[i] = helpers.IncreaseBalanceWithVal(balances[i], attsRewards[i])
		balances[i] = helpers.DecreaseBalanceWithVal(balances[i], attsPenalties[i])

		vals[i].AfterEpochTransitionBalance = balances[i]
	}

	if err := state.SetBalances(balances); err != nil {
		return nil, errors.Wrap(err, "could not set validator balances")
	}

	return state, nil
}

// AttestationsDelta computes and returns the rewards and penalties differences for individual validators based on the
// voting records.
func AttestationsDelta(state state.BeaconStateAltair, bal *precompute.Balance, vals []*precompute.Validator) (rewards, penalties []uint64, err error) {
	numOfVals := state.NumValidators()
	rewards = make([]uint64, numOfVals)
	penalties = make([]uint64, numOfVals)
	prevEpoch := helpers.PrevEpoch(state)
	finalizedEpoch := state.FinalizedCheckpointEpoch()

	for i, v := range vals {
		rewards[i], penalties[i] = attestationDelta(bal, v, prevEpoch, finalizedEpoch)
	}

	return rewards, penalties, nil
}

func attestationDelta(bal *precompute.Balance, v *precompute.Validator, prevEpoch, finalizedEpoch types.Epoch) (r, p uint64) {
	eligible := v.IsActivePrevEpoch || (v.IsSlashed && !v.IsWithdrawableCurrentEpoch)
	// Per spec `ActiveCurrentEpoch` can't be 0 to process attestation delta.
	if !eligible || bal.ActiveCurrentEpoch == 0 {
		return 0, 0
	}

	cfg := params.BeaconConfig()
	balIncrement := cfg.EffectiveBalanceIncrement
	rewardFactor := cfg.BaseRewardFactor
	eb := v.CurrentEpochEffectiveBalance
	baseReward := (eb / balIncrement) * (balIncrement * rewardFactor / mathutil.IntegerSquareRoot(bal.ActiveCurrentEpoch))
	activeEpochIncrement := bal.ActiveCurrentEpoch / balIncrement

	weightDenominator := cfg.WeightDenominator
	srcWeight := cfg.TimelySourceWeight
	tgtWeight := cfg.TimelyTargetWeight
	headWeight := cfg.TimelyHeadWeight
	r, p = uint64(0), uint64(0)
	// Process source reward / penalty
	inactivityLeak := helpers.IsInInactivityLeak(prevEpoch, finalizedEpoch)
	if v.IsPrevEpochAttester && !v.IsSlashed {
		if !inactivityLeak {
			rewardNumerator := baseReward * srcWeight * (bal.PrevEpochAttested / balIncrement)
			r += rewardNumerator / (activeEpochIncrement * weightDenominator)
		}
	} else {
		p += baseReward * srcWeight / weightDenominator
	}

	// Process target reward / penalty
	if v.IsPrevEpochTargetAttester && !v.IsSlashed {
		if !inactivityLeak {
			rewardNumerator := baseReward * tgtWeight * (bal.PrevEpochTargetAttested / balIncrement)
			r += rewardNumerator / (activeEpochIncrement * weightDenominator)
		}
	} else {
		p += baseReward * tgtWeight / weightDenominator
	}

	// Process head reward / penalty
	if v.IsPrevEpochHeadAttester && !v.IsSlashed {
		if !inactivityLeak {
			rewardNumerator := baseReward * headWeight * (bal.PrevEpochHeadAttested / balIncrement)
			r += rewardNumerator / (activeEpochIncrement * weightDenominator)
		}
	}

	// Process finality delay penalty
	// Apply an additional penalty to validators that did not vote on the correct target or slashed
	if !v.IsPrevEpochTargetAttester || v.IsSlashed {
		penaltyNumerator := eb * v.InactivityScore
		scoreBias := cfg.InactivityScoreBias
		quotient := cfg.InactivityPenaltyQuotientAltair
		penaltyDenominator := scoreBias * quotient
		p += penaltyNumerator / penaltyDenominator
	}

	return r, p
}
