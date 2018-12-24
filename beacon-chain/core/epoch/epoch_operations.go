package epoch

import (
	"bytes"
	"fmt"

	block "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	b "github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Attestations returns the pending attestations of slots in the epoch
// (state.slot-EPOCH_LENGTH...state.slot-1), not attestations that got
// included in the chain during the epoch.
//
// Spec pseudocode definition:
//   return [a for a in state.latest_attestations
//   if state.slot - EPOCH_LENGTH <= a.data.slot < state.slot]
func Attestations(state *pb.BeaconState) []*pb.PendingAttestationRecord {
	epochLength := params.BeaconConfig().EpochLength
	var thisEpochAttestations []*pb.PendingAttestationRecord
	var earliestSlot uint64

	for _, attestation := range state.LatestAttestations {

		// If the state slot is less than epochLength, then the earliestSlot would
		// result in a negative number. Therefore we should default to
		// earliestSlot = 0 in this case.
		if state.Slot > epochLength {
			earliestSlot = state.Slot - epochLength
		}

		if earliestSlot <= attestation.GetData().GetSlot() && attestation.GetData().GetSlot() < state.Slot {
			thisEpochAttestations = append(thisEpochAttestations, attestation)
		}
	}
	return thisEpochAttestations
}

// BoundaryAttestations returns the pending attestations from
// the epoch's boundary block.
//
// Spec pseudocode definition:
//   return [a for a in this_epoch_attestations if a.data.epoch_boundary_root ==
//   get_block_root(state, state.slot-EPOCH_LENGTH) and a.justified_slot ==
//   state.justified_slot]
func BoundaryAttestations(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
) ([]*pb.PendingAttestationRecord, error) {
	epochLength := params.BeaconConfig().EpochLength
	var boundaryAttestations []*pb.PendingAttestationRecord

	for _, attestation := range thisEpochAttestations {

		boundaryBlockRoot, err := block.BlockRoot(state, state.Slot-epochLength)
		if err != nil {
			return nil, err
		}

		attestationData := attestation.GetData()
		sameRoot := bytes.Equal(attestationData.JustifiedBlockRootHash32, boundaryBlockRoot)
		sameSlotNum := attestationData.JustifiedSlot == state.JustifiedSlot
		if sameRoot && sameSlotNum {
			boundaryAttestations = append(boundaryAttestations, attestation)
		}
	}
	return boundaryAttestations, nil
}

// PrevAttestations returns the attestations of the previous epoch
// (state.slot - 2 * EPOCH_LENGTH...state.slot - EPOCH_LENGTH).
//
// Spec pseudocode definition:
//   return [a for a in state.latest_attestations
//   if state.slot - 2 * EPOCH_LENGTH <= a.slot < state.slot - EPOCH_LENGTH]
func PrevAttestations(state *pb.BeaconState) []*pb.PendingAttestationRecord {
	epochLength := params.BeaconConfig().EpochLength
	var prevEpochAttestations []*pb.PendingAttestationRecord
	var earliestSlot uint64

	for _, attestation := range state.LatestAttestations {

		// If the state slot is less than 2 * epochLength, then the earliestSlot would
		// result in a negative number. Therefore we should default to
		// earliestSlot = 0 in this case.
		if state.Slot > 2*epochLength {
			earliestSlot = state.Slot - 2*epochLength
		}

		if earliestSlot <= attestation.GetData().GetSlot() &&
			attestation.GetData().GetSlot() < state.Slot-epochLength {
			prevEpochAttestations = append(prevEpochAttestations, attestation)
		}
	}
	return prevEpochAttestations
}

// PrevJustifiedAttestations returns the justified attestations
// of the previous 2 epochs.
//
// Spec pseudocode definition:
//   return [a for a in this_epoch_attestations + previous_epoch_attestations
//   if a.justified_slot == state.previous_justified_slot]
func PrevJustifiedAttestations(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord,
) []*pb.PendingAttestationRecord {

	var prevJustifiedAttestations []*pb.PendingAttestationRecord
	epochAttestations := append(thisEpochAttestations, prevEpochAttestations...)

	for _, attestation := range epochAttestations {
		if attestation.GetData().GetJustifiedSlot() == state.PreviousJustifiedSlot {
			prevJustifiedAttestations = append(prevJustifiedAttestations, attestation)
		}
	}
	return prevJustifiedAttestations
}

// PrevHeadAttestations returns the pending attestations from
// the canonical beacon chain.
//
// Spec pseudocode definition:
//   return [a for a in previous_epoch_attestations
//   if a.beacon_block_root == get_block_root(state, a.slot)]
func PrevHeadAttestations(
	state *pb.BeaconState,
	prevEpochAttestations []*pb.PendingAttestationRecord,
) ([]*pb.PendingAttestationRecord, error) {

	var headAttestations []*pb.PendingAttestationRecord
	for _, attestation := range prevEpochAttestations {
		canonicalBlockRoot, err := block.BlockRoot(state, attestation.GetData().GetSlot())
		if err != nil {
			return nil, err
		}

		attestationData := attestation.GetData()
		if bytes.Equal(attestationData.BeaconBlockRootHash32, canonicalBlockRoot) {
			headAttestations = append(headAttestations, attestation)
		}
	}
	return headAttestations, nil
}

// WinningRoot returns the shard block root with the most combined validator
// effective balance. The ties broken by favoring lower shard block root values.
//
// Spec pseudocode definition:
//   Let winning_root(shard_committee) be equal to the value of shard_block_root
//   such that sum([get_effective_balance(state, i)
//   for i in attesting_validator_indices(shard_committee, shard_block_root)])
//   is maximized (ties broken by favoring lower shard_block_root values)
func WinningRoot(
	state *pb.BeaconState,
	shardCommittee *pb.ShardAndCommittee,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) ([]byte, error) {

	var winnerBalance uint64
	var winnerRoot []byte
	var candidateRoots [][]byte
	attestations := append(thisEpochAttestations, prevEpochAttestations...)

	for _, attestation := range attestations {
		if attestation.GetData().GetShard() == shardCommittee.Shard {
			candidateRoots = append(candidateRoots, attestation.GetData().ShardBlockRootHash32)
		}
	}

	for _, candidateRoot := range candidateRoots {
		indices, err := validators.AttestingValidatorIndices(
			state,
			shardCommittee,
			candidateRoot,
			thisEpochAttestations,
			prevEpochAttestations)
		if err != nil {
			return nil, fmt.Errorf("could not get attesting validator indices: %v", err)
		}

		var rootBalance uint64
		for _, index := range indices {
			rootBalance += validators.EffectiveBalance(state, index)
		}

		if rootBalance > winnerBalance ||
			(rootBalance == winnerBalance && b.LowerThan(candidateRoot, winnerRoot)) {
			winnerBalance = rootBalance
			winnerRoot = candidateRoot
		}
	}
	return winnerRoot, nil
}

// AttestingValidators returns the validators of the winning root.
//
// Spec pseudocode definition:
//    Let `attesting_validators(shard_committee)` be equal to
//    `attesting_validator_indices(shard_committee, winning_root(shard_committee))` for convenience
func AttestingValidators(
	state *pb.BeaconState,
	shardCommittee *pb.ShardAndCommittee,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) ([]uint32, error) {

	root, err := WinningRoot(
		state,
		shardCommittee,
		thisEpochAttestations,
		prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get winning root: %v", err)
	}

	indices, err := validators.AttestingValidatorIndices(
		state,
		shardCommittee,
		root,
		thisEpochAttestations,
		prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get attesting validator indices: %v", err)
	}

	return indices, nil
}

// TotalAttestingBalance returns the total balance at stake of the validators
// attested to the winning root.
//
// Spec pseudocode definition:
//    Let total_balance(shard_committee) =
//    sum([get_effective_balance(state, i) for i in shard_committee.committee])
func TotalAttestingBalance(
	state *pb.BeaconState,
	shardCommittee *pb.ShardAndCommittee,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) (uint64, error) {

	var totalBalance uint64
	attestedValidatorIndices, err := AttestingValidators(state, shardCommittee, thisEpochAttestations, prevEpochAttestations)
	if err != nil {
		return 0, fmt.Errorf("could not get attesting validator indices: %v", err)
	}

	for _, index := range attestedValidatorIndices {
		totalBalance += validators.EffectiveBalance(state, index)
	}

	return totalBalance, nil
}

// TotalBalance returns the total balance at stake of the validators
// from the shard committee regardless of validators attested or not.
//
// Spec pseudocode definition:
//    Let total_balance(shard_committee) =
//    sum([get_effective_balance(state, i) for i in shard_committee.committee])
func TotalBalance(
	state *pb.BeaconState,
	shardCommittee *pb.ShardAndCommittee) uint64 {

	var totalBalance uint64
	for _, index := range shardCommittee.Committee {
		totalBalance += validators.EffectiveBalance(state, index)
	}

	return totalBalance
}

// InclusionSlot returns the slot number of when the validator's
// attestation gets included in the beacon chain.
//
// Spec pseudocode definition:
//    Let inclusion_slot(state, index) =
//    a.slot_included for the attestation a where index is in
//    get_attestation_participants(state, a.data, a.participation_bitfield)
func InclusionSlot(state *pb.BeaconState, validatorIndex uint32) (uint64, error) {

	for _, attestation := range state.LatestAttestations {
		participatedValidators, err := validators.AttestationParticipants(state, attestation.GetData(), attestation.ParticipationBitfield)
		if err != nil {
			return 0, fmt.Errorf("could not get attestation participants: %v", err)
		}

		for _, index := range participatedValidators {
			if index == validatorIndex {
				return attestation.SlotIncluded, nil
			}
		}
	}
	return 0, fmt.Errorf("could not find inclusion slot for validator index %d", validatorIndex)
}

// InclusionDistance returns the difference in slot number of when attestation
// gets submitted and when it gets included.
//
// Spec pseudocode definition:
//    Let inclusion_distance(state, index) =
//    a.slot_included - a.data.slot where a is the above attestation same as
//    inclusion_slot
func InclusionDistance(state *pb.BeaconState, validatorIndex uint32) (uint64, error) {

	for _, attestation := range state.LatestAttestations {
		participatedValidators, err := validators.AttestationParticipants(state, attestation.GetData(), attestation.ParticipationBitfield)
		if err != nil {
			return 0, fmt.Errorf("could not get attestation participants: %v", err)
		}

		for _, index := range participatedValidators {
			if index == validatorIndex {
				return attestation.SlotIncluded - attestation.GetData().GetSlot(), nil
			}
		}
	}
	return 0, fmt.Errorf("could not find inclusion distance for validator index %d", validatorIndex)
}

// AdjustForInclusionDistance returns the calculated reward based on
// how long it took for attestation to get included. The longer, the lower
// the the reward.
//
// Spec pseudocode definition:
//    def adjust_for_inclusion_distance(magnitude: int, distance: int) -> int:
//    """
//    Adjusts the reward of an attestation based on how long it took to get included
//    (the longer, the lower the reward). Returns a value between ``0`` and ``magnitude``.
//    ""
//    return magnitude // 2 + (magnitude // 2) * MIN_ATTESTATION_INCLUSION_DELAY // distance
func AdjustForInclusionDistance(magniture uint64, distance uint64) uint64 {
	return magniture/2 + (magniture/2)*
		params.BeaconConfig().MinAttestationInclusionDelay/distance
}

// SinceFinality calculates and returns how many epoch has it been since
// a finalized slot.
//
// Spec pseudocode definition:
//    epochs_since_finality = (state.slot - state.finalized_slot) // EPOCH_LENGTH
func SinceFinality(state *pb.BeaconState) uint64 {
	return state.Slot - state.FinalizedSlot/params.BeaconConfig().EpochLength
}
