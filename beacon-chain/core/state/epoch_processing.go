package state

import (
	"github.com/prysmaticlabs/prysm/bazel-prysm/external/go_sdk/src/bytes"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
)

// EpochAttestations returns the pending attestations of slots in the epoch
// (state.slot-EPOCH_LENGTH...state.slot-1), not attestations that got
// included in the chain during the epoch.
//
// Spec pseudocode definition:
//   this_epoch_attestations = [a for a in state.latest_attestations
//   if state.slot - EPOCH_LENGTH <= a.data.slot < state.slot]
func EpochAttestations(state *pb.BeaconState) []*pb.PendingAttestationRecord {
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

		if earliestSlot <= attestation.Data.Slot && attestation.Data.Slot < state.Slot {
			thisEpochAttestations = append(thisEpochAttestations, attestation)
		}
	}
	return thisEpochAttestations
}

// EpochBoundaryAttestations returns the pending attestations from
// the epoch boundary block.
//
// Spec pseudocode definition:
//   [a for a in this_epoch_attestations if a.data.epoch_boundary_root ==
//   get_block_root(state, state.slot-EPOCH_LENGTH) and a.justified_slot ==
//   if state.slot - EPOCH_LENGTH <= a.data.slot < state.slot]
func EpochBoundaryAttestations(state *pb.BeaconState, thisEpochAttestations []*pb.PendingAttestationRecord) ([]*pb.PendingAttestationRecord, error) {
	epochLength := params.BeaconConfig().EpochLength
	var boundaryAttestations []*pb.PendingAttestationRecord

	for _, attestation := range thisEpochAttestations {

		boundaryBlockRoot, err := types.BlockRoot(state, state.Slot - epochLength)
		if err != nil {
			return nil, err
		}

		sameRoot := bytes.Equal(attestation.Data.JustifiedBlockHash32, boundaryBlockRoot)
		sameSlotNum := attestation.Data.JustifiedSlot == state.JustifiedSlot
		if sameRoot && sameSlotNum {
			boundaryAttestations = append(boundaryAttestations, attestation)
		}
	}
	return boundaryAttestations, nil
}
