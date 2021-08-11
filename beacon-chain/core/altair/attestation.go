package altair

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// HasValidatorFlag returns true if the flag at position has set.
func HasValidatorFlag(flag, flagPosition uint8) bool {
	return ((flag >> flagPosition) & 1) == 1
}

// AddValidatorFlag adds new validator flag to existing one.
func AddValidatorFlag(flag, flagPosition uint8) uint8 {
	return flag | (1 << flagPosition)
}

// EpochParticipation sets and returns the proposer reward numerator and epoch participation.
//
// Spec code:
//    proposer_reward_numerator = 0
//    for index in get_attesting_indices(state, data, attestation.aggregation_bits):
//        for flag_index, weight in enumerate(PARTICIPATION_FLAG_WEIGHTS):
//            if flag_index in participation_flag_indices and not has_flag(epoch_participation[index], flag_index):
//                epoch_participation[index] = add_flag(epoch_participation[index], flag_index)
//                proposer_reward_numerator += get_base_reward(state, index) * weight
func EpochParticipation(beaconState state.BeaconState, indices []uint64, epochParticipation []byte, participatedFlags map[uint8]bool) (uint64, []byte, error) {
	cfg := params.BeaconConfig()
	sourceFlagIndex := cfg.TimelySourceFlagIndex
	targetFlagIndex := cfg.TimelyTargetFlagIndex
	headFlagIndex := cfg.TimelyHeadFlagIndex
	proposerRewardNumerator := uint64(0)
	totalBalance, err := helpers.TotalActiveBalance(beaconState)
	if err != nil {
		return 0, nil, err
	}
	for _, index := range indices {
		if index >= uint64(len(epochParticipation)) {
			return 0, nil, fmt.Errorf("index %d exceeds participation length %d", index, len(epochParticipation))
		}
		br, err := BaseRewardWithTotalBalance(beaconState, types.ValidatorIndex(index), totalBalance)
		if err != nil {
			return 0, nil, err
		}
		if participatedFlags[sourceFlagIndex] && !HasValidatorFlag(epochParticipation[index], sourceFlagIndex) {
			epochParticipation[index] = AddValidatorFlag(epochParticipation[index], sourceFlagIndex)
			proposerRewardNumerator += br * cfg.TimelySourceWeight
		}
		if participatedFlags[targetFlagIndex] && !HasValidatorFlag(epochParticipation[index], targetFlagIndex) {
			epochParticipation[index] = AddValidatorFlag(epochParticipation[index], targetFlagIndex)
			proposerRewardNumerator += br * cfg.TimelyTargetWeight
		}
		if participatedFlags[headFlagIndex] && !HasValidatorFlag(epochParticipation[index], headFlagIndex) {
			epochParticipation[index] = AddValidatorFlag(epochParticipation[index], headFlagIndex)
			proposerRewardNumerator += br * cfg.TimelyHeadWeight
		}
	}

	return proposerRewardNumerator, epochParticipation, nil
}

// RewardProposer rewards proposer by increasing proposer's balance with input reward numerator and calculated reward denominator.
//
// Spec code:
//    proposer_reward_denominator = (WEIGHT_DENOMINATOR - PROPOSER_WEIGHT) * WEIGHT_DENOMINATOR // PROPOSER_WEIGHT
//    proposer_reward = Gwei(proposer_reward_numerator // proposer_reward_denominator)
//    increase_balance(state, get_beacon_proposer_index(state), proposer_reward)
func RewardProposer(beaconState state.BeaconState, proposerRewardNumerator uint64) error {
	cfg := params.BeaconConfig()
	d := (cfg.WeightDenominator - cfg.ProposerWeight) * cfg.WeightDenominator / cfg.ProposerWeight
	proposerReward := proposerRewardNumerator / d
	i, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return err
	}

	return helpers.IncreaseBalance(beaconState, i, proposerReward)
}

// AttestationParticipationFlagIndices retrieves a map of attestation scoring based on Altair's participation flag indices.
// This is used to facilitate process attestation during state transition and during upgrade to altair state.
//
// Spec code:
// def get_attestation_participation_flag_indices(state: BeaconState,
//                                               data: AttestationData,
//                                               inclusion_delay: uint64) -> Sequence[int]:
//    """
//    Return the flag indices that are satisfied by an attestation.
//    """
//    if data.target.epoch == get_current_epoch(state):
//        justified_checkpoint = state.current_justified_checkpoint
//    else:
//        justified_checkpoint = state.previous_justified_checkpoint
//
//    # Matching roots
//    is_matching_source = data.source == justified_checkpoint
//    is_matching_target = is_matching_source and data.target.root == get_block_root(state, data.target.epoch)
//    is_matching_head = is_matching_target and data.beacon_block_root == get_block_root_at_slot(state, data.slot)
//    assert is_matching_source
//
//    participation_flag_indices = []
//    if is_matching_source and inclusion_delay <= integer_squareroot(SLOTS_PER_EPOCH):
//        participation_flag_indices.append(TIMELY_SOURCE_FLAG_INDEX)
//    if is_matching_target and inclusion_delay <= SLOTS_PER_EPOCH:
//        participation_flag_indices.append(TIMELY_TARGET_FLAG_INDEX)
//    if is_matching_head and inclusion_delay == MIN_ATTESTATION_INCLUSION_DELAY:
//        participation_flag_indices.append(TIMELY_HEAD_FLAG_INDEX)
//
//    return participation_flag_indices
func AttestationParticipationFlagIndices(beaconState state.BeaconStateAltair, data *ethpb.AttestationData, delay types.Slot) (map[uint8]bool, error) {
	currEpoch := helpers.CurrentEpoch(beaconState)
	var justifiedCheckpt *ethpb.Checkpoint
	if data.Target.Epoch == currEpoch {
		justifiedCheckpt = beaconState.CurrentJustifiedCheckpoint()
	} else {
		justifiedCheckpt = beaconState.PreviousJustifiedCheckpoint()
	}

	matchedSrc, matchedTgt, matchedHead, err := MatchingStatus(beaconState, data, justifiedCheckpt)
	if err != nil {
		return nil, err
	}
	if !matchedSrc {
		return nil, errors.New("source epoch does not match")
	}

	participatedFlags := make(map[uint8]bool)
	cfg := params.BeaconConfig()
	sourceFlagIndex := cfg.TimelySourceFlagIndex
	targetFlagIndex := cfg.TimelyTargetFlagIndex
	headFlagIndex := cfg.TimelyHeadFlagIndex
	slotsPerEpoch := cfg.SlotsPerEpoch
	sqtRootSlots := cfg.SqrRootSlotsPerEpoch
	if matchedSrc && delay <= sqtRootSlots {
		participatedFlags[sourceFlagIndex] = true
	}
	matchedSrcTgt := matchedSrc && matchedTgt
	if matchedSrcTgt && delay <= slotsPerEpoch {
		participatedFlags[targetFlagIndex] = true
	}
	matchedSrcTgtHead := matchedHead && matchedSrcTgt
	if matchedSrcTgtHead && delay == cfg.MinAttestationInclusionDelay {
		participatedFlags[headFlagIndex] = true
	}
	return participatedFlags, nil
}

// MatchingStatus returns the matching statues for attestation data's source target and head.
//
// Spec code:
//    is_matching_source = data.source == justified_checkpoint
//    is_matching_target = is_matching_source and data.target.root == get_block_root(state, data.target.epoch)
//    is_matching_head = is_matching_target and data.beacon_block_root == get_block_root_at_slot(state, data.slot)
func MatchingStatus(beaconState state.BeaconState, data *ethpb.AttestationData, cp *ethpb.Checkpoint) (matchedSrc bool, matchedTgt bool, matchedHead bool, err error) {
	matchedSrc = attestationutil.CheckPointIsEqual(data.Source, cp)

	r, err := helpers.BlockRoot(beaconState, data.Target.Epoch)
	if err != nil {
		return false, false, false, err
	}
	matchedTgt = bytes.Equal(r, data.Target.Root)

	r, err = helpers.BlockRootAtSlot(beaconState, data.Slot)
	if err != nil {
		return false, false, false, err
	}
	matchedHead = bytes.Equal(r, data.BeaconBlockRoot)
	return
}
