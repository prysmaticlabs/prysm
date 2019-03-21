// Package epoch contains epoch processing libraries. These libraries
// process new balance for the validators, justify and finalize new
// check points, shuffle and reassign validators to different slots and
// shards.
package epoch

import (
	"bytes"
	"context"
	"fmt"
	"math"

	block "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	b "github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

// CurrentAttestations returns the pending attestations from current epoch.
//
// Spec pseudocode definition:
//   return [a for a in state.latest_attestations if current_epoch ==
//   	slot_to_epoch(a.data.slot)]
//  (Note: this is the set of attestations of slots in the epoch
//  current_epoch, not attestations that got included in the chain
//  during the epoch current_epoch.)
func CurrentAttestations(ctx context.Context, state *pb.BeaconState) []*pb.PendingAttestation {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.CurrentAttestations")
	defer span.End()

	var currentEpochAttestations []*pb.PendingAttestation
	currentEpoch := helpers.CurrentEpoch(state)

	for _, attestation := range state.LatestAttestations {
		if currentEpoch == helpers.SlotToEpoch(attestation.Data.Slot) {
			currentEpochAttestations = append(currentEpochAttestations, attestation)
		}
	}
	return currentEpochAttestations
}

// CurrentEpochBoundaryAttestations returns the pending attestations from
// the epoch's boundary block.
//
// Spec pseudocode definition:
//   return [a for a in current_epoch_attestations if a.data.epoch_boundary_root ==
//   	get_block_root(state, get_epoch_start_slot(current_epoch))
func CurrentEpochBoundaryAttestations(
	ctx context.Context,
	state *pb.BeaconState,
	currentEpochAttestations []*pb.PendingAttestation,
) ([]*pb.PendingAttestation, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.CurrentEpochBoundaryAttestations")
	defer span.End()

	var boundaryAttestations []*pb.PendingAttestation
	for _, attestation := range currentEpochAttestations {
		boundaryBlockRoot, err := block.BlockRoot(state, helpers.StartSlot(helpers.CurrentEpoch(state)))
		if err != nil {
			return nil, err
		}

		attestationData := attestation.Data
		sameRoot := bytes.Equal(attestationData.EpochBoundaryRootHash32, boundaryBlockRoot)
		if sameRoot {
			boundaryAttestations = append(boundaryAttestations, attestation)
		}
	}
	return boundaryAttestations, nil
}

// PrevAttestations returns the attestations of the previous epoch
// (state.slot - 2 * SLOTS_PER_EPOCH...state.slot - EPOCH_LENGTH).
//
// Spec pseudocode definition:
//   return [a for a in state.latest_attestations if
//   	previous_epoch == slot_to_epoch(a.data.slot)].
func PrevAttestations(ctx context.Context, state *pb.BeaconState) []*pb.PendingAttestation {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.PrevAttestations")
	defer span.End()

	var prevEpochAttestations []*pb.PendingAttestation
	prevEpoch := helpers.PrevEpoch(state)

	for _, attestation := range state.LatestAttestations {
		if prevEpoch == helpers.SlotToEpoch(attestation.Data.Slot) {
			prevEpochAttestations = append(prevEpochAttestations, attestation)
		}
	}

	return prevEpochAttestations
}

// PrevJustifiedAttestations returns the justified attestations
// of the previous 2 epochs.
//
// Spec pseudocode definition:
//   return [a for a in current_epoch_attestations + previous_epoch_attestations
//   if a.data.justified_epoch  == state.previous_justified_epoch]
func PrevJustifiedAttestations(
	ctx context.Context,
	state *pb.BeaconState,
	currentEpochAttestations []*pb.PendingAttestation,
	prevEpochAttestations []*pb.PendingAttestation,
) []*pb.PendingAttestation {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.PrevJustifiedAttestations")
	defer span.End()

	var prevJustifiedAttestations []*pb.PendingAttestation
	epochAttestations := append(currentEpochAttestations, prevEpochAttestations...)

	for _, attestation := range epochAttestations {
		if attestation.Data.JustifiedEpoch == state.PreviousJustifiedEpoch {
			prevJustifiedAttestations = append(prevJustifiedAttestations, attestation)
		}
	}
	return prevJustifiedAttestations
}

// PrevEpochBoundaryAttestations returns the boundary attestations
// at the start of the previous epoch.
//
// Spec pseudocode definition:
//   return [a for a in previous_epoch_attestations
// 	 if a.epoch_boundary_root == get_block_root(state, get_epoch_start_slot(previous_epoch)]
func PrevEpochBoundaryAttestations(
	ctx context.Context,
	state *pb.BeaconState,
	prevEpochAttestations []*pb.PendingAttestation,
) ([]*pb.PendingAttestation, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.PrevEpochBoundaryAttestations")
	defer span.End()

	var prevEpochBoundaryAttestations []*pb.PendingAttestation

	prevBoundaryBlockRoot, err := block.BlockRoot(state,
		helpers.StartSlot(helpers.PrevEpoch(state)))
	if err != nil {
		return nil, err
	}

	for _, attestation := range prevEpochAttestations {
		if bytes.Equal(attestation.Data.EpochBoundaryRootHash32, prevBoundaryBlockRoot) {
			prevEpochBoundaryAttestations = append(prevEpochBoundaryAttestations, attestation)
		}
	}
	return prevEpochBoundaryAttestations, nil
}

// PrevHeadAttestations returns the pending attestations from
// the canonical beacon chain.
//
// Spec pseudocode definition:
//   return [a for a in previous_epoch_attestations
//   if a.data.beacon_block_root == get_block_root(state, a.data.slot)]
func PrevHeadAttestations(
	ctx context.Context,
	state *pb.BeaconState,
	prevEpochAttestations []*pb.PendingAttestation,
) ([]*pb.PendingAttestation, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.PrevHeadAttestations")
	defer span.End()

	var headAttestations []*pb.PendingAttestation
	for _, attestation := range prevEpochAttestations {
		canonicalBlockRoot, err := block.BlockRoot(state, attestation.Data.Slot)
		if err != nil {
			return nil, err
		}

		attestationData := attestation.Data
		if bytes.Equal(attestationData.BeaconBlockRootHash32, canonicalBlockRoot) {
			headAttestations = append(headAttestations, attestation)
		}
	}
	return headAttestations, nil
}

// TotalBalance returns the total balance at stake of the validators
// from the shard committee regardless of validators attested or not.
//
// Spec pseudocode definition:
//    def get_total_balance(state: BeaconState, validators: List[ValidatorIndex]) -> Gwei:
//    """
//    Return the combined effective balance of an array of validators.
//    """
//    return sum([get_effective_balance(state, i) for i in validators])
func TotalBalance(
	ctx context.Context,
	state *pb.BeaconState,
	activeValidatorIndices []uint64) uint64 {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.TotalBalance")
	defer span.End()

	var totalBalance uint64
	for _, index := range activeValidatorIndices {
		totalBalance += helpers.EffectiveBalance(state, index)
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
//    If multiple attestations are applicable, the attestation with
//    lowest `slot_included` is considered.
func InclusionSlot(state *pb.BeaconState, validatorIndex uint64) (uint64, error) {
	lowestSlotIncluded := uint64(math.MaxUint64)
	for _, attestation := range state.LatestAttestations {
		participatedValidators, err := helpers.AttestationParticipants(state, attestation.Data, attestation.AggregationBitfield)
		if err != nil {
			return 0, fmt.Errorf("could not get attestation participants: %v", err)
		}
		for _, index := range participatedValidators {
			if index == validatorIndex {
				if attestation.InclusionSlot < lowestSlotIncluded {
					lowestSlotIncluded = attestation.InclusionSlot
				}
			}
		}
	}
	if lowestSlotIncluded == math.MaxUint64 {
		return 0, fmt.Errorf("could not find inclusion slot for validator index %d", validatorIndex)
	}
	return lowestSlotIncluded, nil
}

// InclusionDistance returns the difference in slot number of when attestation
// gets submitted and when it gets included.
//
// Spec pseudocode definition:
//    Let inclusion_distance(state, index) =
//    a.slot_included - a.data.slot where a is the above attestation same as
//    inclusion_slot.
func InclusionDistance(state *pb.BeaconState, validatorIndex uint64) (uint64, error) {

	for _, attestation := range state.LatestAttestations {
		participatedValidators, err := helpers.AttestationParticipants(state, attestation.Data, attestation.AggregationBitfield)
		if err != nil {
			return 0, fmt.Errorf("could not get attestation participants: %v", err)
		}
		for _, index := range participatedValidators {
			if index == validatorIndex {
				return attestation.InclusionSlot - attestation.Data.Slot, nil
			}
		}
	}
	return 0, fmt.Errorf("could not find inclusion distance for validator index %d", validatorIndex)
}

// AttestingValidators returns the validators of the winning root.
//
// Spec pseudocode definition:
//    Let `attesting_validators(crosslink_committee)` be equal to
//    `attesting_validator_indices(crosslink_committee, winning_root(crosslink_committee))` for convenience
func AttestingValidators(
	ctx context.Context,
	state *pb.BeaconState,
	shard uint64,
	currentEpochAttestations []*pb.PendingAttestation,
	prevEpochAttestations []*pb.PendingAttestation) ([]uint64, error) {

	root, err := winningRoot(
		ctx,
		state,
		shard,
		currentEpochAttestations,
		prevEpochAttestations)
	if err != nil {
		return nil, fmt.Errorf("could not get winning root: %v", err)
	}

	indices, err := validators.AttestingValidatorIndices(
		state,
		shard,
		root,
		currentEpochAttestations,
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
//    Let total_balance(crosslink_committee) =
//    sum([get_effective_balance(state, i) for i in crosslink_committee.committee])
func TotalAttestingBalance(
	ctx context.Context,
	state *pb.BeaconState,
	shard uint64,
	currentEpochAttestations []*pb.PendingAttestation,
	prevEpochAttestations []*pb.PendingAttestation) (uint64, error) {

	var totalBalance uint64
	attestedValidatorIndices, err := AttestingValidators(ctx, state, shard, currentEpochAttestations, prevEpochAttestations)
	if err != nil {
		return 0, fmt.Errorf("could not get attesting validator indices: %v", err)
	}

	for _, index := range attestedValidatorIndices {
		totalBalance += helpers.EffectiveBalance(state, index)
	}

	return totalBalance, nil
}

// SinceFinality calculates and returns how many epoch has it been since
// a finalized slot.
//
// Spec pseudocode definition:
//    epochs_since_finality = next_epoch - state.finalized_epoch
func SinceFinality(state *pb.BeaconState) uint64 {
	return helpers.NextEpoch(state) - state.FinalizedEpoch
}

// winningRoot returns the shard block root with the most combined validator
// effective balance. The ties broken by favoring lower shard block root values.
//
// Spec pseudocode definition:
//   Let winning_root(crosslink_committee) be equal to the value of crosslink_data_root
//   such that get_total_balance(state, attesting_validator_indices(crosslink_committee, crosslink_data_root))
//   is maximized (ties broken by favoring lexicographically smallest crosslink_data_root).
func winningRoot(
	ctx context.Context,
	state *pb.BeaconState,
	shard uint64,
	currentEpochAttestations []*pb.PendingAttestation,
	prevEpochAttestations []*pb.PendingAttestation) ([]byte, error) {

	ctx, span := trace.StartSpan(ctx, "beacon-chain.ChainService.state.ProcessEpoch.CalculateWinningRoot")
	defer span.End()

	var winnerBalance uint64
	var winnerRoot []byte
	var candidateRoots [][]byte
	attestations := append(currentEpochAttestations, prevEpochAttestations...)

	for _, attestation := range attestations {
		if attestation.Data.Shard == shard {
			candidateRoots = append(candidateRoots, attestation.Data.CrosslinkDataRootHash32)
		}
	}

	for _, candidateRoot := range candidateRoots {
		indices, err := validators.AttestingValidatorIndices(
			state,
			shard,
			candidateRoot,
			currentEpochAttestations,
			prevEpochAttestations)
		if err != nil {
			return nil, fmt.Errorf("could not get attesting validator indices: %v", err)
		}

		var rootBalance uint64
		for _, index := range indices {
			rootBalance += helpers.EffectiveBalance(state, index)
		}

		if rootBalance > winnerBalance ||
			(rootBalance == winnerBalance && b.LowerThan(candidateRoot, winnerRoot)) {
			winnerBalance = rootBalance
			winnerRoot = candidateRoot
		}
	}
	return winnerRoot, nil
}
