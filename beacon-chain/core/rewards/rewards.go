package rewards

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/attestation"
	"go.opencensus.io/trace"
)

func AttestationsReward(
	ctx context.Context,
	beaconState state.BeaconState,
	b interfaces.ReadOnlySignedBeaconBlock,
) (state.BeaconState, uint64, error) {
	if err := consensusblocks.BeaconBlockIsNil(b); err != nil {
		return nil, 0, err
	}
	body := b.Block().Body()
	totalBalance, err := helpers.TotalActiveBalance(beaconState)
	if err != nil {
		return nil, 0, err
	}
	var totalReward uint64
	for idx, att := range body.Attestations() {
		var reward uint64
		beaconState, reward, err = AttestationReward(ctx, beaconState, att, totalBalance)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "could not get reward for attestation at index %d in block", idx)
		}
		totalReward += reward
	}
	return beaconState, totalReward, nil
}

func AttestationReward(
	ctx context.Context,
	beaconState state.BeaconState,
	att *ethpb.Attestation,
	totalBalance uint64,
) (state.BeaconState, uint64, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessAttestationNoVerifySignature")
	defer span.End()

	delay, err := beaconState.Slot().SafeSubSlot(att.Data.Slot)
	if err != nil {
		return nil, 0, fmt.Errorf("att slot %d can't be greater than state slot %d", att.Data.Slot, beaconState.Slot())
	}
	participatedFlags, err := altair.AttestationParticipationFlagIndices(beaconState, att.Data, delay)
	if err != nil {
		return nil, 0, err
	}
	committee, err := helpers.BeaconCommitteeFromState(ctx, beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return nil, 0, err
	}
	indices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		return nil, 0, err
	}

	return SetParticipationAndRewardProposer(ctx, beaconState, att.Data.Target.Epoch, indices, participatedFlags, totalBalance)
}

func SetParticipationAndRewardProposer(
	ctx context.Context,
	beaconState state.BeaconState,
	targetEpoch primitives.Epoch,
	indices []uint64,
	participatedFlags map[uint8]bool, totalBalance uint64) (state.BeaconState, uint64, error) {
	var proposerRewardNumerator uint64
	currentEpoch := time.CurrentEpoch(beaconState)
	var stateErr error
	if targetEpoch == currentEpoch {
		stateErr = beaconState.ModifyCurrentParticipationBits(func(val []byte) ([]byte, error) {
			propRewardNum, epochParticipation, err := altair.EpochParticipation(beaconState, indices, val, participatedFlags, totalBalance)
			if err != nil {
				return nil, err
			}
			proposerRewardNumerator = propRewardNum
			return epochParticipation, nil
		})
	} else {
		stateErr = beaconState.ModifyPreviousParticipationBits(func(val []byte) ([]byte, error) {
			propRewardNum, epochParticipation, err := altair.EpochParticipation(beaconState, indices, val, participatedFlags, totalBalance)
			if err != nil {
				return nil, err
			}
			proposerRewardNumerator = propRewardNum
			return epochParticipation, nil
		})
	}
	if stateErr != nil {
		return nil, 0, stateErr
	}

	reward, err := RewardProposer(ctx, beaconState, proposerRewardNumerator)
	if err != nil {
		return nil, 0, err
	}

	return beaconState, reward, nil
}

func RewardProposer(ctx context.Context, beaconState state.BeaconState, proposerRewardNumerator uint64) (uint64, error) {
	cfg := params.BeaconConfig()
	d := (cfg.WeightDenominator - cfg.ProposerWeight) * cfg.WeightDenominator / cfg.ProposerWeight
	proposerReward := proposerRewardNumerator / d
	i, err := helpers.BeaconProposerIndex(ctx, beaconState)
	if err != nil {
		return 0, err
	}
	if err := helpers.IncreaseBalance(beaconState, i, proposerReward); err != nil {
		return 0, err
	}
	return proposerReward, nil
}
