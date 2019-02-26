// Package helpers contains helper functions outlined in ETH2.0 spec:
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#helper-functions
package helpers

import (
	"encoding/binary"
	"errors"
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
//            SHARD_COUNT // SLOTS_PER_EPOCH,
//            active_validator_count // SLOTS_PER_EPOCH // TARGET_COMMITTEE_SIZE,
//        )
//    ) * SLOTS_PER_EPOCH
func EpochCommitteeCount(activeValidatorCount uint64) uint64 {
	var minCommitteePerSlot = uint64(1)

	// Max committee count per slot will be 0 when shard count is less than epoch length, this
	// covers the special case to ensure there's always 1 max committee count per slot.
	var maxCommitteePerSlot = minCommitteePerSlot
	if params.BeaconConfig().ShardCount/params.BeaconConfig().SlotsPerEpoch > minCommitteePerSlot {
		maxCommitteePerSlot = params.BeaconConfig().ShardCount / params.BeaconConfig().SlotsPerEpoch
	}

	var currCommitteePerSlot = activeValidatorCount / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize

	if currCommitteePerSlot > maxCommitteePerSlot {
		return maxCommitteePerSlot * params.BeaconConfig().SlotsPerEpoch
	}
	if currCommitteePerSlot < 1 {
		return minCommitteePerSlot * params.BeaconConfig().SlotsPerEpoch
	}
	return currCommitteePerSlot * params.BeaconConfig().SlotsPerEpoch
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
		state.ValidatorRegistry, state.CurrentShufflingEpoch)
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
		state.ValidatorRegistry, state.PreviousShufflingEpoch)
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
//    if epoch == current_epoch:
//        committees_per_epoch = get_current_epoch_committee_count(state)
//        seed = state.current_epoch_seed
//        shuffling_epoch = state.current_calculation_epoch
//        shuffling_start_shard = state.current_epoch_start_shard
//    elif epoch == previous_epoch:
//        committees_per_epoch = get_previous_epoch_committee_count(state)
//        seed = state.previous_shuffling_seed
//        shuffling_epoch = state.previous_shuffling_epoch
//        shuffling_start_shard = state.previous_shuffling_start_shard
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
//    offset = slot % SLOTS_PER_EPOCH
//    committees_per_slot = committees_per_epoch // SLOTS_PER_EPOCH
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

	if wantedEpoch == currentEpoch {
		committeesPerEpoch = PrevEpochCommitteeCount(state)
		seed = bytesutil.ToBytes32(state.CurrentShufflingSeedHash32)
		shufflingEpoch = state.CurrentShufflingEpoch
		shufflingStartShard = state.CurrentShufflingStartShard
	} else if wantedEpoch == prevEpoch {
		committeesPerEpoch = PrevEpochCommitteeCount(state)
		seed = bytesutil.ToBytes32(state.PreviousShufflingSeedHash32)
		shufflingEpoch = state.PreviousShufflingEpoch
		shufflingStartShard = state.PreviousShufflingStartShard
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
			shufflingStartShard = (state.CurrentShufflingStartShard + currentCommitteesPerEpoch) %
				params.BeaconConfig().ShardCount
		} else if epochsSinceLastRegistryUpdate > 1 &&
			mathutil.IsPowerOf2(epochsSinceLastRegistryUpdate) {
			seed, err = GenerateSeed(state, nextEpoch)
			if err != nil {
				return nil, fmt.Errorf("could not generate seed: %v", err)
			}
			shufflingStartShard = state.CurrentShufflingStartShard
		} else {
			seed = bytesutil.ToBytes32(state.CurrentShufflingSeedHash32)
			shufflingStartShard = state.CurrentShufflingStartShard
		}
	}

	shuffledIndices, err := Shuffling(
		seed,
		state.ValidatorRegistry,
		shufflingEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not shuffle epoch validators: %v", err)
	}

	offSet := slot % params.BeaconConfig().SlotsPerEpoch
	committeesPerSlot := committeesPerEpoch / params.BeaconConfig().SlotsPerEpoch
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
	slot -= slot % params.BeaconConfig().SlotsPerEpoch

	// Figure out how many committees can be in a single slot.
	activeIndices := ActiveValidatorIndices(validators, slot)
	activeCount := uint64(len(activeIndices))
	committeesPerEpoch := EpochCommitteeCount(activeCount)

	// Convert slot to bytes and xor it with seed.
	slotInBytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(slotInBytes, slot)
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
	var registryChanged bool
	if state.ValidatorRegistryUpdateEpoch == SlotToEpoch(attestationData.Slot)-1 &&
		state.ValidatorRegistryUpdateEpoch != params.BeaconConfig().GenesisEpoch {
		registryChanged = true
	}
	crosslinkCommittees, err := CrosslinkCommitteesAtSlot(state, attestationData.Slot, registryChanged)
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

	if isValidated, err := VerifyBitfield(bitfield, len(committee)); !isValidated || err != nil {
		if err != nil {
			return nil, err
		}

		return nil, errors.New("bitfield is unable to be verified")
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

// VerifyBitfield validates a bitfield with a given committee size.
//
// Spec pseudocode:
//
// def verify_bitfield(bitfield: bytes, committee_size: int) -> bool:
// """
// Verify ``bitfield`` against the ``committee_size``.
// """
// if len(bitfield) != (committee_size + 7) // 8:
// return False
//
// # Check `bitfield` is padded with zero bits only
// for i in range(committee_size, len(bitfield) * 8):
// if get_bitfield_bit(bitfield, i) == 0b1:
// return False
//
// return True
func VerifyBitfield(bitfield []byte, committeeSize int) (bool, error) {
	if len(bitfield) != mathutil.CeilDiv8(committeeSize) {
		return false, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(committeeSize),
			len(bitfield))
	}

	for i := committeeSize; i < len(bitfield)*8; i++ {
		bitSet, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return false, fmt.Errorf("unable to check bit in bitfield %v", err)
		}

		if bitSet {
			return false, nil
		}
	}

	return true, nil
}

// NextEpochCommitteeAssignment query slots in the next epoch
// for it to discover which shard and slot a validator gets assigned.
//
// Spec pseudocode definition:
//   def get_next_epoch_committee_assignment(
//        state: BeaconState,
//        validator_index: ValidatorIndex,
//        registry_change: bool) -> Tuple[List[ValidatorIndex], ShardNumber, SlotNumber, bool]:
//    """
//    Return the committee assignment in the next epoch for ``validator_index`` and ``registry_change``.
//    ``assignment`` returned is a tuple of the following form:
//        * ``assignment[0]`` is the list of validators in the committee
//        * ``assignment[1]`` is the shard to which the committee is assigned
//        * ``assignment[2]`` is the slot at which the committee is assigned
//        * ``assignment[3]`` is a bool signalling if the validator is expected to propose
//            a beacon block at the assigned slot.
//    """
//    current_epoch = get_current_epoch(state)
//    next_epoch = current_epoch + 1
//    next_epoch_start_slot = get_epoch_start_slot(next_epoch)
//    for slot in range(next_epoch_start_slot, next_epoch_start_slot + SLOTS_PER_EPOCH):
//        crosslink_committees = get_crosslink_committees_at_slot(
//            state,
//            slot,
//            registry_change=registry_change,
//        )
//        selected_committees = [
//            committee  # Tuple[List[ValidatorIndex], ShardNumber]
//            for committee in crosslink_committees
//            if validator_index in committee[0]
//        ]
//        if len(selected_committees) > 0:
//            validators = selected_committees[0][0]
//            shard = selected_committees[0][1]
//            first_committee_at_slot = crosslink_committees[0][0]  # List[ValidatorIndex]
//            is_proposer = first_committee_at_slot[slot % len(first_committee_at_slot)] == validator_index
//
//            assignment = (validators, shard, slot, is_proposer)
//            return assignment
func NextEpochCommitteeAssignment(
	state *pb.BeaconState,
	index uint64,
	registryChange bool) ([]uint64, uint64, uint64, bool, error) {
	var selectedCommittees []*CrosslinkCommittee
	nextEpoch := NextEpoch(state)
	nextEpochStartSlot := StartSlot(nextEpoch)
	for slot := nextEpochStartSlot; slot < nextEpochStartSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		crosslinkCommittees, err := CrosslinkCommitteesAtSlot(
			state, slot, registryChange)
		if err != nil {
			return []uint64{}, 0, 0, false, fmt.Errorf("could not get crosslink committee: %v", err)
		}
		for _, committee := range crosslinkCommittees {
			for _, idx := range committee.Committee {
				if idx == index {
					selectedCommittees = append(selectedCommittees, committee)
				}

				if len(selectedCommittees) > 0 {
					validators := selectedCommittees[0].Committee
					shard := selectedCommittees[0].Shard
					firstCommitteeAtSlot := crosslinkCommittees[0].Committee
					isProposer := firstCommitteeAtSlot[slot%
						uint64(len(firstCommitteeAtSlot))] == index
					return validators, shard, slot, isProposer, nil
				}
			}
		}
	}
	return []uint64{}, 0, 0, false, fmt.Errorf("could not get assignment validator %d", index)
}
