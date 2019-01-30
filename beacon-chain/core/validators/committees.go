// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// CrosslinkCommittee defines the validator committee of slot and shard combinations.
type CrosslinkCommittee struct {
	Committee []uint64
	Shard     uint64
}

// ShuffleValidatorRegistryToCommittees shuffles validator indices and splits them by slot and shard.
// To be deprecated by #1352.
func ShuffleValidatorRegistryToCommittees(
	seed [32]byte,
	validators []*pb.ValidatorRecord,
	crosslinkStartShard uint64,
	slot uint64,
) ([]*pb.ShardCommitteeArray, error) {
	indices := ActiveValidatorIndices(validators, slot)
	// split the shuffled list for slot.
	shuffledValidatorRegistry, err := utils.ShuffleIndices(seed, indices)
	if err != nil {
		return nil, err
	}
	return splitBySlotShard(shuffledValidatorRegistry, crosslinkStartShard), nil
}

// splitBySlotShard splits the validator list into evenly sized committees and assigns each
// committee to a slot and a shard. If the validator set is large, multiple committees are assigned
// to a single slot and shard. See getCommitteesPerSlot for more details.
// To be deprecated by #1352.
func splitBySlotShard(shuffledValidatorRegistry []uint64, crosslinkStartShard uint64) []*pb.ShardCommitteeArray {
	committeesPerSlot := getCommitteesPerSlot(uint64(len(shuffledValidatorRegistry)))
	committeBySlotAndShard := []*pb.ShardCommitteeArray{}

	// split the validator indices by slot.
	validatorsBySlot := utils.SplitIndices(shuffledValidatorRegistry, config.EpochLength)
	for i, validatorsForSlot := range validatorsBySlot {
		shardCommittees := []*pb.ShardCommittee{}
		validatorsByShard := utils.SplitIndices(validatorsForSlot, committeesPerSlot)
		shardStart := crosslinkStartShard + uint64(i)*committeesPerSlot

		for j, validatorsForShard := range validatorsByShard {
			shardID := (shardStart + uint64(j)) % config.ShardCount
			shardCommittees = append(shardCommittees, &pb.ShardCommittee{
				Shard:               shardID,
				Committee:           validatorsForShard,
				TotalValidatorCount: uint64(len(shuffledValidatorRegistry)),
			})
		}

		committeBySlotAndShard = append(committeBySlotAndShard, &pb.ShardCommitteeArray{
			ArrayShardCommittee: shardCommittees,
		})
	}
	return committeBySlotAndShard
}

// getCommitteesPerSlot calculates the parameters for ShuffleValidatorRegistryToCommittees.
// The minimum value for committeesPerSlot is 1.
// Otherwise, the value for committeesPerSlot is the smaller of
// numActiveValidatorRegistry / EpochLength / TargetCommitteeSize or
// ShardCount / CycleLength.//
// To be deprecated by #1352.
func getCommitteesPerSlot(numActiveValidatorRegistry uint64) uint64 {
	epochLength := config.EpochLength
	targetCommitteeSize := config.TargetCommitteeSize

	boundOnValidatorRegistry := numActiveValidatorRegistry / epochLength / targetCommitteeSize
	boundOnShardCount := config.ShardCount / epochLength
	// Ensure that comitteesPerSlot is at least 1.
	if boundOnShardCount == 0 {
		return 1
	} else if boundOnValidatorRegistry > boundOnShardCount {
		return boundOnShardCount
	}
	return boundOnValidatorRegistry
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
//     crosslink_committees = get_crosslink_committees_at_slot(state, attestation_data.slot)
//	   assert attestation_data.shard in [shard for _, shard in crosslink_committees]
//     crosslink_committee = [committee for committee,
//     		shard in crosslink_committees if shard == attestation_data.shard][0]
//     assert len(participation_bitfield) == ceil_div8(len(shard_committee))
//
//     # Find the participating attesters in the committee
//     participants = []
//     for i, validator_index in enumerate(crosslink_committee):
//         aggregation_bit = (aggregation_bitfield[i // 8] >> (7 - (i % 8))) % 2
//         if aggregation_bit == 1:
//             participants.append(validator_index)
//     return participants
func AttestationParticipants(
	state *pb.BeaconState,
	attestationData *pb.AttestationData,
	participationBitfield []byte) ([]uint64, error) {

	// Find the relevant committee.
	crosslinkCommittees, err := CrosslinkCommitteesAtSlot(state, attestationData.Slot)
	if err != nil {
		return nil, err
	}

	var committee []uint64
	for _, crosslinkCommittee := range crosslinkCommittees {
		if crosslinkCommittee.Shard == attestationData.Shard {
			committee = crosslinkCommittee.Committee
			break
		}
	}
	if len(participationBitfield) != mathutil.CeilDiv8(len(committee)) {
		return nil, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(len(committee)),
			len(participationBitfield))
	}

	// Find the participating validators in the committee.
	var participants []uint64
	for i, validatorIndex := range committee {
		bitSet, err := bitutil.CheckBit(participationBitfield, i)
		if err != nil {
			return nil, fmt.Errorf("could not get participant bitfield: %v", err)
		}
		if bitSet {
			participants = append(participants, validatorIndex)
		}
	}
	return participants, nil
}

// CurrCommitteesCountPerSlot returns the number of crosslink committees per slot
// of the current epoch.
// Ex: Returns 8 means there's 8 committees assigned to one slot in current epoch.
//
// Spec pseudocode definition:
//   def get_current_epoch_committee_count_per_slot(state: BeaconState) -> int:
//         current_active_validators =
// 			get_active_validator_indices(validators, state.current_epoch_calculation_slot)
//        return get_committees_per_slot(len(current_active_validators))
func CurrCommitteesCountPerSlot(state *pb.BeaconState) uint64 {
	currActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.CurrentEpochCalculationSlot)
	return committeeCountPerSlot(uint64(len(currActiveValidatorIndices)))
}

// CrosslinkCommitteesAtSlot returns the list of crosslink committees, it
// contains the shard associated with the committee and the validator indices
// in that committee.
//   def get_crosslink_committees_at_slot(state: BeaconState,
//                                     slot: int) -> List[Tuple[List[int], int]]:
//    """
//    Returns the list of ``(committee, shard)`` tuples for the ``slot``.
//    """
//    state_epoch_slot = state.slot - (state.slot % EPOCH_LENGTH)
//    assert state_epoch_slot <= slot + EPOCH_LENGTH
//    assert slot < state_epoch_slot + EPOCH_LENGTH
//    offset = slot % EPOCH_LENGTH
//
//    if slot < state_epoch_slot:
//        committees_per_slot = get_previous_epoch_committee_count_per_slot(state)
//        shuffling = get_shuffling(
//            state.previous_epoch_randao_mix,
//            state.validator_registry,
//            state.previous_epoch_calculation_slot,
//        )
//        slot_start_shard = (state.previous_epoch_start_shard + committees_per_slot * offset) % SHARD_COUNT
//    else:
//        committees_per_slot = get_current_epoch_committee_count_per_slot(state)
//        shuffling = get_shuffling(
//            state.current_epoch_randao_mix,
//            state.validator_registry,
//            state.current_epoch_calculation_slot,
//        )
//        slot_start_shard = (state.current_epoch_start_shard + committees_per_slot * offset) % SHARD_COUNT
//
//    return [
//        (
//            shuffling[committees_per_slot * offset + i],
//            (slot_start_shard + i) % SHARD_COUNT,
//        )
//        for i in range(committees_per_slot)
//    ]
func CrosslinkCommitteesAtSlot(state *pb.BeaconState, slot uint64) ([]*CrosslinkCommittee, error) {
	var earliestSlot uint64
	var countPerSlot uint64
	var startShard uint64
	var shuffledIndices [][]uint64
	var err error

	epochLength := config.EpochLength
	startEpochSlot := state.Slot - (state.Slot % epochLength)

	// If the start epoch slot is less than epochLength, then the earliestSlot would
	// result in a negative number. Therefore we should default to
	// earliestSlot = 0 in this case.
	if startEpochSlot > epochLength {
		earliestSlot = startEpochSlot - epochLength
	}

	if slot < earliestSlot || slot >= startEpochSlot+epochLength {
		return nil, fmt.Errorf(
			"input committee slot %d out of bounds: %d <= slot < %d",
			slot,
			earliestSlot,
			startEpochSlot+epochLength,
		)
	}

	offSet := slot % config.EpochLength
	if slot < startEpochSlot {
		countPerSlot = prevCommitteesCountPerSlot(state)
		shuffledIndices, err = Shuffling(
			bytesutil.ToBytes32(state.PreviousEpochRandaoMixHash32),
			state.ValidatorRegistry,
			state.PreviousEpochCalculationSlot)
		if err != nil {
			return nil, fmt.Errorf("could not shuffle prev epoch validators: %v", err)
		}
		startShard = (state.PreviousEpochStartShard + countPerSlot*offSet) %
			config.ShardCount
	} else {
		countPerSlot = CurrCommitteesCountPerSlot(state)
		shuffledIndices, err = Shuffling(
			bytesutil.ToBytes32(state.CurrentEpochRandaoMixHash32),
			state.ValidatorRegistry,
			state.CurrentEpochCalculationSlot)
		if err != nil {
			return nil, fmt.Errorf("could not shuffle current epoch validators: %v", err)
		}
		startShard = (state.CurrentEpochCalculationSlot + countPerSlot*offSet) %
			config.ShardCount
	}

	var crosslinkCommittees []*CrosslinkCommittee
	for i := uint64(0); i < countPerSlot; i++ {
		crosslinkCommittees = append(crosslinkCommittees, &CrosslinkCommittee{
			Committee: shuffledIndices[countPerSlot*offSet+i],
			Shard:     (startShard + i) % config.ShardCount,
		})
	}

	return crosslinkCommittees, nil
}

// Shuffling shuffles input validator indices and splits them by slot and shard.
//
// Spec pseudocode definition:
//   def get_shuffling(seed: Bytes32,
//                  validators: List[Validator],
//                  slot: int) -> List[List[int]]
//    """
//    Shuffles ``validators`` into crosslink committees seeded by ``seed`` and ``slot``.
//    Returns a list of ``EPOCH_LENGTH * committees_per_slot`` committees where each
//    committee is itself a list of validator indices.
//    """
//
//    # Normalizes slot to start of epoch boundary
//    slot -= slot % EPOCH_LENGTH
//
//    active_validator_indices = get_active_validator_indices(validators, slot)
//
//    committees_per_slot = get_committee_count_per_slot(len(active_validator_indices))
//
//    # Shuffle
//    seed = xor(seed, int_to_bytes32(slot))
//    shuffled_active_validator_indices = shuffle(active_validator_indices, seed)
//
//    # Split the shuffled list into epoch_length * committees_per_slot pieces
//    return split(shuffled_active_validator_indices, committees_per_slot * EPOCH_LENGTH)
func Shuffling(
	seed [32]byte,
	validators []*pb.ValidatorRecord,
	slot uint64) ([][]uint64, error) {

	// Normalize slot to start of epoch boundary.
	slot -= slot % config.EpochLength

	// Figure out how many committees can be in a single slot.
	activeIndices := ActiveValidatorIndices(validators, slot)
	activeCount := uint64(len(activeIndices))
	committeesPerSlot := committeeCountPerSlot(activeCount)

	// Convert slot to bytes and xor it with seed.
	slotInBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(slotInBytes, slot)
	seed = bytesutil.ToBytes32(bytesutil.Xor(seed[:], slotInBytes))

	shuffledIndices, err := utils.ShuffleIndices(seed, activeIndices)
	if err != nil {
		return nil, err
	}

	// Split the shuffled list into epoch_length * committees_per_slot pieces.
	return utils.SplitIndices(shuffledIndices, committeesPerSlot*config.EpochLength), nil
}

// committeeCountPerSlot returns the number of crosslink committees of one slot.
//
// Spec pseudocode definition:
//   def get_committee_count_per_slot(active_validator_count: int) -> int:
//    return max(
//        1,
//        min(
//            SHARD_COUNT // EPOCH_LENGTH,
//            active_validator_count // EPOCH_LENGTH // TARGET_COMMITTEE_SIZE,
//        )
//    )
func committeeCountPerSlot(activeValidatorCount uint64) uint64 {
	var minCommitteePerSlot = uint64(1)
	var maxCommitteePerSlot = config.ShardCount / config.EpochLength
	var currCommitteePerSlot = activeValidatorCount / config.EpochLength / config.TargetCommitteeSize
	if currCommitteePerSlot > maxCommitteePerSlot {
		return maxCommitteePerSlot
	}
	if currCommitteePerSlot < 1 {
		return minCommitteePerSlot
	}
	return currCommitteePerSlot
}

// prevCommitteesCountPerSlot returns the number of committees per slot
// of the previous epoch.
// Ex: Returns 16 means there's 16 committees assigned to one slot in previous epoch.
//
// Spec pseudocode definition:
//   def get_previous_epoch_committee_count_per_slot(state: BeaconState) -> int:
//         previous_active_validators =
// 			get_active_validator_indices(validators, state.previous_epoch_calculation_slot)
//        return get_committees_per_slot(len(previous_active_validators))
func prevCommitteesCountPerSlot(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.PreviousEpochCalculationSlot)
	return committeeCountPerSlot(uint64(len(prevActiveValidatorIndices)))
}
