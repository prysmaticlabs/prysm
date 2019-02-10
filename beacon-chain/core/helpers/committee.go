package helpers

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// CrosslinkCommittee defines the validator committee of slot and shard combinations.
type CrosslinkCommittee struct {
	Committee []uint64
	Shard     uint64
}

// EpochCommitteeCount returns the number of crosslink committees of an epoch.
//
// Spec pseudocode definition:
//   def get_epoch_committee_count(active_validator_count: int) -> int:
//    """
//    Return the number of committees in one epoch.
//    """
//    return max(
//        1,
//        min(
//            SHARD_COUNT // EPOCH_LENGTH,
//            active_validator_count // EPOCH_LENGTH // TARGET_COMMITTEE_SIZE,
//        )
//    ) * EPOCH_LENGTH
func EpochCommitteeCount(activeValidatorCount uint64) uint64 {
	var minCommitteePerSlot = uint64(1)
	var maxCommitteePerSlot = params.BeaconConfig().ShardCount / params.BeaconConfig().EpochLength
	var currCommitteePerSlot = activeValidatorCount / params.BeaconConfig().EpochLength / params.BeaconConfig().TargetCommitteeSize
	if currCommitteePerSlot > maxCommitteePerSlot {
		return maxCommitteePerSlot * params.BeaconConfig().EpochLength
	}
	if currCommitteePerSlot < 1 {
		return minCommitteePerSlot * params.BeaconConfig().EpochLength
	}
	return currCommitteePerSlot * params.BeaconConfig().EpochLength
}

// CurrentEpochCommitteeCount returns the number of crosslink committees per epoch
// of the current epoch.
// Ex: Returns 100 means there's 8 committees assigned to current epoch.
//
// Spec pseudocode definition:
//   def get_current_epoch_committee_count(state: BeaconState) -> int:
//    """
//    Return the number of committees in the current epoch of the given ``state``.
//    """
//    current_active_validators = get_active_validator_indices(
//        state.validator_registry,
//        state.current_calculation_epoch,
//    )
//    return get_epoch_committee_count(len(current_active_validators)
func CurrentEpochCommitteeCount(state *pb.BeaconState) uint64 {
	currActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.CurrentCalculationEpoch)
	return EpochCommitteeCount(uint64(len(currActiveValidatorIndices)))
}

// PrevEpochCommitteeCount returns the number of committees per slot
// of the previous epoch.
//
// Spec pseudocode definition:
//   def get_previous_epoch_committee_count(state: BeaconState) -> int:
//    """
//    Return the number of committees in the previous epoch of the given ``state``.
//    """
//    previous_active_validators = get_active_validator_indices(
//        state.validator_registry,
//        state.previous_calculation_epoch,
//    )
//    return get_epoch_committee_count(len(previous_active_validators))
func PrevEpochCommitteeCount(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.PreviousCalculationEpoch)
	return EpochCommitteeCount(uint64(len(prevActiveValidatorIndices)))
}

// NextEpochCommitteeCount returns the number of committees per slot
// of the next epoch.
//
// Spec pseudocode definition:
//   def get_next_epoch_committee_count(state: BeaconState) -> int:
//    """
//    Return the number of committees in the next epoch of the given ``state``.
//    """
//    next_active_validators = get_active_validator_indices(
//        state.validator_registry,
//        get_current_epoch(state) + 1,
//    )
//    return get_epoch_committee_count(len(next_active_validators))
func NextEpochCommitteeCount(state *pb.BeaconState) uint64 {
	prevActiveValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, CurrentEpoch(state)+1)
	return EpochCommitteeCount(uint64(len(prevActiveValidatorIndices)))
}

// CrosslinkCommitteesAtSlot returns the list of crosslink committees, it
// contains the shard associated with the committee and the validator indices
// in that committee.
//   def get_crosslink_committees_at_slot(state: BeaconState,
//                                     slot: SlotNumber,
//                                     registry_change=False: bool) -> List[Tuple[List[ValidatorIndex], ShardNumber]]:
//    """
//    Return the list of ``(committee, shard)`` tuples for the ``slot``.
//
//    Note: There are two possible shufflings for crosslink committees for a
//    ``slot`` in the next epoch -- with and without a `registry_change`
//    """
//    epoch = slot_to_epoch(slot)
//    current_epoch = get_current_epoch(state)
//    previous_epoch = current_epoch - 1 if current_epoch > GENESIS_EPOCH else current_epoch
//    next_epoch = current_epoch + 1
//
//    assert previous_epoch <= epoch <= next_epoch
//
//    if epoch == previous_epoch:
//        committees_per_epoch = get_previous_epoch_committee_count(state)
//        seed = state.previous_epoch_seed
//        shuffling_epoch = state.previous_calculation_epoch
//        shuffling_start_shard = state.previous_epoch_start_shard
//    elif epoch == current_epoch:
//        committees_per_epoch = get_current_epoch_committee_count(state)
//        seed = state.current_epoch_seed
//        shuffling_epoch = state.current_calculation_epoch
//        shuffling_start_shard = state.current_epoch_start_shard
//    elif epoch == next_epoch:
//        current_committees_per_epoch = get_current_epoch_committee_count(state)
//        committees_per_epoch = get_next_epoch_committee_count(state)
//        shuffling_epoch = next_epoch
//
//        epochs_since_last_registry_update = current_epoch - state.validator_registry_update_epoch
//        if registry_change:
//            seed = generate_seed(state, next_epoch)
//            shuffling_start_shard = (state.current_epoch_start_shard + current_committees_per_epoch) % SHARD_COUNT
//        elif epochs_since_last_registry_update > 1 and is_power_of_two(epochs_since_last_registry_update):
//            seed = generate_seed(state, next_epoch)
//            shuffling_start_shard = state.current_epoch_start_shard
//        else:
//            seed = state.current_epoch_seed
//            shuffling_start_shard = state.current_epoch_start_shard
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
func CrosslinkCommitteesAtSlot(
	state *pb.BeaconState,
	slot uint64,
	registryChange bool) ([]*CrosslinkCommittee, error) {
	var committeesPerEpoch uint64
	var shufflingEpoch uint64
	var shufflingStartShard uint64
	var seed [32]byte
	var err error

	wantedEpoch := SlotToEpoch(slot)
	currentEpoch := CurrentEpoch(state)
	prevEpoch := PrevEpoch(state)
	nextEpoch := NextEpoch(state)

	if wantedEpoch < prevEpoch || wantedEpoch > nextEpoch {
		return nil, fmt.Errorf(
			"input committee epoch %d out of bounds: %d <= epoch <= %d",
			wantedEpoch,
			prevEpoch,
			currentEpoch,
		)
	}

	if wantedEpoch == prevEpoch {
		committeesPerEpoch = PrevEpochCommitteeCount(state)
		seed = bytesutil.ToBytes32(state.PreviousEpochSeedHash32)
		shufflingEpoch = state.PreviousCalculationEpoch
		shufflingStartShard = state.PreviousEpochStartShard
	} else if wantedEpoch == currentEpoch {
		committeesPerEpoch = PrevEpochCommitteeCount(state)
		seed = bytesutil.ToBytes32(state.CurrentEpochSeedHash32)
		shufflingEpoch = state.CurrentCalculationEpoch
		shufflingStartShard = state.CurrentEpochStartShard
	} else if wantedEpoch == nextEpoch {
		currentCommitteesPerEpoch := CurrentEpochCommitteeCount(state)
		committeesPerEpoch = NextEpochCommitteeCount(state)
		shufflingEpoch = nextEpoch

		epochsSinceLastRegistryUpdate := currentEpoch - state.ValidatorRegistryUpdateEpoch
		if registryChange {
			seed, err = GenerateSeed(state, nextEpoch)
			if err != nil {
				return nil, fmt.Errorf("could not generate seed: %v", err)
			}
			shufflingStartShard = (state.CurrentEpochStartShard + currentCommitteesPerEpoch) %
				params.BeaconConfig().ShardCount
		} else if epochsSinceLastRegistryUpdate > 1 &&
			mathutil.IsPowerOf2(epochsSinceLastRegistryUpdate) {
			seed, err = GenerateSeed(state, nextEpoch)
			if err != nil {
				return nil, fmt.Errorf("could not generate seed: %v", err)
			}
			shufflingStartShard = state.CurrentEpochStartShard
		} else {
			seed = bytesutil.ToBytes32(state.CurrentEpochSeedHash32)
			shufflingStartShard = state.CurrentEpochStartShard
		}
	}

	shuffledIndices, err := Shuffling(
		seed,
		state.ValidatorRegistry,
		shufflingEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not shuffle epoch validators: %v", err)
	}

	offSet := slot % params.BeaconConfig().EpochLength
	committeesPerSlot := committeesPerEpoch / params.BeaconConfig().EpochLength
	slotStardShard := (shufflingStartShard + committeesPerSlot*offSet) %
		params.BeaconConfig().ShardCount

	var crosslinkCommittees []*CrosslinkCommittee
	for i := uint64(0); i < committeesPerSlot; i++ {
		crosslinkCommittees = append(crosslinkCommittees, &CrosslinkCommittee{
			Committee: shuffledIndices[committeesPerSlot*offSet+i],
			Shard:     (slotStardShard + i) % params.BeaconConfig().ShardCount,
		})
	}

	return crosslinkCommittees, nil
}

// Shuffling shuffles input validator indices and splits them by slot and shard.
//
// Spec pseudocode definition:
//   def get_shuffling(seed: Bytes32,
//                  validators: List[Validator],
//                  epoch: EpochNumber) -> List[List[ValidatorIndex]]
//    """
//    Shuffle ``validators`` into crosslink committees seeded by ``seed`` and ``epoch``.
//    Return a list of ``committees_per_epoch`` committees where each
//    committee is itself a list of validator indices.
//    """
//
//    active_validator_indices = get_active_validator_indices(validators, epoch)
//
//    committees_per_epoch = get_epoch_committee_count(len(active_validator_indices))
//
//    # Shuffle
//    seed = xor(seed, int_to_bytes32(epoch))
//    shuffled_active_validator_indices = shuffle(active_validator_indices, seed)
//
//    # Split the shuffled list into committees_per_epoch pieces
//    return split(shuffled_active_validator_indices, committees_per_epoch)
func Shuffling(
	seed [32]byte,
	validators []*pb.Validator,
	slot uint64) ([][]uint64, error) {

	// Normalize slot to start of epoch boundary.
	slot -= slot % params.BeaconConfig().EpochLength

	// Figure out how many committees can be in a single slot.
	activeIndices := ActiveValidatorIndices(validators, slot)
	activeCount := uint64(len(activeIndices))
	committeesPerEpoch := EpochCommitteeCount(activeCount)

	// Convert slot to bytes and xor it with seed.
	slotInBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(slotInBytes, slot)
	seed = bytesutil.ToBytes32(bytesutil.Xor(seed[:], slotInBytes))

	shuffledIndices, err := utils.ShuffleIndices(seed, activeIndices)
	if err != nil {
		return nil, err
	}

	// Split the shuffled list into epoch_length * committees_per_slot pieces.
	return utils.SplitIndices(shuffledIndices, committeesPerEpoch), nil
}

// AttestationParticipants returns the attesting participants indices.
//
// Spec pseudocode definition:
//   def get_attestation_participants(state: BeaconState,
//     attestation_data: AttestationData,
//     bitfield: bytes) -> List[ValidatorIndex]:
//     """
//     Returns the participant indices at for the ``attestation_data`` and ``bitfield``.
//     """
//     # Find the committee in the list with the desired shard
//     crosslink_committees = get_crosslink_committees_at_slot(state, attestation_data.slot)
//
//	   assert attestation_data.shard in [shard for _, shard in crosslink_committees]
//     crosslink_committee = [committee for committee,
//     		shard in crosslink_committees if shard == attestation_data.shard][0]
//
//	   assert verify_bitfield(bitfield, len(crosslink_committee))
//
//     # Find the participating attesters in the committee
//     participants = []
//     for i, validator_index in enumerate(crosslink_committee):
//         aggregation_bit = get_bitfield_bit(bitfield, i)
//         if aggregation_bit == 0b1:
//            participants.append(validator_index)
//    return participants
func AttestationParticipants(
	state *pb.BeaconState,
	attestationData *pb.AttestationData,
	bitfield []byte) ([]uint64, error) {

	// Find the relevant committee.
	crosslinkCommittees, err := CrosslinkCommitteesAtSlot(state, attestationData.Slot, false)
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
	if len(bitfield) != mathutil.CeilDiv8(len(committee)) {
		return nil, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(len(committee)),
			len(bitfield))
	}

	// Find the participating validators in the committee.
	var participants []uint64
	for i, validatorIndex := range committee {
		bitSet, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return nil, fmt.Errorf("could not get participant bitfield: %v", err)
		}
		if bitSet {
			participants = append(participants, validatorIndex)
		}
	}
	return participants, nil
}
