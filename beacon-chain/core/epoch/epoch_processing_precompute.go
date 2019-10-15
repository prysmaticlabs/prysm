package epoch

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessJustificationAndFinalization processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
// This is an optimized version by passing in precomputed attested and total epoch balances.
func ProcessJustificationAndFinalizationPreCompute(state *pb.BeaconState, p *BalancePrecompute) (*pb.BeaconState, error) {
	if state.Slot <= helpers.StartSlot(2) {
		return state, nil
	}

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	oldPrevJustifiedCheckpoint := state.PreviousJustifiedCheckpoint
	oldCurrJustifiedCheckpoint := state.CurrentJustifiedCheckpoint

	// Process justifications
	state.PreviousJustifiedCheckpoint = state.CurrentJustifiedCheckpoint
	state.JustificationBits.Shift(1)

	// Note: the spec refers to the bit index position starting at 1 instead of starting at zero.
	// We will use that paradigm here for consistency with the godoc spec definition.

	// If 2/3 or more of total balance attested in the previous epoch.
	if 3*p.PrevEpochTargetAttesters >= 2*p.CurrentEpoch {
		blockRoot, err := helpers.BlockRoot(state, prevEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for previous epoch %d", prevEpoch)
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{Epoch: prevEpoch, Root: blockRoot}
		state.JustificationBits.SetBitAt(1, true)
	}

	// If 2/3 or more of the total balance attested in the current epoch.
	if 3*p.CurrentEpochTargetAttesters >= 2*p.CurrentEpoch {
		blockRoot, err := helpers.BlockRoot(state, currentEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for current epoch %d", prevEpoch)
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{Epoch: currentEpoch, Root: blockRoot}
		state.JustificationBits.SetBitAt(0, true)
	}

	// Process finalization according to ETH2.0 specifications.
	justification := state.JustificationBits.Bytes()[0]

	// 2nd/3rd/4th (0b1110) most recent epochs are justified, the 2nd using the 4th as source.
	if justification&0x0E == 0x0E && (oldPrevJustifiedCheckpoint.Epoch+3) == currentEpoch {
		state.FinalizedCheckpoint = oldPrevJustifiedCheckpoint
	}

	// 2nd/3rd (0b0110) most recent epochs are justified, the 2nd using the 3rd as source.
	if justification&0x06 == 0x06 && (oldPrevJustifiedCheckpoint.Epoch+2) == currentEpoch {
		state.FinalizedCheckpoint = oldPrevJustifiedCheckpoint
	}

	// 1st/2nd/3rd (0b0111) most recent epochs are justified, the 1st using the 3rd as source.
	if justification&0x07 == 0x07 && (oldCurrJustifiedCheckpoint.Epoch+2) == currentEpoch {
		state.FinalizedCheckpoint = oldCurrJustifiedCheckpoint
	}

	// The 1st/2nd (0b0011) most recent epochs are justified, the 1st using the 2nd as source
	if justification&0x03 == 0x03 && (oldCurrJustifiedCheckpoint.Epoch+1) == currentEpoch {
		state.FinalizedCheckpoint = oldCurrJustifiedCheckpoint
	}

	return state, nil
}

// ProcessRewardsAndPenalties processes the rewards and penalties of individual validator.
// This is an optimized version by passing in precomputed validator attesting records and and total epoch balances.
func ProcessRewardsAndPenaltiesPrecompute(state *pb.BeaconState, bp *BalancePrecompute, vp []*ValidatorPrecompute) (*pb.BeaconState, error) {
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

// ProcessSlashings processes the slashed validators during epoch processing.
// This is an optimized version by passing in precomputed total epoch balances.
func ProcessSlashingsPrecompute(state *pb.BeaconState, p *BalancePrecompute) *pb.BeaconState {
	currentEpoch := helpers.CurrentEpoch(state)

	// Compute slashed balances in the current epoch
	exitLength := params.BeaconConfig().EpochsPerSlashingsVector

	// Compute the sum of state slashings
	totalSlashing := uint64(0)
	for _, slashing := range state.Slashings {
		totalSlashing += slashing
	}

	// Compute slashing for each validator.
	for index, validator := range state.Validators {
		correctEpoch := (currentEpoch + exitLength/2) == validator.WithdrawableEpoch
		if validator.Slashed && correctEpoch {
			minSlashing := mathutil.Min(totalSlashing*3, p.CurrentEpoch)
			increment := params.BeaconConfig().EffectiveBalanceIncrement
			penaltyNumerator := validator.EffectiveBalance / increment * minSlashing
			penalty := penaltyNumerator / p.CurrentEpoch * increment
			state = helpers.DecreaseBalance(state, uint64(index), penalty)
		}
	}
	return state
}

func attestationDeltaPrecompute(state *pb.BeaconState, bp *BalancePrecompute, vp []*ValidatorPrecompute) ([]uint64, []uint64, error) {
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

		if v.IsHeadAttester && !v.IsSlashed {
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

func proposerDeltaPrecompute(state *pb.BeaconState, bp *BalancePrecompute, vp []*ValidatorPrecompute) ([]uint64, error) {
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
