// Package helpers contains helper functions outlined in ETH2.0 spec:
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#helper-functions
package helpers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var committeeCache = cache.NewCommitteesCache()

// CrosslinkCommittee defines the validator committee of slot and shard combinations.
type CrosslinkCommittee struct {
	Committee []uint64
	Shard     uint64
}

// EpochCommitteeCount returns the number of crosslink committees of an epoch.
//
// Spec pseudocode definition:
//   def get_epoch_committee_count(state: BeaconState, epoch: Epoch) -> int:
//    """
//    Return the number of committees in one epoch.
//    """
//    active_validators = get_active_validator_indices(state, epoch)
//    return max(
//        1,
//        min(
//            SHARD_COUNT // SLOTS_PER_EPOCH,
//            len(active_validators) // SLOTS_PER_EPOCH // TARGET_COMMITTEE_SIZE,
//        )
//    ) * SLOTS_PER_EPOCH
func EpochCommitteeCount(beaconState *pb.BeaconState, epoch uint64) uint64 {
	var minCommitteePerSlot = uint64(1)
	activeValidatorCount := uint64(len(ActiveValidatorIndices(beaconState.ValidatorRegistry, epoch)))
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
//        get_current_epoch(state),
//    )
//    return get_epoch_committee_count(len(current_active_validators)
func CurrentEpochCommitteeCount(state *pb.BeaconState) uint64 {
	return EpochCommitteeCount(state, CurrentEpoch(state))
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
//        state.previous_epoch,
//    )
//    return get_epoch_committee_count(len(previous_active_validators))
func PrevEpochCommitteeCount(state *pb.BeaconState) uint64 {
	return EpochCommitteeCount(state, PrevEpoch(state))
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
	return EpochCommitteeCount(state, CurrentEpoch(state)+1)
}

// ComputeCommittee returns the requested shuffled committee out of the total committees using
// validator indices and seed.
//
// Spec pseudocode definition:
//  def compute_committee(validator_indices: List[ValidatorIndex],
//    seed: Bytes32,
//    index: int,
//    total_committees: int) -> List[ValidatorIndex]:
//    """
//    Return the ``index``'th shuffled committee out of a total ``total_committees``
//    using ``validator_indices`` and ``seed``.
//    """
//    start_offset = get_split_offset(len(validator_indices), total_committees, index)
//    end_offset = get_split_offset(len(validator_indices), total_committees, index + 1)
//    return [
//    validator_indices[get_permuted_index(i, len(validator_indices), seed)]
//    for i in range(start_offset, end_offset)
//    ]
func ComputeCommittee(
	validatorIndices []uint64,
	seed [32]byte,
	index uint64,
	totalCommittees uint64,
) ([]uint64, error) {
	validatorCount := uint64(len(validatorIndices))
	startOffset := utils.SplitOffset(validatorCount, totalCommittees, index)
	endOffset := utils.SplitOffset(validatorCount, totalCommittees, index+1)

	indices := make([]uint64, endOffset-startOffset)
	for i := startOffset; i < endOffset; i++ {
		permutedIndex, err := utils.PermutedIndex(i, validatorCount, seed)
		if err != nil {
			return []uint64{}, fmt.Errorf("could not get permuted index at index %d: %v", i, err)
		}
		indices[i-startOffset] = validatorIndices[permutedIndex]
	}
	return indices, nil
}

// CrosslinkCommitteesAtSlot returns the list of crosslink committees, it
// contains the shard associated with the committee and the validator indices
// in that committee.
//
// Spec pseudocode definition:
//  def get_crosslink_committees_at_slot(state: BeaconState,
//    slot: Slot) -> List[Tuple[List[ValidatorIndex], Shard]]:
//    """
//    Return the list of ``(committee, shard)`` tuples for the ``slot``.
//    """
//    epoch = slot_to_epoch(slot)
//    current_epoch = get_current_epoch(state)
//    previous_epoch = get_previous_epoch(state)
//    next_epoch = current_epoch + 1
//
//    assert previous_epoch <= epoch <= next_epoch
//    indices = get_active_validator_indices(state, epoch)
//
//    if epoch == current_epoch:
//      start_shard = state.latest_start_shard
//    elif epoch == previous_epoch:
//      previous_shard_delta = get_shard_delta(state, previous_epoch)
//      start_shard = (state.latest_start_shard - previous_shard_delta) % SHARD_COUNT
//    elif epoch == next_epoch:
//      current_shard_delta = get_shard_delta(state, current_epoch)
//      start_shard = (state.latest_start_shard + current_shard_delta) % SHARD_COUNT
//
//    committees_per_epoch = get_epoch_committee_count(state, epoch)
//    committees_per_slot = committees_per_epoch // SLOTS_PER_EPOCH
//    offset = slot % SLOTS_PER_EPOCH
//    slot_start_shard = (start_shard + committees_per_slot * offset) % SHARD_COUNT
//    seed = generate_seed(state, epoch)
//
//    return [
//      (
//        compute_committee(indices, seed, committees_per_slot * offset + i, committees_per_epoch),
//        (slot_start_shard + i) % SHARD_COUNT,
//      )
//      for i in range(committees_per_slot)
//    ]
func CrosslinkCommitteesAtSlot(state *pb.BeaconState, slot uint64) ([]*CrosslinkCommittee, error) {
	wantedEpoch := SlotToEpoch(slot)
	currentEpoch := CurrentEpoch(state)
	prevEpoch := PrevEpoch(state)
	nextEpoch := NextEpoch(state)

	var startShard uint64
	shardCount := params.BeaconConfig().ShardCount
	switch wantedEpoch {
	case currentEpoch:
		startShard = state.LatestStartShard
	case prevEpoch:
		previousShardDelta := ShardDelta(state, prevEpoch)
		startShard = (state.LatestStartShard - previousShardDelta) % shardCount
	case nextEpoch:
		currentShardDelta := ShardDelta(state, currentEpoch)
		startShard = (state.LatestStartShard + currentShardDelta) % shardCount
	default:
		return nil, fmt.Errorf(
			"input committee epoch %d out of bounds: %d <= epoch <= %d",
			wantedEpoch-params.BeaconConfig().GenesisEpoch,
			prevEpoch-params.BeaconConfig().GenesisEpoch,
			currentEpoch-params.BeaconConfig().GenesisEpoch,
		)
	}

	committeesPerEpoch := EpochCommitteeCount(state, wantedEpoch)
	committeesPerSlot := committeesPerEpoch / params.BeaconConfig().SlotsPerEpoch
	offset := slot % params.BeaconConfig().SlotsPerEpoch
	slotStartShard := (startShard + committeesPerSlot + offset) % shardCount
	seed, err := GenerateSeed(state, wantedEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not generate seed: %v", err)
	}

	indices := ActiveValidatorIndices(state.ValidatorRegistry, wantedEpoch)
	committees := make([]*CrosslinkCommittee, committeesPerSlot)
	for i := uint64(0); i < committeesPerSlot; i++ {
		committee, err := ComputeCommittee(indices, seed, committeesPerSlot*offset+i, committeesPerEpoch)
		if err != nil {
			return nil, fmt.Errorf("could not compute committee: %v", err)
		}
		committees[i] = &CrosslinkCommittee{
			Committee: committee,
			Shard:     (slotStartShard + i) % shardCount,
		}
	}

	return committees, nil
}

// Shuffling shuffles input validator indices and splits them by slot and shard.
//
// Spec pseudocode definition:
//   def get_shuffling(seed: Bytes32,
//                  validators: List[Validator],
//                  epoch: Epoch) -> List[List[ValidatorIndex]]
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
	epoch uint64) ([][]uint64, error) {

	// Figure out how many committees can be in a single epoch.
	s := &pb.BeaconState{ValidatorRegistry: validators}
	activeIndices := ActiveValidatorIndices(s.ValidatorRegistry, epoch)
	committeesPerEpoch := EpochCommitteeCount(s, epoch)

	// Convert slot to bytes and xor it with seed.
	epochInBytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(epochInBytes, epoch)
	seed = bytesutil.ToBytes32(bytesutil.Xor(seed[:], epochInBytes))

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
//   def get_attesting_indices(state: BeaconState,
//     attestation_data: AttestationData,
// 	   bitfield: bytes) -> List[ValidatorIndex]:
//     """
//     Return the sorted attesting indices corresponding to ``attestation_data`` and ``bitfield``.
//     """
//     crosslink_committees = get_crosslink_committees_at_slot(state, attestation_data.slot)
//     crosslink_committee = [committee for committee, shard in crosslink_committees if shard == attestation_data.shard][0]
//     assert verify_bitfield(bitfield, len(crosslink_committee))
//     return sorted([index for i, index in enumerate(crosslink_committee) if get_bitfield_bit(bitfield, i) == 0b1])
func AttestationParticipants(
	state *pb.BeaconState,
	attestationData *pb.AttestationData,
	bitfield []byte) ([]uint64, error) {

	var cachedCommittees *cache.CommitteesInSlot
	var err error
	slot := attestationData.Slot

	// When enabling committee cache, we fetch the committees using slot.
	// If it's not prev cached, we compute for the committees of slot and
	// add it to the cache.
	cachedCommittees, err = committeeCache.CommitteesInfoBySlot(slot)
	if err != nil {
		return nil, err
	}

	if cachedCommittees == nil {
		crosslinkCommittees, err := CrosslinkCommitteesAtSlot(state, slot)
		if err != nil {
			return nil, err
		}
		cachedCommittees = ToCommitteeCache(slot, crosslinkCommittees)

		if err := committeeCache.AddCommittees(cachedCommittees); err != nil {
			return nil, err
		}
	}

	var selectedCommittee []uint64
	for _, committee := range cachedCommittees.Committees {
		if committee.Shard == attestationData.Shard {
			selectedCommittee = committee.Committee
			break
		}
	}

	if selectedCommittee == nil {
		committeesFound := make([]uint64, len(cachedCommittees.Committees))
		for i := 0; i < len(committeesFound); i++ {
			committeesFound[i] = cachedCommittees.Committees[i].Shard
		}
		return nil, fmt.Errorf("could not find a committee with the shard: %d, wanted one of: %v", attestationData.Shard, committeesFound)
	}

	if isValidated, err := VerifyBitfield(bitfield, len(selectedCommittee)); !isValidated || err != nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("bitfield is unable to be verified")
	}

	// Find the participating validators in the committee.
	var participants []uint64
	for i, validatorIndex := range selectedCommittee {
		bitSet, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return nil, fmt.Errorf("could not get participant bitfield: %v", err)
		}
		if bitSet {
			participants = append(participants, validatorIndex)
		}
	}
	sort.Slice(participants, func(i, j int) bool { return participants[i] < participants[j] })
	return participants, nil
}

// VerifyBitfield verifies bitfield against the committee_size.
//
// Spec pseudocode definition:
//   def verify_bitfield(bitfield: bytes, committee_size: int) -> bool:
//     """
//     Verify ``bitfield`` against the ``committee_size``.
//     """
//     if len(bitfield) != (committee_size + 7) // 8:
//         return False
//     # Check `bitfield` is padded with zero bits only
//     for i in range(committee_size, len(bitfield) * 8):
//         if get_bitfield_bit(bitfield, i) == 0b1:
//             return False
//     return True
func VerifyBitfield(bitfield []byte, committeeSize int) (bool, error) {
	if len(bitfield) != mathutil.CeilDiv8(committeeSize) {
		return false, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(committeeSize),
			len(bitfield))
	}
	bitLength := len(bitfield) << 3
	for i := committeeSize; i < bitLength; i++ {
		set, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return false, err
		}
		if set {
			return false, nil
		}
	}
	return true, nil
}

// CommitteeAssignment is used to query committee assignment from
// current and previous epoch.
//
// Spec pseudocode definition:
//   def get_committee_assignment(
//        state: BeaconState,
//        epoch: Epoch,
//        validator_index: ValidatorIndex,
//        registry_change: bool=False) -> Tuple[List[ValidatorIndex], Shard, Slot, bool]:
//    """
//    Return the committee assignment in the ``epoch`` for ``validator_index`` and ``registry_change``.
//    ``assignment`` returned is a tuple of the following form:
//        * ``assignment[0]`` is the list of validators in the committee
//        * ``assignment[1]`` is the shard to which the committee is assigned
//        * ``assignment[2]`` is the slot at which the committee is assigned
//        * ``assignment[3]`` is a bool signaling if the validator is expected to propose
//            a beacon block at the assigned slot.
//    """
//    previous_epoch = get_previous_epoch(state)
//    next_epoch = get_current_epoch(state)
//    assert previous_epoch <= epoch <= next_epoch
//
//    epoch_start_slot = get_epoch_start_slot(epoch)
//    for slot in range(epoch_start_slot, epoch_start_slot + SLOTS_PER_EPOCH):
//        crosslink_committees = get_crosslink_committees_at_slot(
//            state,
//            slot,
//            registry_change=registry_change,
//        )
//        selected_committees = [
//            committee  # Tuple[List[ValidatorIndex], Shard]
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
func CommitteeAssignment(
	state *pb.BeaconState,
	slot uint64,
	validatorIndex uint64,
	registryChange bool) ([]uint64, uint64, uint64, bool, error) {
	var selectedCommittees []*cache.CommitteeInfo

	wantedEpoch := slot / params.BeaconConfig().SlotsPerEpoch
	prevEpoch := PrevEpoch(state)
	nextEpoch := NextEpoch(state)

	if wantedEpoch < prevEpoch || wantedEpoch > nextEpoch {
		return nil, 0, 0, false, fmt.Errorf(
			"epoch %d out of bounds: %d <= epoch <= %d",
			wantedEpoch-params.BeaconConfig().GenesisEpoch,
			prevEpoch-params.BeaconConfig().GenesisEpoch,
			nextEpoch-params.BeaconConfig().GenesisEpoch,
		)
	}

	var cachedCommittees *cache.CommitteesInSlot
	var err error
	startSlot := StartSlot(wantedEpoch)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {

		cachedCommittees, err = committeeCache.CommitteesInfoBySlot(slot)
		if err != nil {
			return []uint64{}, 0, 0, false, err
		}
		if cachedCommittees == nil {
			crosslinkCommittees, err := CrosslinkCommitteesAtSlot(state, slot)
			if err != nil {
				return []uint64{}, 0, 0, false, fmt.Errorf("could not get crosslink committee: %v", err)
			}
			cachedCommittees = ToCommitteeCache(slot, crosslinkCommittees)
			if err := committeeCache.AddCommittees(cachedCommittees); err != nil {
				return []uint64{}, 0, 0, false, err
			}
		}
		for _, committee := range cachedCommittees.Committees {
			for _, idx := range committee.Committee {
				if idx == validatorIndex {
					selectedCommittees = append(selectedCommittees, committee)
				}

				if len(selectedCommittees) > 0 {
					validators := selectedCommittees[0].Committee
					shard := selectedCommittees[0].Shard
					firstCommitteeAtSlot := cachedCommittees.Committees[0].Committee
					isProposer := firstCommitteeAtSlot[slot%
						uint64(len(firstCommitteeAtSlot))] == validatorIndex
					return validators, shard, slot, isProposer, nil
				}
			}
		}
	}
	return []uint64{}, 0, 0, false, status.Error(codes.NotFound, "validator not found found in assignments")
}

// ShardDelta returns the minimum number of shards get processed in one epoch.
//
// Spec pseudocode definition:
// 	def get_shard_delta(state: BeaconState, epoch: Epoch) -> int:
//    return min(get_epoch_committee_count(state, epoch), SHARD_COUNT - SHARD_COUNT // SLOTS_PER_EPOCH)
func ShardDelta(beaconState *pb.BeaconState, epoch uint64) uint64 {
	shardCount := params.BeaconConfig().ShardCount
	minShardDelta := shardCount - shardCount/params.BeaconConfig().SlotsPerEpoch
	if EpochCommitteeCount(beaconState, epoch) < minShardDelta {
		return EpochCommitteeCount(beaconState, epoch)
	}
	return minShardDelta
}

// RestartCommitteeCache restarts the committee cache from scratch.
func RestartCommitteeCache() {
	committeeCache = cache.NewCommitteesCache()
}

// ToCommitteeCache converts crosslink committee object
// into a cache format, to be saved in cache.
func ToCommitteeCache(slot uint64, crosslinkCommittees []*CrosslinkCommittee) *cache.CommitteesInSlot {
	var cacheCommittee []*cache.CommitteeInfo
	for _, crosslinkCommittee := range crosslinkCommittees {
		cacheCommittee = append(cacheCommittee, &cache.CommitteeInfo{
			Committee: crosslinkCommittee.Committee,
			Shard:     crosslinkCommittee.Shard,
		})
	}
	committees := &cache.CommitteesInSlot{
		Slot:       slot,
		Committees: cacheCommittee,
	}

	return committees
}
