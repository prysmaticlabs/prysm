// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// CrosslinkCommittee defines the validator committee of slot and shard combinations.
type CrosslinkCommittee struct {
	Committee []uint64
	Shard     uint64
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

// CrosslinkCommitteesAtSlot returns the list of crosslink committees, it
// contains the shard associated with the committee and the validator indices
// in that committee.
//   def get_crosslink_committees_at_slot(state: BeaconState,
//                                     slot: SlotNumber) -> List[Tuple[List[ValidatorIndex], ShardNumber]]:
//    """
//    Returns the list of ``(committee, shard)`` tuples for the ``slot``.
//    """
//    epoch = slot_to_epoch(slot)
//    current_epoch = get_current_epoch(state)
//    previous_epoch = current_epoch - 1 if current_epoch > GENESIS_EPOCH else current_epoch
//    next_epoch = current_epoch + 1
//
//    assert previous_epoch <= epoch < next_epoch
//
//    if epoch < current_epoch:
//        committees_per_epoch = get_previous_epoch_committee_count(state)
//        seed = state.previous_epoch_seed
//        shuffling_epoch = state.previous_calculation_epoch
//        shuffling_start_shard = state.previous_epoch_start_shard
//    else:
//        committees_per_epoch = get_current_epoch_committee_count(state)
//        seed = state.current_epoch_seed
//        shuffling_epoch = state.current_calculation_epoch
//        shuffling_start_shard = state.current_epoch_start_shard
//
//    shuffling = get_shuffling(
//        seed,
//        state.validator_registry,
//        shuffling_epoch,
//    )
//    offset = slot % EPOCH_LENGTH
//    committees_per_slot = committees_per_epoch // EPOCH_LENGTH
//    slot_start_shard = (shuffling_start_shard + committees_per_slot * offset) % SHARD_COUNT
//
//    return [
//        (
//            shuffling[committees_per_slot * offset + i],
//            (slot_start_shard + i) % SHARD_COUNT,
//        )
//        for i in range(committees_per_slot)
//    ]
func CrosslinkCommitteesAtSlot(state *pb.BeaconState, slot uint64) ([]*CrosslinkCommittee, error) {
	var countPerSlot uint64
	var startShard uint64
	var shuffledIndices [][]uint64
	var err error

	wantedEpoch := helpers.SlotToEpoch(slot)
	currentEpoch := helpers.CurrentEpoch(state)
	prevEpoch := helpers.PrevEpoch(state)
	nextEpoch := helpers.NextEpoch(state)

	if wantedEpoch < prevEpoch || wantedEpoch >= nextEpoch {
		return nil, fmt.Errorf(
			"input committee epoch %d out of bounds: %d <= epoch < %d",
			wantedEpoch,
			prevEpoch,
			currentEpoch,
		)
	}

	offSet := slot % config.EpochLength
	if wantedEpoch < currentEpoch {
		countPerSlot = helpers.PrevEpochCommitteeCount(state)
		shuffledIndices, err = Shuffling(
			bytesutil.ToBytes32(state.PreviousEpochSeedHash32),
			state.ValidatorRegistry,
			state.PreviousEpochCalculationSlot)
		if err != nil {
			return nil, fmt.Errorf("could not shuffle prev epoch validators: %v", err)
		}
		startShard = (state.PreviousEpochStartShard + countPerSlot*offSet) %
			config.ShardCount
	} else {
		countPerSlot = helpers.CurrentEpochCommitteeCount(state)
		shuffledIndices, err = Shuffling(
			bytesutil.ToBytes32(state.CurrentEpochSeedHash32),
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
	activeIndices := helpers.ActiveValidatorIndices(validators, slot)
	activeCount := uint64(len(activeIndices))
	committeesPerSlot := helpers.EpochCommitteeCount(activeCount)

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
