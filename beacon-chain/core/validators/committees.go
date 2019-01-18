// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"encoding/binary"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/bytes"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ShardCommittee defines the validator committee of each slot and shard combinations.
type ShardCommittee struct {
	Committee []uint32
	Shard     uint64
}

var config = params.BeaconConfig()

// ShuffleValidatorRegistryToCommittees shuffles validator indices and splits them by slot and shard.
func ShuffleValidatorRegistryToCommittees(
	seed [32]byte,
	validators []*pb.ValidatorRecord,
	slot uint64,
) ([][]uint32, error) {
	// Normalizes slot to start of epoch boundary.
	slot -= slot % config.EpochLength

	indices := ActiveValidatorIndices(validators, slot)
	countPerSlot := committeesCountPerSlot(uint64(len(indices)))

	slotInBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(slotInBytes, slot)
	seed = bytes.ToBytes32(bytes.Xor(seed[:], slotInBytes))

	shuffledIndices, err := utils.ShuffleIndices(seed, indices)
	if err != nil {
		return nil, err
	}

	// Split the shuffled list into epoch_length * committees_per_slot pieces.
	return utils.SplitIndices(shuffledIndices, countPerSlot*config.EpochLength), nil
}

// committeesCountPerSlot calculates the number of committees per slot.
// The minimum value for committees per slot is 1.
// Otherwise, the value for committees per slot is the smaller of
// activeValidatorCount / EpochLength / TargetCommitteeSize or
// ShardCount / CycleLength.
//
// Spec pseudocode definition:
//   def get_committees_per_slot(active_validator_count: int) -> int:
//         return max(
//        1,
//        min(
//            SHARD_COUNT // EPOCH_LENGTH,
//            len(active_validator_indices) // EPOCH_LENGTH // TARGET_COMMITTEE_SIZE,
//        )
//    )
func committeesCountPerSlot(activeValidatorCount uint64) uint64 {
	epochLength := config.EpochLength
	targetCommitteeSize := config.TargetCommitteeSize

	boundOnValidatorRegistry := activeValidatorCount / epochLength / targetCommitteeSize
	boundOnShardCount := config.ShardCount / epochLength
	// Ensure that committeesPerSlot is at least 1.
	if boundOnValidatorRegistry == 0 {
		return 1
	} else if boundOnValidatorRegistry > boundOnShardCount {
		return boundOnShardCount
	}
	return boundOnValidatorRegistry
}

// prevCommitteesCountPerSlot returns the number of committees per slot
// for the previous epoch.
//
// Spec pseudocode definition:
//   def get_previous_epoch_committee_count_per_slot(state: BeaconState) -> int:
//         previous_active_validators =
// 			get_active_validator_indices(validators, state.previous_epoch_calculation_slot)
//        return get_committees_per_slot(len(previous_active_validators))
func prevCommitteesCountPerSlot(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.PreviousEpochCalculationSlot)
	return committeesCountPerSlot(uint64(len(prevActiveValidatorIndices)))
}

// CurrCommitteesCountPerSlot returns the number of committees per slot
// for the curent epoch.
//
// Spec pseudocode definition:
//   def get_current_epoch_committee_count_per_slot(state: BeaconState) -> int:
//         current_active_validators =
// 			get_active_validator_indices(validators, state.current_epoch_calculation_slot)
//        return get_committees_per_slot(len(current_active_validators))
func CurrCommitteesCountPerSlot(state *pb.BeaconState) uint64 {
	currActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.CurrentEpochCalculationSlot)
	return committeesCountPerSlot(uint64(len(currActiveValidatorIndices)))
}

// ShardCommitteesAtSlot returns (i) the list of committees and
// (ii) the shard associated with the first committee for the ``slot``
// It's bounded within the range of 2 * epoch length.
func ShardCommitteesAtSlot(state *pb.BeaconState, slot uint64) ([]*ShardCommittee, error) {
	var earliestSlot uint64
	var slotStartShard uint64
	var countPerSlot uint64
	var shuffledIndices [][]uint32
	var err error
	epochLength := config.EpochLength

	// If the state slot is less than epochLength, then the earliestSlot would
	// result in a negative number. Therefore we should default to
	// earliestSlot = 0 in this case.
	if state.Slot > epochLength {
		earliestSlot = state.Slot - (state.Slot % epochLength) - epochLength
	}

	if slot < earliestSlot || slot >= earliestSlot+(epochLength*2) {
		return nil, fmt.Errorf("slot %d out of bounds: %d <= slot < %d",
			slot,
			earliestSlot,
			earliestSlot+(epochLength*2),
		)
	}
	offSet := slot % config.EpochLength
	if slot < earliestSlot+epochLength {
		countPerSlot = prevCommitteesCountPerSlot(state)
		shuffledIndices, err = ShuffleValidatorRegistryToCommittees(
			bytes.ToBytes32(state.PreviousEpochRandaoMixHash32),
			state.ValidatorRegistry,
			state.PreviousEpochCalculationSlot)
		if err != nil {
			return nil, fmt.Errorf("could not shuffle prev epoch validators: %v", err)
		}
		slotStartShard = (state.PreviousEpochStartShard + countPerSlot*offSet) %
			config.ShardCount
	} else {
		countPerSlot = CurrCommitteesCountPerSlot(state)
		shuffledIndices, err = ShuffleValidatorRegistryToCommittees(
			bytes.ToBytes32(state.CurrentEpochRandaoMixHash32),
			state.ValidatorRegistry,
			state.CurrentEpochCalculationSlot)
		if err != nil {
			return nil, fmt.Errorf("could not shuffle current epoch validators: %v", err)
		}
		slotStartShard = (state.CurrentEpochCalculationSlot + countPerSlot*offSet) %
			config.ShardCount
	}

	var shardCommittees []*ShardCommittee
	for i := uint64(0); i < countPerSlot; i++ {
		shardCommittees = append(shardCommittees, &ShardCommittee{
			Committee: shuffledIndices[countPerSlot*offSet+i],
			Shard:     (slotStartShard + i) % config.ShardCount,
		})
	}

	return shardCommittees, nil
}

// AttestationParticipants returns the attesting participants indices.
//
// Spec pseudocode definition:
//   def get_attestation_participants(state: BeaconState,
//     attestation_data: AttestationData,
//     participation_bitfield: bytes) -> List[int]:
//     """
//     Returns the participant indices at for the ``attestation_data`` and ``participation_bitfield``.
//     """
//
//     # Find the committee in the list with the desired shard
//     shard_committees = get_shard_committees_at_slot(state, attestation_data.slot)
//	   assert attestation.shard in [shard for _, shard in shard_committees]
//     shard_committee = [committee for committee,
// 			shard in shard_committees if shard == attestation_data.shard][0]
//     assert len(participation_bitfield) == ceil_div8(len(shard_committee))
//
//     # Find the participating attesters in the committee
//     participants = []
//     for i, validator_index in enumerate(shard_committee):
//         participation_bit = (participation_bitfield[i//8] >> (7 - (i % 8))) % 2
//         if participation_bit == 1:
//             participants.append(validator_index)
//     return participants
func AttestationParticipants(
	state *pb.BeaconState,
	attestationData *pb.AttestationData,
	participationBitfield []byte) ([]uint32, error) {

	// Find the relevant committee.
	shardCommittees, err := ShardCommitteesAtSlot(state, attestationData.Slot)
	if err != nil {
		return nil, err
	}

	var committee []uint32
	for _, shardCommittee := range shardCommittees {
		if shardCommittee.Shard == attestationData.Shard {
			committee = shardCommittee.Committee
			break
		}
	}
	if len(committee) == 0 {
		return nil, fmt.Errorf("could not find committee with shard %d",
			attestationData.Shard)
	}
	if len(participationBitfield) != mathutil.CeilDiv8(len(committee)) {
		return nil, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(len(committee)),
			len(participationBitfield))
	}

	// Find the participating attesters in the committee.
	var participantIndices []uint32
	for i, validatorIndex := range committee {
		bitSet, err := bitutil.CheckBit(participationBitfield, i)
		if err != nil {
			return nil, fmt.Errorf("could not get participant bitfield: %v", err)
		}
		if bitSet {
			participantIndices = append(participantIndices, validatorIndex)
		}
	}
	return participantIndices, nil
}
