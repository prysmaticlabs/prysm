// Package helpers contains helper functions outlined in ETH2.0 spec beacon chain spec
package helpers

import (
	"errors"
	"fmt"
	"sort"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var shuffledIndicesCache = cache.NewShuffledIndicesCache()
var startShardCache = cache.NewStartShardCache()

// EpochCommitteeCount returns the number of crosslink committees of an epoch.
//
// Spec pseudocode definition:
//   def get_epoch_committee_count(state: BeaconState, epoch: Epoch) -> int:
//    """
//    Return the number of committees at ``epoch``.
//    """
//    active_validator_indices = get_active_validator_indices(state, epoch)
//    return max(
//        1,
//        min(
//            SHARD_COUNT // SLOTS_PER_EPOCH,
//            len(active_validator_indices) // SLOTS_PER_EPOCH // TARGET_COMMITTEE_SIZE,
//        )
//    ) * SLOTS_PER_EPOCH
func EpochCommitteeCount(state *pb.BeaconState, epoch uint64) (uint64, error) {
	minCommitteePerSlot := uint64(1)
	// Max committee count per slot will be 0 when shard count is less than epoch length, this
	// covers the special case to ensure there's always 1 max committee count per slot.
	var committeeSizesPerSlot = minCommitteePerSlot
	if params.BeaconConfig().ShardCount/params.BeaconConfig().SlotsPerEpoch > minCommitteePerSlot {
		committeeSizesPerSlot = params.BeaconConfig().ShardCount / params.BeaconConfig().SlotsPerEpoch
	}
	count, err := ActiveValidatorCount(state, epoch)
	if err != nil {
		return 0, fmt.Errorf("could not get active count: %v", err)
	}

	var currCommitteePerSlot = count / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize

	if currCommitteePerSlot > committeeSizesPerSlot {
		return committeeSizesPerSlot * params.BeaconConfig().SlotsPerEpoch, nil
	}
	if currCommitteePerSlot < 1 {
		return minCommitteePerSlot * params.BeaconConfig().SlotsPerEpoch, nil
	}
	return currCommitteePerSlot * params.BeaconConfig().SlotsPerEpoch, nil
}

// CrosslinkCommitteeAtEpoch returns the crosslink committee of a given epoch.
//
// Spec pseudocode definition:
//   def get_crosslink_committee(state: BeaconState, epoch: Epoch, shard: Shard) -> List[ValidatorIndex]:
//    return compute_committee(
//        indices=get_active_validator_indices(state, epoch),
//        seed=generate_seed(state, epoch),
//        index=(shard + SHARD_COUNT - get_epoch_start_shard(state, epoch)) % SHARD_COUNT,
//        count=get_epoch_committee_count(state, epoch),
//    )
func CrosslinkCommitteeAtEpoch(state *pb.BeaconState, epoch uint64, shard uint64) ([]uint64, error) {
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		return nil, fmt.Errorf("could not get seed: %v", err)
	}

	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, fmt.Errorf("could not get active indices: %v", err)
	}

	startShard, err := EpochStartShard(state, epoch)
	if err != nil {
		return nil, fmt.Errorf("could not get start shard: %v", err)
	}

	shardCount := params.BeaconConfig().ShardCount
	currentShard := (shard + shardCount - startShard) % shardCount
	committeeCount, err := EpochCommitteeCount(state, epoch)
	if err != nil {
		return nil, fmt.Errorf("could not get committee count: %v", err)
	}

	return ComputeCommittee(indices, seed, currentShard, committeeCount)
}

// ComputeCommittee returns the requested shuffled committee out of the total committees using
// validator indices and seed.
//
// Spec pseudocode definition:
//  def compute_committee(indices: List[ValidatorIndex], seed: Bytes32, index: int, count: int) -> List[ValidatorIndex]:
//    start = (len(indices) * index) // count
//    end = (len(indices) * (index + 1)) // count
//    return [indices[get_shuffled_index(i, len(indices), seed)] for i in range(start, end)]
func ComputeCommittee(
	validatorIndices []uint64,
	seed [32]byte,
	index uint64,
	totalCommittees uint64,
) ([]uint64, error) {
	validatorCount := uint64(len(validatorIndices))
	start := utils.SplitOffset(validatorCount, totalCommittees, index)
	end := utils.SplitOffset(validatorCount, totalCommittees, index+1)

	// Use cached shuffled indices list if we have seen the seed before.
	cachedShuffledList, err := shuffledIndicesCache.IndicesByIndexSeed(index, seed[:])
	if err != nil {
		return nil, err
	}
	if cachedShuffledList != nil {
		return cachedShuffledList, nil
	}

	// Save the shuffled indices in cache, this is only needed once per epoch or once per new shard index.
	shuffledIndices := make([]uint64, end-start)
	for i := start; i < end; i++ {
		permutedIndex, err := utils.ShuffledIndex(i, validatorCount, seed)
		if err != nil {
			return []uint64{}, fmt.Errorf("could not get shuffled index at index %d: %v", i, err)
		}
		shuffledIndices[i-start] = validatorIndices[permutedIndex]
	}
	if err := shuffledIndicesCache.AddShuffledValidatorList(&cache.IndicesByIndexSeed{
		Index:           index,
		Seed:            seed[:],
		ShuffledIndices: shuffledIndices,
	}); err != nil {
		return []uint64{}, fmt.Errorf("could not add shuffled indices list to cache: %v", err)
	}
	return shuffledIndices, nil
}

// AttestingIndices returns the attesting participants indices. We removed sorting because it's irrelevant
// in production, we don't need to reduce the surface of possible valid input.
//
// Spec pseudocode definition:
//   def get_attesting_indices(state: BeaconState,
//                          attestation_data: AttestationData,
//                          bitfield: bytes) -> List[ValidatorIndex]:
//    """
//    Return the sorted attesting indices corresponding to ``attestation_data`` and ``bitfield``.
//    """
//    committee = get_crosslink_committee(state, attestation_data.target_epoch, attestation_data.crosslink.shard)
//    assert verify_bitfield(bitfield, len(committee))
//    return sorted([index for i, index in enumerate(committee) if get_bitfield_bit(bitfield, i) == 0b1])
func AttestingIndices(state *pb.BeaconState, data *pb.AttestationData, bitfield []byte) ([]uint64, error) {
	committee, err := CrosslinkCommitteeAtEpoch(state, data.TargetEpoch, data.Crosslink.Shard)
	if err != nil {
		return nil, fmt.Errorf("could not get committee: %v", err)
	}
	if isValidated, err := VerifyBitfield(bitfield, len(committee)); !isValidated || err != nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("bitfield is unable to be verified")
	}

	var attestingIndices []uint64
	for i, index := range committee {
		if mathutil.CeilDiv8(int(i)) > len(bitfield) {
			continue
		}
		if bitutil.BitfieldBit(bitfield, int(i)) == 1 {
			attestingIndices = append(attestingIndices, index)
		}
	}
	sort.SliceStable(attestingIndices, func(i, j int) bool {
		return attestingIndices[i] < attestingIndices[j]
	})
	return attestingIndices, nil
}

// VerifyBitfield validates a bitfield with a given committee size.
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
//        validator_index: ValidatorIndex) -> Tuple[List[ValidatorIndex], Shard, Slot]:
//    """
//    Return the committee assignment in the ``epoch`` for ``validator_index``.
//    ``assignment`` returned is a tuple of the following form:
//        * ``assignment[0]`` is the list of validators in the committee
//        * ``assignment[1]`` is the shard to which the committee is assigned
//        * ``assignment[2]`` is the slot at which the committee is assigned
//    """
//    next_epoch = get_current_epoch(state) + 1
//    assert epoch <= next_epoch
//
//    committees_per_slot = get_epoch_committee_count(state, epoch) // SLOTS_PER_EPOCH
//    epoch_start_slot = get_epoch_start_slot(epoch)
//    for slot in range(epoch_start_slot, epoch_start_slot + SLOTS_PER_EPOCH)
//        offset = committees_per_slot * (slot % SLOTS_PER_EPOCH)
//        slot_start_shard = (get_epoch_start_shard(state, epoch) + offset) % SHARD_COUNT
//        for i in range(committees_per_slot):
//            shard = (slot_start_shard + i) % SHARD_COUNT
//            committee = get_crosslink_committee(state, epoch, shard)
//            if validator_index in committee:
//                return committee, shard, slot
func CommitteeAssignment(
	state *pb.BeaconState,
	epoch uint64,
	validatorIndex uint64) ([]uint64, uint64, uint64, bool, error) {

	if epoch > NextEpoch(state) {
		return nil, 0, 0, false, fmt.Errorf(
			"epoch %d can't be greater than next epoch %d",
			epoch, NextEpoch(state))
	}

	committeeCount, err := EpochCommitteeCount(state, epoch)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("could not get committee count: %v", err)
	}
	committeesPerSlot := committeeCount / params.BeaconConfig().SlotsPerEpoch

	epochStartShard, err := EpochStartShard(state, epoch)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf(
			"could not get epoch start shard: %v", err)
	}
	startSlot := StartSlot(epoch)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		offset := committeesPerSlot * (slot % params.BeaconConfig().SlotsPerEpoch)
		slotStatShard := (epochStartShard + offset) % params.BeaconConfig().ShardCount
		for i := uint64(0); i < committeesPerSlot; i++ {
			shard := (slotStatShard + i) % params.BeaconConfig().ShardCount
			committee, err := CrosslinkCommitteeAtEpoch(state, epoch, shard)
			if err != nil {
				return nil, 0, 0, false, fmt.Errorf(
					"could not get crosslink committee: %v", err)
			}
			for _, index := range committee {
				if validatorIndex == index {
					proposerIndex, err := BeaconProposerIndex(state)
					if err != nil {
						return nil, 0, 0, false, fmt.Errorf(
							"could not check proposer index: %v", err)
					}
					isProposer := proposerIndex == validatorIndex
					return committee, shard, slot, isProposer, nil
				}
			}
		}
	}

	return []uint64{}, 0, 0, false, status.Error(codes.NotFound, "validator not found found in assignments")
}

// ShardDelta returns the minimum number of shards get processed in one epoch.
//
// Note: if you already have the committee count,
// use ShardDeltaFromCommitteeCount as EpochCommitteeCount (specifically
// ActiveValidatorCount) iterates over the entire validator set.
//
// Spec pseudocode definition:
//  def get_shard_delta(state: BeaconState, epoch: Epoch) -> int:
//    return min(get_epoch_committee_count(state, epoch), SHARD_COUNT - SHARD_COUNT // SLOTS_PER_EPOCH)
func ShardDelta(beaconState *pb.BeaconState, epoch uint64) (uint64, error) {
	committeeCount, err := EpochCommitteeCount(beaconState, epoch)
	if err != nil {
		return 0, fmt.Errorf("could not get committee count: %v", err)
	}
	return ShardDeltaFromCommitteeCount(committeeCount), nil
}

// ShardDeltaFromCommitteeCount returns the number of shards that get processed
// in one epoch. This method is the inner logic of ShardDelta.
// Returns the minimum of the committeeCount and maximum shard delta which is
// defined as SHARD_COUNT - SHARD_COUNT // SLOTS_PER_EPOCH.
func ShardDeltaFromCommitteeCount(committeeCount uint64) uint64 {
	shardCount := params.BeaconConfig().ShardCount
	maxShardDelta := shardCount - shardCount/params.BeaconConfig().SlotsPerEpoch
	if committeeCount < maxShardDelta {
		return committeeCount
	}
	return maxShardDelta
}

// EpochStartShard returns the start shard used to process crosslink
// of a given epoch. The start shard is cached using epoch as key,
// it gets rewritten where there's a reorg or a new finalized block.
//
// Spec pseudocode definition:
//   def get_epoch_start_shard(state: BeaconState, epoch: Epoch) -> Shard:
//    assert epoch <= get_current_epoch(state) + 1
//    check_epoch = get_current_epoch(state) + 1
//    shard = (state.latest_start_shard + get_shard_delta(state, get_current_epoch(state))) % SHARD_COUNT
//    while check_epoch > epoch:
//        check_epoch -= 1
//        shard = (shard + SHARD_COUNT - get_shard_delta(state, check_epoch)) % SHARD_COUNT
//    return shard
func EpochStartShard(state *pb.BeaconState, epoch uint64) (uint64, error) {
	startShard, err := startShardCache.StartShardInEpoch(epoch)
	if err != nil {
		return 0, fmt.Errorf("could not retrieve start shard from cache: %v", err)
	}
	if startShard != params.BeaconConfig().FarFutureEpoch {
		return startShard, nil
	}

	currentEpoch := CurrentEpoch(state)
	checkEpoch := currentEpoch + 1
	if epoch > checkEpoch {
		return 0, fmt.Errorf("epoch %d can't be greater than %d",
			epoch, checkEpoch)
	}
	delta, err := ShardDelta(state, currentEpoch)
	if err != nil {
		return 0, fmt.Errorf("could not get shard delta: %v", err)
	}

	startShard = (state.LatestStartShard + delta) % params.BeaconConfig().ShardCount
	for checkEpoch > epoch {
		checkEpoch--
		d, err := ShardDelta(state, checkEpoch)
		if err != nil {
			return 0, fmt.Errorf("could not get shard delta: %v", err)
		}
		startShard = (startShard + params.BeaconConfig().ShardCount - d) % params.BeaconConfig().ShardCount
	}

	if err := startShardCache.AddStartShard(&cache.StartShardByEpoch{
		Epoch:      epoch,
		StartShard: startShard,
	}); err != nil {
		return 0, fmt.Errorf("could not save start shard for cache: %v", err)
	}

	return startShard, nil
}

// VerifyAttestationBitfield verifies that an attestations bitfield is valid in respect
// to the committees at that slot.
func VerifyAttestationBitfield(bState *pb.BeaconState, att *pb.Attestation) (bool, error) {
	committee, err := CrosslinkCommitteeAtEpoch(bState, att.Data.TargetEpoch, att.Data.Crosslink.Shard)
	if err != nil {
		return false, fmt.Errorf("could not retrieve crosslink committees at slot: %v", err)
	}

	if committee == nil {
		return false, fmt.Errorf("no committee exist for shard in the attestation")
	}
	return VerifyBitfield(att.AggregationBitfield, len(committee))
}
