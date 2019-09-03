package helpers

import (
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ShardSlotToEpoch returns the epoch number of the input shard slot.
//
// Spec pseudocode definition:
//  def compute_epoch_of_shard_slot(slot: ShardSlot) -> Epoch:
//    return compute_epoch_of_slot(slot // SHARD_SLOTS_PER_EPOCH)
func ShardSlotToEpoch(slot uint64) uint64 {
	return slot / params.BeaconConfig().ShardSlotsPerEpoch
}

// ShardPeriodStartEpoch returns the start epoch number of the a
// given shard period.
//
// Spec pseudocode definition:
//  def compute_shard_period_start_epoch(epoch: Epoch, lookback: uint64) -> Epoch:
//    return Epoch(epoch - (epoch % EPOCHS_PER_SHARD_PERIOD) - lookback * EPOCHS_PER_SHARD_PERIOD)
func ShardPeriodStartEpoch(epoch uint64, lookback uint64) uint64 {
	epochsPerShardPeriod := params.BeaconConfig().EpochsPerShardPeriod
	return epoch - (epoch % epochsPerShardPeriod) - lookback*epochsPerShardPeriod
}
