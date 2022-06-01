package core

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/math"
)

func ProcessInactivityScores(ctx context.Context,
	inactivityScores []uint64,
	currentEpoch, previousEpoch, finalizedEpoch types.Epoch,
	vals []*types.Validator,
) ([]uint64, []*types.Validator, error) {

	cfg := params.BeaconConfig()
	if currentEpoch == cfg.GenesisEpoch {
		return inactivityScores, vals, nil
	}

	bias := cfg.InactivityScoreBias
	recoveryRate := cfg.InactivityScoreRecoveryRate

	var err error
	for i, v := range vals {
		if !precompute.EligibleForRewards(v) {
			continue
		}

		if v.IsPrevEpochTargetAttester && !v.IsSlashed {
			// Decrease inactivity score when validator gets target correct.
			if v.InactivityScore > 0 {
				v.InactivityScore -= 1
			}
		} else {
			v.InactivityScore, err = math.Add64(v.InactivityScore, bias)
			if err != nil {
				return nil, nil, err
			}
		}

		if !helpers.IsInInactivityLeak(previousEpoch, finalizedEpoch) {
			score := recoveryRate
			// Prevents underflow below 0.
			if score > v.InactivityScore {
				score = v.InactivityScore
			}
			v.InactivityScore -= score
		}
		inactivityScores[i] = v.InactivityScore
	}

	return inactivityScores, vals, nil
}
