package altair

import (
	"bytes"

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
