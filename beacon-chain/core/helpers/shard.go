package helpers

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ShardFromCommitteeIndex converts the index of a committee into which shard that committee is responsible for
// at the given slot.
//
// Spec code:
// def compute_shard_from_committee_index(state: BeaconState, index: CommitteeIndex, slot: Slot) -> Shard:
//    active_shards = get_active_shard_count(state)
//    return Shard((index + get_start_shard(state, slot)) % active_shards)
//func ShardFromCommitteeIndex(beaconState *state.BeaconState, slot uint64, committeeID uint64) (uint64, error) {
//	activeShards := ActiveShardCount()
//	startShard, err := StartShard(beaconState, slot)
//	if err != nil {
//		return 0, err
//	}
//	return (startShard + committeeID) % activeShards, nil
//}

// StartShard returns the starting shard of a historical epoch
//
// Spec code:
// def get_start_shard(state: BeaconState, slot: Slot) -> Shard:
//    """
//    Return the start shard at ``slot``.
//    """
//    current_epoch_start_slot = compute_start_slot_at_epoch(get_current_epoch(state))
//    shard = state.current_epoch_start_shard
//    if slot > current_epoch_start_slot:
//        # Current epoch or the next epoch lookahead
//        for _slot in range(current_epoch_start_slot, slot):
//            committee_count = get_committee_count_per_slot(state, compute_epoch_at_slot(Slot(_slot)))
//            active_shard_count = get_active_shard_count(state, compute_epoch_at_slot(Slot(_slot)))
//            shard = (shard + committee_count) % active_shard_count
//        return Shard(shard)
//    elif slot < current_epoch_start_slot:
//        # Previous epoch
//        for _slot in list(range(slot, current_epoch_start_slot))[::-1]:
//            committee_count = get_committee_count_per_slot(state, compute_epoch_at_slot(Slot(_slot)))
//            active_shard_count = get_active_shard_count(state, compute_epoch_at_slot(Slot(_slot)))
//            # Ensure positive
//            shard = (shard + active_shard_count - committee_count) % active_shard_count
//    return Shard(shard)
func StartShard(beaconState *state.BeaconState, slot uint64) (uint64, error) {
	currentEpoch := CurrentEpoch(beaconState)
	currentEpochStartSlot, err := StartSlot(currentEpoch)
	if err != nil {
		return 0, err
	}
	shard := beaconState.CurrentEpochStartShard()

	if slot > currentEpochStartSlot {
		for i := currentEpochStartSlot; i < slot; i++ {
			activeValidatorCount, err := ActiveValidatorCount(beaconState, SlotToEpoch(i))
			if err != nil {
				return 0, err
			}
			committeeCount := SlotCommitteeCount(activeValidatorCount)
			activeShardCount := ActiveShardCount()
			shard = (shard + committeeCount) % activeShardCount
		}
		return shard, nil
	} else if slot < currentEpochStartSlot {
		for i := currentEpochStartSlot; i > slot; i-- {
			activeValidatorCount, err := ActiveValidatorCount(beaconState, SlotToEpoch(i))
			if err != nil {
				return 0, err
			}
			committeeCount := SlotCommitteeCount(activeValidatorCount)
			activeShardCount := ActiveShardCount()
			shard = (shard + activeValidatorCount - committeeCount) % activeShardCount
		}
		return shard, nil
	}
	return shard, nil
}

// ActiveShardCount returns the active shard count.
// Currently 64, may be changed in the future.
//
// Spec code:
// def def get_active_shard_count(state: BeaconState, epoch: Epoch) -> uint64: -> uint64:
//    return len(state.shard_states)  # May adapt in the future, or change over time.
//    """
//    Return the number of active shards.
//    Note that this puts an upper bound on the number of committees per slot.
//    """
//    return INITIAL_ACTIVE_SHARDS
func ActiveShardCount() uint64 {
	return params.BeaconConfig().InitialActiveShards
}

// UpdatedGasPrice returns the updated gas price based on the EIP 1599 formulas.
//
// Spec code:
// def compute_updated_gasprice(prev_gasprice: Gwei, shard_block_length: uint64, adjustment_quotient: uint64) -> Gwei:
//    if shard_block_length > TARGET_SAMPLES_PER_BLOCK:
//        delta = max(1, prev_gasprice * (shard_block_length - TARGET_SAMPLES_PER_BLOCK)
//                       // TARGET_SAMPLES_PER_BLOCK // adjustment_quotient)
//        return min(prev_gasprice + delta, MAX_GASPRICE)
//    else:
//        delta = max(1, prev_gasprice * (TARGET_SAMPLES_PER_BLOCK - shard_block_length)
//                       // TARGET_SAMPLES_PER_BLOCK // adjustment_quotient)
//        return max(prev_gasprice, MIN_GASPRICE + delta) - delta
func UpdatedGasPrice(prevGasPrice uint64, shardBlockLength uint64, adjustmentCoefficient uint64) uint64 {
	targetBlockSize := params.BeaconConfig().TargetShardBlockSize
	maxGasPrice := params.BeaconConfig().MaxGasPrice
	minGasPrice := params.BeaconConfig().MinGasPrice
	if shardBlockLength > targetBlockSize {
		delta := prevGasPrice * (shardBlockLength - targetBlockSize) / targetBlockSize / adjustmentCoefficient
		// Max gas price is the upper bound.
		if prevGasPrice+delta > maxGasPrice {
			return maxGasPrice
		}
		return prevGasPrice + delta
	}

	delta := prevGasPrice * (targetBlockSize - shardBlockLength) / targetBlockSize / adjustmentCoefficient
	// Min gas price is the lower bound.
	if prevGasPrice < minGasPrice+delta {
		return minGasPrice
	}
	return prevGasPrice - delta
}

// ShardCommittee returns the shard committee of a given slot and shard.
// The proposer of a shard block is randomly sampled from the shard committee,
//// which changes only once per ~1 day (with committees being computable 1 day ahead of time).

// def get_shard_committee(beacon_state: BeaconState, epoch: Epoch, shard: Shard) -> Sequence[ValidatorIndex]:
//    """
//    Return the shard committee of the given ``epoch`` of the given ``shard``.
//    """
//    source_epoch = compute_committee_source_epoch(epoch, SHARD_COMMITTEE_PERIOD)
//    active_validator_indices = get_active_validator_indices(beacon_state, source_epoch)
//    seed = get_seed(beacon_state, source_epoch, DOMAIN_SHARD_COMMITTEE)
//    return compute_committee(
//        indices=active_validator_indices,
//        seed=seed,
//        index=shard,
//        count=get_active_shard_count(beacon_state, epoch),
//    )
func ShardCommittee(beaconState *state.BeaconState, epoch uint64, shard uint64) ([]uint64, error) {
	se := SourceEpoch(epoch, params.BeaconConfig().ShardCommitteePeriod)
	activeValidatorIndices, err := ActiveValidatorIndices(beaconState, se)
	if err != nil {
		return nil, err
	}
	seed, err := Seed(beaconState, se, params.BeaconConfig().DomainShardCommittee)
	if err != nil {
		return nil, err
	}
	return ComputeCommittee(activeValidatorIndices, seed, shard, ActiveShardCount())
}
