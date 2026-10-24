package eip8453

import (
	"context"
	"fmt"
	"slices"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/validators"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/math"
	"github.com/pkg/errors"
)

// ProcessRound settles inactivity and rewards at the end of a round and carries progress into the next one.
func ProcessRound(ctx context.Context, st state.BeaconState) error {
	if err := ProcessInactivityUpdates(ctx, st); err != nil {
		return errors.Wrap(err, "could not process inactivity updates")
	}
	if err := ProcessRewards(ctx, st); err != nil {
		return errors.Wrap(err, "could not process rewards")
	}
	return ProcessParticipationFlagUpdates(st)
}

// ProcessInactivityUpdates lowers the scores of validators that made progress and raises the others, exiting those that reach ejection.
func ProcessInactivityUpdates(ctx context.Context, st state.BeaconState) error {
	participating, err := unslashedParticipating(st, progressFlagIndex)
	if err != nil {
		return err
	}
	scores, err := st.InactivityScores()
	if err != nil {
		return err
	}
	scores = slices.Clone(scores)
	cfg := params.BeaconConfig()
	var ejected []primitives.ValidatorIndex
	for _, index := range activeIndices(st, time.CurrentEpoch(st)) {
		if uint64(index) >= uint64(len(scores)) {
			return fmt.Errorf("validator %d has no inactivity score", index)
		}
		if participating[index] {
			scores[index] -= min(scores[index], cfg.InactivityScoreRecoveryRate)
			continue
		}
		scores[index], err = math.Add64(scores[index], cfg.InactivityScoreBias)
		if err != nil {
			return err
		}
		if scores[index] >= cfg.EjectionInactivityScore {
			ejected = append(ejected, index)
		}
	}
	if err := st.SetInactivityScores(scores); err != nil {
		return err
	}
	if len(ejected) == 0 {
		return nil
	}
	exitInfo := validators.ExitInformation(st)
	for _, index := range ejected {
		_, err := validators.InitiateValidatorExitForTotalBal(ctx, st, index, exitInfo, primitives.Gwei(exitInfo.TotalActiveBalance))
		if err != nil && !errors.Is(err, validators.ErrValidatorAlreadyExited) {
			return err
		}
	}
	return nil
}

// ProcessRewards pays each flag's reward to its unslashed participants, nothing during an inactivity leak.
func ProcessRewards(ctx context.Context, st state.BeaconState) error {
	leak, err := inInactivityLeak(st)
	if err != nil || leak {
		return err
	}
	total, err := helpers.TotalActiveBalance(ctx, st)
	if err != nil {
		return err
	}
	increment := params.BeaconConfig().EffectiveBalanceIncrement
	activeIncrements := total / increment
	for flag, weight := range flagWeights {
		participating, err := unslashedParticipating(st, flag)
		if err != nil {
			return err
		}
		var participatingBalance uint64
		for index := range participating {
			eb, err := st.EffectiveBalanceAtIndex(index)
			if err != nil {
				return err
			}
			participatingBalance += eb
		}
		participatingIncrements := max(increment, participatingBalance) / increment
		for index := range participating {
			eb, err := st.EffectiveBalanceAtIndex(index)
			if err != nil {
				return err
			}
			numerator := roundBaseReward(eb, total) * weight * participatingIncrements
			if err := helpers.IncreaseBalance(st, index, numerator/(activeIncrements*params.BeaconConfig().WeightDenominator)); err != nil {
				return err
			}
		}
	}
	return nil
}

// ProcessParticipationFlagUpdates clears round flags but keeps the progress flag of validators that progressed at the current height.
func ProcessParticipationFlagUpdates(st state.BeaconState) error {
	progress, err := st.ProgressParticipation()
	if err != nil {
		return err
	}
	n := st.NumValidators()
	if len(progress) < n {
		return fmt.Errorf("progress participation has %d entries for %d validators", len(progress), n)
	}
	round := make([]byte, n)
	for index := range round {
		if progress[index] != 0 {
			round[index] = 1 << progressFlagIndex
		}
	}
	return st.SetRoundParticipation(round)
}

func unslashedParticipating(st state.ReadOnlyBeaconState, flag int) (map[primitives.ValidatorIndex]bool, error) {
	round, err := st.RoundParticipation()
	if err != nil {
		return nil, err
	}
	epoch := time.CurrentEpoch(st)
	participating := make(map[primitives.ValidatorIndex]bool)
	for index, val := range st.ValidatorsReadOnlySeq() {
		if !helpers.IsActiveValidatorUsingTrie(val, epoch) {
			continue
		}
		if uint64(index) >= uint64(len(round)) {
			return nil, fmt.Errorf("validator %d has no round participation", index)
		}
		if round[index]&(1<<flag) != 0 && !val.Slashed() {
			participating[index] = true
		}
	}
	return participating, nil
}
