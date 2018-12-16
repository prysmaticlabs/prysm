package epoch

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slices"
)

// Attestations returns the pending attestations of slots in the epoch
// (state.slot-EPOCH_LENGTH...state.slot-1), not attestations that got
// included in the chain during the epoch.
//
// Spec pseudocode definition:
//   [a for a in state.latest_attestations if state.slot - EPOCH_LENGTH <=
//   a.data.slot < state.slot]
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

		if earliestSlot <= attestation.GetData().Slot && attestation.GetData().Slot < state.Slot {
			thisEpochAttestations = append(thisEpochAttestations, attestation)
		}
	}
	return thisEpochAttestations
}

// BoundaryAttestations returns the pending attestations from
// the epoch boundary block.
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

		boundaryBlockRoot, err := types.BlockRoot(state, state.Slot-epochLength)
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

// BoundaryAttesterIndices returns the attester indices from epoch
// boundary block.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attester_indices be the union of the validator
//   index sets given by [get_attestation_participants(state, a.data, a.participation_bitfield)
//   for a in this_epoch_boundary_attestations]
func BoundaryAttesterIndices(
	state *pb.BeaconState,
	boundaryAttestations []*pb.PendingAttestationRecord,
) ([]uint32, error) {

	var attesterIndicesIntersection []uint32
	for _, boundaryAttestation := range boundaryAttestations {
		attesterIndices, err := validators.AttestationParticipants(
			state,
			boundaryAttestation.Data,
			boundaryAttestation.ParticipationBitfield)
		if err != nil {
			return nil, err
		}
		attesterIndicesIntersection = slices.Intersection(attesterIndicesIntersection, attesterIndices)
	}

	return attesterIndicesIntersection, nil
}

// BoundaryAttesters returns the attesters validator object from epoch
// boundary block.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attesters = [state.validator_registry[i]
//   for indices in this_epoch_boundary_attester_indices for i in indices].
func BoundaryAttesters(state *pb.BeaconState, attesterIndices []uint32) []*pb.ValidatorRecord {

	var boundaryAttesters []*pb.ValidatorRecord
	for _, attesterIndex := range attesterIndices {
		boundaryAttesters = append(boundaryAttesters, state.ValidatorRegistry[attesterIndex])
	}

	return boundaryAttesters
}

// BoundaryAttestingBalance returns the epoch boundary attesting
// balance.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attesting_balance =
//   sum([get_effective_balance(v) for v in this_epoch_boundary_attesters])
func BoundaryAttestingBalance(boundaryAttesters []*pb.ValidatorRecord) uint64 {

	var boundaryAttestingBalance uint64
	for _, boundaryAttester := range boundaryAttesters {
		boundaryAttestingBalance += validators.EffectiveBalance(boundaryAttester)
	}

	return boundaryAttestingBalance
}
