package precompute

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessRewardsAndPenaltiesPrecompute processes the rewards and penalties of individual validator.
// This is an optimized version by passing in precomputed validator attesting records and and total epoch balances.
func ProcessRewardsAndPenaltiesPrecompute(state *pb.BeaconState, bp *Balance, vp []*Validator) (*pb.BeaconState, error) {
	// Can't process rewards and penalties in genesis epoch.
	if helpers.CurrentEpoch(state) == 0 {
		return state, nil
	}

	// Guard against an out-of-bounds using validator balance precompute.
	if len(vp) != len(state.Validators) || len(vp) != len(state.Balances) {
		return state, errors.New("precomputed registries not the same length as state registries")
	}

	attsRewards, attsPenalties, err := attestationDeltaPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	proposerRewards, err := proposerDeltaPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	clRewards, clPenalties, err := crosslinkDeltaPreCompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get crosslink delta")
	}

	for i := 0; i < len(state.Validators); i++ {
		state = helpers.IncreaseBalance(state, uint64(i), attsRewards[i]+clRewards[i]+proposerRewards[i])
		state = helpers.DecreaseBalance(state, uint64(i), attsPenalties[i]+clPenalties[i])
	}
	return state, nil
}

func attestationDeltaPrecompute(state *pb.BeaconState, bp *Balance, vp []*Validator) ([]uint64, []uint64, error) {
	prevEpoch := helpers.PrevEpoch(state)

	rewards := make([]uint64, len(state.Validators))
	penalties := make([]uint64, len(state.Validators))

	totalBalance := bp.CurrentEpoch
	totalAttested := bp.PrevEpochAttesters
	targetBalance := bp.PrevEpochTargetAttesters
	headBalance := bp.PrevEpochHeadAttesters

	for i, v := range vp {
		eligible := v.IsActivePrevEpoch || (v.IsSlashed && !v.IsWithdrawableCurrentEpoch)
		if !eligible {
			continue
		}

		vBalance := v.CurrentEpochEffectiveBalance
		baseReward := vBalance * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(totalBalance) / params.BeaconConfig().BaseRewardsPerEpoch

		if v.IsPrevEpochAttester && !v.IsSlashed {
			rewards[i] += baseReward * totalAttested / totalBalance
			proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient
			maxAtteserReward := baseReward - proposerReward
			slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
			rewards[i] += maxAtteserReward * (slotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay - v.InclusionDistance) / slotsPerEpoch
		} else {
			penalties[i] += baseReward
		}

		if v.IsPrevEpochTargetAttester && !v.IsSlashed {
			rewards[i] += baseReward * targetBalance / totalBalance
		} else {
			penalties[i] += baseReward
		}

		if v.IsPrevEpochHeadAttester && !v.IsSlashed {
			rewards[i] += baseReward * headBalance / totalBalance
		} else {
			penalties[i] += baseReward
		}

		finalityDelay := prevEpoch - state.FinalizedCheckpoint.Epoch
		if finalityDelay > params.BeaconConfig().MinEpochsToInactivityPenalty {
			penalties[i] += params.BeaconConfig().BaseRewardsPerEpoch * baseReward
			if !v.IsPrevEpochTargetAttester {
				penalties[i] += vBalance * finalityDelay / params.BeaconConfig().InactivityPenaltyQuotient
			}
		}
	}
	return rewards, penalties, nil
}

func proposerDeltaPrecompute(state *pb.BeaconState, bp *Balance, vp []*Validator) ([]uint64, error) {
	rewards := make([]uint64, len(state.Validators))

	totalBalance := bp.CurrentEpoch

	for i, v := range vp {
		if v.IsPrevEpochAttester {
			vBalance := v.CurrentEpochEffectiveBalance
			baseReward := vBalance * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(totalBalance) / params.BeaconConfig().BaseRewardsPerEpoch
			proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient
			rewards[i] += proposerReward
		}
	}
	return rewards, nil
}

func crosslinkDeltaPreCompute(state *pb.BeaconState, bp *Balance, vp []*Validator) ([]uint64, []uint64, error) {
	rewards := make([]uint64, len(state.Validators))
	penalties := make([]uint64, len(state.Validators))
	prevEpoch := helpers.PrevEpoch(state)
	count, err := helpers.CommitteeCount(state, prevEpoch)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get epoch committee count")
	}
	startShard, err := helpers.StartShard(state, prevEpoch)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get epoch start shard")
	}
	for i := uint64(0); i < count; i++ {
		shard := (startShard + i) % params.BeaconConfig().ShardCount
		committee, err := helpers.CrosslinkCommittee(state, prevEpoch, shard)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get crosslink's committee")
		}
		_, attestingIndices, err := epoch.WinningCrosslink(state, shard, prevEpoch)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get winning crosslink")
		}

		attested := make(map[uint64]bool)
		// Construct a map to look up validators that voted for crosslink.
		for _, index := range attestingIndices {
			attested[index] = true
		}
		committeeBalance := helpers.TotalBalance(state, committee)
		attestingBalance := helpers.TotalBalance(state, attestingIndices)

		for _, index := range committee {
			base := vp[i].CurrentEpochEffectiveBalance * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(bp.CurrentEpoch) / params.BeaconConfig().BaseRewardsPerEpoch
			if _, ok := attested[index]; ok {
				rewards[index] += base * attestingBalance / committeeBalance
			} else {
				penalties[index] += base
			}
		}
	}
	return rewards, penalties, nil
}
