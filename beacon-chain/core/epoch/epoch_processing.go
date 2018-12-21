package epoch

import (
	"bytes"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Attestations returns the pending attestations of slots in the epoch
// (state.slot-EPOCH_LENGTH...state.slot-1), not attestations that got
// included in the chain during the epoch.
//
// Spec pseudocode definition:
//   [a for a in state.latest_attestations
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
//   [a for a in this_epoch_attestations if a.data.epoch_boundary_root ==
//   get_block_root(state, state.slot-EPOCH_LENGTH) and a.justified_slot ==
//   state.justified_slot]
func BoundaryAttestations(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestationRecord,
) ([]*pb.PendingAttestationRecord, error) {
	epochLength := params.BeaconConfig().EpochLength
	var boundaryAttestations []*pb.PendingAttestationRecord

	for _, attestation := range thisEpochAttestations {

		boundaryBlockRoot, err := b.BlockRoot(state, state.Slot-epochLength)
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
//   [a for a in state.latest_attestations
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
//   [a for a in this_epoch_attestations + previous_epoch_attestations
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
//   [a for a in previous_epoch_attestations
//   if a.beacon_block_root == get_block_root(state, a.slot)]
func PrevHeadAttestations(
	state *pb.BeaconState,
	prevEpochAttestations []*pb.PendingAttestationRecord,
) ([]*pb.PendingAttestationRecord, error) {
	var headAttestations []*pb.PendingAttestationRecord

	for _, attestation := range prevEpochAttestations {
		canonicalBlockRoot, err := b.BlockRoot(state, attestation.GetData().GetSlot())
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
