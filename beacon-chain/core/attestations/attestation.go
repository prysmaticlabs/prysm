// Package attestations tracks the life-cycle of the latest attestations
// from each validator. It also contains libraries to create attestation
// message, verify attestation correctness and slashing conditions.
package attestations

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// IsDoubleVote checks if both of the attestations have been used to vote for the same slot.
// Spec:
//	def is_double_vote(attestation_data_1: AttestationData,
//                   attestation_data_2: AttestationData) -> bool
//    """
//    Checks if the two ``AttestationData`` have the same target.
//    """
//    target_epoch_1 = slot_to_epoch(attestation_data_1.slot)
//    target_epoch_2 = slot_to_epoch(attestation_data_2.slot)
//    return target_epoch_1 == target_epoch_2
func IsDoubleVote(attestation1 *pb.AttestationData, attestation2 *pb.AttestationData) bool {
	targetEpoch1 := helpers.SlotToEpoch(attestation1.Slot)
	targetEpoch2 := helpers.SlotToEpoch(attestation2.Slot)
	return targetEpoch1 == targetEpoch2
}

// IsSurroundVote checks if the data provided by the attestations fulfill the conditions for
// a surround vote.
// Spec:
//	def is_surround_vote(attestation_data_1: AttestationData,
//                     attestation_data_2: AttestationData) -> bool:
//    """
//    Checks if ``attestation_data_1`` surrounds ``attestation_data_2``.
//    """
//    source_epoch_1 = attestation_data_1.justified_epoch
//    source_epoch_2 = attestation_data_2.justified_epoch
//    target_epoch_1 = slot_to_epoch(attestation_data_1.slot)
//    target_epoch_2 = slot_to_epoch(attestation_data_2.slot)
//
//    return source_epoch_1 < source_epoch_2 and target_epoch_2 < target_epoch_1
func IsSurroundVote(attestation1 *pb.AttestationData, attestation2 *pb.AttestationData) bool {
	sourceEpoch1 := attestation1.JustifiedEpoch
	sourceEpoch2 := attestation2.JustifiedEpoch
	targetEpoch1 := helpers.SlotToEpoch(attestation1.Slot)
	targetEpoch2 := helpers.SlotToEpoch(attestation2.Slot)

	return sourceEpoch1 < sourceEpoch2 && targetEpoch2 < targetEpoch1
}
