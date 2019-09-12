// Package helpers contains helper functions outlined in ETH2.0 spec beacon chain spec
package helpers

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var shuffledIndicesCache = cache.NewShuffledIndicesCache()
var startShardCache = cache.NewStartShardCache()

// CommitteeCount returns the number of crosslink committees of an epoch.
//
// Spec pseudocode definition:
//   def get_committee_count(state: BeaconState, epoch: Epoch) -> uint64:
//    """
//    Return the number of committees at ``epoch``.
//    """
//    committees_per_slot = max(1, min(
//        SHARD_COUNT // SLOTS_PER_EPOCH,
//        len(get_active_validator_indices(state, epoch)) // SLOTS_PER_EPOCH // TARGET_COMMITTEE_SIZE,
//    ))
//    return committees_per_slot * SLOTS_PER_EPOCH
func CommitteeCount(state *pb.BeaconState, epoch uint64) (uint64, error) {
	minCommitteePerSlot := uint64(1)
	// Max committee count per slot will be 0 when shard count is less than epoch length, this
	// covers the special case to ensure there's always 1 max committee count per slot.
	var committeeSizesPerSlot = minCommitteePerSlot
	if params.BeaconConfig().ShardCount/params.BeaconConfig().SlotsPerEpoch > minCommitteePerSlot {
		committeeSizesPerSlot = params.BeaconConfig().ShardCount / params.BeaconConfig().SlotsPerEpoch
	}
	count, err := ActiveValidatorCount(state, epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get active count")
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

// CrosslinkCommittee returns the crosslink committee of a given epoch.
//
// Spec pseudocode definition:
//   def get_crosslink_committee(state: BeaconState, epoch: Epoch, shard: Shard) -> Sequence[ValidatorIndex]:
//    """
//    Return the crosslink committee at ``epoch`` for ``shard``.
//    """
//    return compute_committee(
//        indices=get_active_validator_indices(state, epoch),
//        seed=get_seed(state, epoch),
//        index=(shard + SHARD_COUNT - get_start_shard(state, epoch)) % SHARD_COUNT,
//        count=get_committee_count(state, epoch),
//    )
func CrosslinkCommittee(state *pb.BeaconState, epoch uint64, shard uint64) ([]uint64, error) {
	seed, err := Seed(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seed")
	}

	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active indices")
	}

	startShard, err := StartShard(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get start shard")
	}

	shardCount := params.BeaconConfig().ShardCount
	currentShard := (shard + shardCount - startShard) % shardCount
	committeeCount, err := CommitteeCount(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get committee count")
	}

	return ComputeCommittee(indices, seed, currentShard, committeeCount)
}

// ComputeCommittee returns the requested shuffled committee out of the total committees using
// validator indices and seed.
//
// Spec pseudocode definition:
//  def compute_committee(indices: Sequence[ValidatorIndex],
//                      seed: Hash,
//                      index: uint64,
//                      count: uint64) -> Sequence[ValidatorIndex]:
//    """
//    Return the committee corresponding to ``indices``, ``seed``, ``index``, and committee ``count``.
//    """
//    start = (len(indices) * index) // count
//    end = (len(indices) * (index + 1)) // count
//    return [indices[compute_shuffled_index(ValidatorIndex(i), len(indices), seed)] for i in range(start, end)
func ComputeCommittee(
	validatorIndices []uint64,
	seed [32]byte,
	index uint64,
	totalCommittees uint64,
) ([]uint64, error) {
	validatorCount := uint64(len(validatorIndices))
	start := SplitOffset(validatorCount, totalCommittees, index)
	end := SplitOffset(validatorCount, totalCommittees, index+1)

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
		permutedIndex, err := ShuffledIndex(i, validatorCount, seed)
		if err != nil {
			return []uint64{}, errors.Wrapf(err, "could not get shuffled index at index %d", i)
		}
		shuffledIndices[i-start] = validatorIndices[permutedIndex]
	}
	if err := shuffledIndicesCache.AddShuffledValidatorList(&cache.IndicesByIndexSeed{
		Index:           index,
		Seed:            seed[:],
		ShuffledIndices: shuffledIndices,
	}); err != nil {
		return []uint64{}, errors.Wrap(err, "could not add shuffled indices list to cache")
	}
	return shuffledIndices, nil
}

// AttestingIndices returns the attesting participants indices from the attestation data.
//
// Spec pseudocode definition:
//   def get_attesting_indices(state: BeaconState,
//                          data: AttestationData,
//                          bits: Bitlist[MAX_VALIDATORS_PER_COMMITTEE]) -> Set[ValidatorIndex]:
//    """
//    Return the set of attesting indices corresponding to ``data`` and ``bits``.
//    """
//    committee = get_crosslink_committee(state, data.target.epoch, data.crosslink.shard)
//    return set(index for i, index in enumerate(committee) if bits[i])
func AttestingIndices(state *pb.BeaconState, data *ethpb.AttestationData, bf bitfield.Bitfield) ([]uint64, error) {
	committee, err := CrosslinkCommittee(state, data.Target.Epoch, data.Crosslink.Shard)
	if err != nil {
		return nil, errors.Wrap(err, "could not get committee")
	}

	indices := make([]uint64, 0, len(committee))
	indicesSet := make(map[uint64]bool)
	for i, idx := range committee {
		if !indicesSet[idx] {
			if bf.BitAt(uint64(i)) {
				indices = append(indices, idx)
			}
		}
		indicesSet[idx] = true
	}
	return indices, nil
}

// VerifyBitfieldLength verifies that a bitfield length matches the given committee size.
func VerifyBitfieldLength(bf bitfield.Bitfield, committeeSize uint64) error {
	if bf.Len() != committeeSize {
		return fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			committeeSize,
			bf.Len())
	}
	return nil
}

// CommitteeAssignment is used to query committee assignment from
// current and previous epoch.
//
// Spec pseudocode definition:
//   def get_committee_assignment(state: BeaconState,
//                             epoch: Epoch,
//                             validator_index: ValidatorIndex) -> Optional[Tuple[Sequence[ValidatorIndex], Shard, Slot]]:
//    """
//    Return the committee assignment in the ``epoch`` for ``validator_index``.
//    ``assignment`` returned is a tuple of the following form:
//        * ``assignment[0]`` is the list of validators in the committee
//        * ``assignment[1]`` is the shard to which the committee is assigned
//        * ``assignment[2]`` is the slot at which the committee is assigned
//    Return None if no assignment.
//    """
//    next_epoch = get_current_epoch(state) + 1
//    assert epoch <= next_epoch
//
//    committees_per_slot = get_committee_count(state, epoch) // SLOTS_PER_EPOCH
//    start_slot = compute_start_slot_of_epoch(epoch)
//    for slot in range(start_slot, start_slot + SLOTS_PER_EPOCH):
//        offset = committees_per_slot * (slot % SLOTS_PER_EPOCH)
//        slot_start_shard = (get_start_shard(state, epoch) + offset) % SHARD_COUNT
//        for i in range(committees_per_slot):
//            shard = Shard((slot_start_shard + i) % SHARD_COUNT)
//            committee = get_crosslink_committee(state, epoch, shard)
//            if validator_index in committee:
//                return committee, shard, Slot(slot)
//    return None
func CommitteeAssignment(
	state *pb.BeaconState,
	epoch uint64,
	validatorIndex uint64) ([]uint64, uint64, uint64, bool, error) {

	if epoch > NextEpoch(state) {
		return nil, 0, 0, false, fmt.Errorf(
			"epoch %d can't be greater than next epoch %d",
			epoch, NextEpoch(state))
	}

	committeeCount, err := CommitteeCount(state, epoch)
	if err != nil {
		return nil, 0, 0, false, errors.Wrap(err, "could not get committee count")
	}
	committeesPerSlot := committeeCount / params.BeaconConfig().SlotsPerEpoch

	epochStartShard, err := StartShard(state, epoch)
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
			committee, err := CrosslinkCommittee(state, epoch, shard)
			if err != nil {
				return nil, 0, 0, false, fmt.Errorf(
					"could not get crosslink committee: %v", err)
			}
			for _, index := range committee {
				if validatorIndex == index {
					state.Slot = slot
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

	return []uint64{}, 0, 0, false, status.Error(codes.NotFound, "validator not found in assignments")
}

// ShardDelta returns the minimum number of shards get processed in one epoch.
//
// Note: if you already have the committee count,
// use ShardDeltaFromCommitteeCount as CommitteeCount (specifically
// ActiveValidatorCount) iterates over the entire validator set.
//
// Spec pseudocode definition:
//  def get_shard_delta(state: BeaconState, epoch: Epoch) -> uint64:
//    """
//    Return the number of shards to increment ``state.start_shard`` at ``epoch``.
//    """
//    return min(get_committee_count(state, epoch), SHARD_COUNT - SHARD_COUNT // SLOTS_PER_EPOCH)
func ShardDelta(beaconState *pb.BeaconState, epoch uint64) (uint64, error) {
	committeeCount, err := CommitteeCount(beaconState, epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get committee count")
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

// StartShard returns the start shard used to process crosslink
// of a given epoch. The start shard is cached using epoch as key,
// it gets rewritten where there's a reorg or a new finalized block.
//
// Spec pseudocode definition:
//   def get_start_shard(state: BeaconState, epoch: Epoch) -> Shard:
//    """
//    Return the start shard of the 0th committee at ``epoch``.
//    """
//    assert epoch <= get_current_epoch(state) + 1
//    check_epoch = Epoch(get_current_epoch(state) + 1)
//    shard = Shard((state.start_shard + get_shard_delta(state, get_current_epoch(state))) % SHARD_COUNT)
//    while check_epoch > epoch:
//        check_epoch -= Epoch(1)
//        shard = Shard((shard + SHARD_COUNT - get_shard_delta(state, check_epoch)) % SHARD_COUNT)
//    return shard
func StartShard(state *pb.BeaconState, epoch uint64) (uint64, error) {
	startShard, err := startShardCache.StartShardInEpoch(epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not retrieve start shard from cache")
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
		return 0, errors.Wrap(err, "could not get shard delta")
	}

	startShard = (state.StartShard + delta) % params.BeaconConfig().ShardCount
	for checkEpoch > epoch {
		checkEpoch--
		d, err := ShardDelta(state, checkEpoch)
		if err != nil {
			return 0, errors.Wrap(err, "could not get shard delta")
		}
		startShard = (startShard + params.BeaconConfig().ShardCount - d) % params.BeaconConfig().ShardCount
	}

	if err := startShardCache.AddStartShard(&cache.StartShardByEpoch{
		Epoch:      epoch,
		StartShard: startShard,
	}); err != nil {
		return 0, errors.Wrap(err, "could not save start shard for cache")
	}

	return startShard, nil
}

// VerifyAttestationBitfieldLengths verifies that an attestations aggregation and custody bitfields are
// a valid length matching the size of the committee.
func VerifyAttestationBitfieldLengths(bState *pb.BeaconState, att *ethpb.Attestation) error {
	committee, err := CrosslinkCommittee(bState, att.Data.Target.Epoch, att.Data.Crosslink.Shard)
	if err != nil {
		return errors.Wrap(err, "could not retrieve crosslink committees")
	}

	if committee == nil {
		return errors.New("no committee exist for shard in the attestation")
	}

	if err := VerifyBitfieldLength(att.AggregationBits, uint64(len(committee))); err != nil {
		return errors.Wrap(err, "failed to verify aggregation bitfield")
	}
	if err := VerifyBitfieldLength(att.CustodyBits, uint64(len(committee))); err != nil {
		return errors.Wrap(err, "failed to verify custody bitfield")
	}
	return nil
}

// CompactCommitteesRoot returns the index root of a given epoch.
//
// Spec pseudocode definition:
//   def get_compact_committees_root(state: BeaconState, epoch: Epoch) -> Hash:
//    """
//    Return the compact committee root at ``epoch``.
//    """
//    committees = [CompactCommittee() for _ in range(SHARD_COUNT)]
//    start_shard = get_epoch_start_shard(state, epoch)
//    for committee_number in range(get_epoch_committee_count(state, epoch)):
//        shard = Shard((start_shard + committee_number) % SHARD_COUNT)
//        for index in get_crosslink_committee(state, epoch, shard):
//            validator = state.validators[index]
//            committees[shard].pubkeys.append(validator.pubkey)
//            compact_balance = validator.effective_balance // EFFECTIVE_BALANCE_INCREMENT
//            # `index` (top 6 bytes) + `slashed` (16th bit) + `compact_balance` (bottom 15 bits)
//            compact_validator = uint64((index << 16) + (validator.slashed << 15) + compact_balance)
//            committees[shard].compact_validators.append(compact_validator)
//    return hash_tree_root(Vector[CompactCommittee, SHARD_COUNT](committees))
func CompactCommitteesRoot(state *pb.BeaconState, epoch uint64) ([32]byte, error) {
	shardCount := params.BeaconConfig().ShardCount
	switch shardCount {
	case 1024:
		compactCommArray := [1024]*pb.CompactCommittee{}
		for i := range compactCommArray {
			compactCommArray[i] = &pb.CompactCommittee{}
		}
		comCount, err := CommitteeCount(state, epoch)
		if err != nil {
			return [32]byte{}, err
		}
		startShard, err := StartShard(state, epoch)
		if err != nil {
			return [32]byte{}, err
		}

		for i := uint64(0); i < comCount; i++ {
			shard := (startShard + i) % shardCount
			crossComm, err := CrosslinkCommittee(state, epoch, shard)
			if err != nil {
				return [32]byte{}, err
			}

			for _, index := range crossComm {
				validator := state.Validators[index]
				compactCommArray[shard].Pubkeys = append(compactCommArray[shard].Pubkeys, validator.PublicKey)
				compactValidator := compressValidator(validator, index)
				compactCommArray[shard].CompactValidators = append(compactCommArray[shard].CompactValidators, compactValidator)
			}
		}
		return ssz.HashTreeRoot(compactCommArray)
	case 8:
		compactCommArray := [8]*pb.CompactCommittee{}
		for i := range compactCommArray {
			compactCommArray[i] = &pb.CompactCommittee{}
		}
		comCount, err := CommitteeCount(state, epoch)
		if err != nil {
			return [32]byte{}, err
		}
		startShard, err := StartShard(state, epoch)
		if err != nil {
			return [32]byte{}, err
		}
		for i := uint64(0); i < comCount; i++ {
			shard := (startShard + i) % shardCount
			crossComm, err := CrosslinkCommittee(state, epoch, shard)
			if err != nil {
				return [32]byte{}, err
			}

			for _, index := range crossComm {
				validator := state.Validators[index]
				compactCommArray[shard].Pubkeys = append(compactCommArray[shard].Pubkeys, validator.PublicKey)
				compactValidator := compressValidator(validator, index)
				compactCommArray[shard].CompactValidators = append(compactCommArray[shard].CompactValidators, compactValidator)
			}
		}
		return ssz.HashTreeRoot(compactCommArray)
	default:
		return [32]byte{}, fmt.Errorf("expected minimal or mainnet config shard count, received %d", shardCount)
	}

}

// compressValidator compacts all the validator data such as validator index, slashing info and balance
// into a single uint64 field.
//
// Spec reference:
//   # `index` (top 6 bytes) + `slashed` (16th bit) + `compact_balance` (bottom 15 bits)
//   compact_validator = uint64((index << 16) + (validator.slashed << 15) + compact_balance)
func compressValidator(validator *ethpb.Validator, idx uint64) uint64 {
	compactBalance := validator.EffectiveBalance / params.BeaconConfig().EffectiveBalanceIncrement
	// index (top 6 bytes) + slashed (16th bit) + compact_balance (bottom 15 bits)
	compactIndex := idx << 16
	var slashedBit uint64
	if validator.Slashed {
		slashedBit = 1 << 15
	}
	// Clear all bits except last 15.
	compactBalance &= 0x7FFF // 0b01111111 0b11111111
	compactValidator := compactIndex | slashedBit | compactBalance
	return compactValidator
}
