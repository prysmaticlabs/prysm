// Package helpers contains helper functions outlined in ETH2.0 spec:
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#helper-functions
package helpers

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var committeeCache = cache.NewCommitteesCache()

// CrosslinkCommittee defines the validator committee of slot and shard combinations.
type CrosslinkCommittee struct {
	Committee []uint64
	Shard     uint64
}

type shufflingInput struct {
	seed               []byte
	shufflingEpoch     uint64
	slot               uint64
	startShard         uint64
	committeesPerEpoch uint64
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
	activeValidatorCount := uint64(len(ActiveValidatorIndices(beaconState, epoch)))
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
	activeIndices := ActiveValidatorIndices(s, epoch)
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

// crosslinkCommittees breaks down the shuffled indices into list of crosslink committee structs
// which contains of validator indices and the shard they are assigned to.
//
// Spec pseudocode definition:
//   def get_crosslink_committees(state: BeaconState,
//                             seed: Bytes32,
//                             shuffling_epoch: Epoch,
//                             slot: Slot,
//                             start_shard: Shard,
//                             committees_per_epoch: int) -> List[Tuple[List[ValidatorIndex], Shard]]:
//    offset = slot % SLOTS_PER_EPOCH
//    committees_per_slot = committees_per_epoch // SLOTS_PER_EPOCH
//    slot_start_shard = (shuffling_start_shard + committees_per_slot * offset) % SHARD_COUNT
//
//    shuffling = get_shuffling(
//        seed,
//        state.validator_registry,
//        shuffling_epoch,
//    )
//
//    return [
//        (
//            shuffling[committees_per_slot * offset + i],
//            (slot_start_shard + i) % SHARD_COUNT,
//        )
//        for i in range(committees_per_slot)
//    ]
func crosslinkCommittees(state *pb.BeaconState, input *shufflingInput) ([]*CrosslinkCommittee, error) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	offSet := input.slot % slotsPerEpoch
	committeesPerSlot := input.committeesPerEpoch / slotsPerEpoch
	slotStartShard := (input.startShard + committeesPerSlot*offSet) %
		params.BeaconConfig().ShardCount
	requestedEpoch := SlotToEpoch(input.slot)

	shuffledIndices, err := Shuffling(
		bytesutil.ToBytes32(input.seed),
		state.ValidatorRegistry,
		requestedEpoch)
	if err != nil {
		return nil, err
	}

	var crosslinkCommittees []*CrosslinkCommittee
	for i := uint64(0); i < committeesPerSlot; i++ {
		crosslinkCommittees = append(crosslinkCommittees, &CrosslinkCommittee{
			Committee: shuffledIndices[committeesPerSlot*offSet+i],
			Shard:     (slotStartShard + i) % params.BeaconConfig().ShardCount,
		})
	}
	return crosslinkCommittees, nil
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
