// Package attestations tracks the life-cycle of the latest attestations
// from each validator. It also contains libraries to create attestation
// message, verify attestation correctness and slashing conditions.
package attestations

import (
	"encoding/binary"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Key generates the blake2b hash of the following attestation fields:
// slotNumber + shardID + blockHash + obliqueParentHash
// This is used for attestation table look up in localDB.
func Key(att *pb.AttestationData) [32]byte {
	key := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(key, att.Slot)
	binary.PutUvarint(key, att.Shard)
	key = append(key, att.ShardBlockRootHash32...)
	return hashutil.Hash(key)
}

// IsDoubleVote checks if both of the attestations have been used to vote for the same slot.
// Spec:
//	def is_double_vote(attestation_data_1: AttestationData,
//                   attestation_data_2: AttestationData) -> bool
//    """
//    Assumes ``attestation_data_1`` is distinct from ``attestation_data_2``.
//    Returns True if the provided ``AttestationData`` are slashable
//    due to a 'double vote'.
//    """
//    target_epoch_1 = attestation_data_1.slot // EPOCH_LENGTH
//    target_epoch_2 = attestation_data_2.slot // EPOCH_LENGTH
//    return target_epoch_1 == target_epoch_2
func IsDoubleVote(attestation1 *pb.AttestationData, attestation2 *pb.AttestationData) bool {
	epochLength := params.BeaconConfig().EpochLength
	return attestation1.Slot/epochLength == attestation2.Slot/epochLength
}

// IsSurroundVote checks if the data provided by the attestations fulfill the conditions for
// a surround vote.
// Spec:
//	def is_surround_vote(attestation_data_1: AttestationData,
//                     attestation_data_2: AttestationData) -> bool:
//    """
//    Assumes ``attestation_data_1`` is distinct from ``attestation_data_2``.
//    Returns True if the provided ``AttestationData`` are slashable
//    due to a 'surround vote'.
//    Note: parameter order matters as this function only checks
//    that ``attestation_data_1`` surrounds ``attestation_data_2``.
//    """
//    source_epoch_1 = attestation_data_1.justified_slot // EPOCH_LENGTH
//    source_epoch_2 = attestation_data_2.justified_slot // EPOCH_LENGTH
//    target_epoch_1 = attestation_data_1.slot // EPOCH_LENGTH
//    target_epoch_2 = attestation_data_2.slot // EPOCH_LENGTH
//    return (
//        (source_epoch_1 < source_epoch_2) and
//        (source_epoch_2 + 1 == target_epoch_2) and
//        (target_epoch_2 < target_epoch_1)
//    )
func IsSurroundVote(attestation1 *pb.AttestationData, attestation2 *pb.AttestationData) bool {
	epochLength := params.BeaconConfig().EpochLength
	sourceEpoch1 := attestation1.JustifiedSlot / epochLength
	sourceEpoch2 := attestation2.JustifiedSlot / epochLength
	targetEpoch1 := attestation1.Slot / epochLength
	targetEpoch2 := attestation2.Slot / epochLength

	return sourceEpoch1 < sourceEpoch2 &&
		sourceEpoch2+1 == targetEpoch2 &&
		targetEpoch2 < targetEpoch1
}
