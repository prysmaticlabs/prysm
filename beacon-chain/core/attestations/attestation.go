// Package attestations tracks the life-cycle of the latest attestations
// from each validator. It also contains libraries to create attestation
// message, verify attestation correctness and slashing conditions.
package attestations

import (
	"encoding/binary"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// CreateAttestationMsg hashes parentHashes + shardID + slotNumber +
// shardBlockHash + justifiedSlot into a message to use for verifying
// with aggregated public key and signature.
func CreateAttestationMsg(
	blockHash []byte,
	slot uint64,
	shardID uint64,
	justifiedSlot uint64,
	forkVersion uint64,
) [32]byte {
	msg := make([]byte, binary.MaxVarintLen64)
	binary.BigEndian.PutUint64(msg, forkVersion)
	binary.PutUvarint(msg, slot%params.BeaconConfig().EpochLength)
	binary.PutUvarint(msg, shardID)
	msg = append(msg, blockHash...)
	binary.PutUvarint(msg, justifiedSlot)
	return hashutil.Hash(msg)
}

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

// VerifyProposerAttestation verifies the proposer's attestation of the block.
// Proposers broadcast the attestation along with the block to its peers.
func VerifyProposerAttestation(att *pb.AttestationData, pubKey [32]byte, proposerShardID uint64) error {
	// Verify the attestation attached with block response.
	// Get proposer index and shardID.
	attestationMsg := CreateAttestationMsg(
		att.ShardBlockRootHash32,
		att.Slot,
		proposerShardID,
		att.JustifiedSlot,
		params.BeaconConfig().GenesisForkVersion,
	)
	_ = attestationMsg
	_ = pubKey
	// TODO(#258): use attestationMsg to verify against signature
	// and public key. Return error if incorrect.
	return nil
}

// ContainsValidator checks if the validator is included in the attestation.
// TODO(#569): Modify method to accept a single index rather than a bitfield.
func ContainsValidator(attesterBitfield []byte, bitfield []byte) bool {
	for i := 0; i < len(bitfield); i++ {
		if bitfield[i]&attesterBitfield[i] != 0 {
			return true
		}
	}
	return false
}

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
	targetEpoch1 := epoch.SlotToEpoch(attestation1.Slot)
	targetEpoch2 := epoch.SlotToEpoch(attestation2.Slot)
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
	epochLength := params.BeaconConfig().EpochLength
	sourceEpoch1 := attestation1.JustifiedSlot / epochLength
	sourceEpoch2 := attestation2.JustifiedSlot / epochLength
	targetEpoch1 := epoch.SlotToEpoch(attestation1.Slot)
	targetEpoch2 := epoch.SlotToEpoch(attestation2.Slot)

	return sourceEpoch1 < sourceEpoch2 && targetEpoch2 < targetEpoch1
}
