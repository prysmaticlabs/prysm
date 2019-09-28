package transition

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessShardSlots processes a slots of a shard.
//
// Spec pseudocode definition:
//  ef process_shard_slots(shard_state: ShardState, slot: ShardSlot) -> None:
//    assert shard_state.slot <= slot
//    while shard_state.slot < slot:
//        process_shard_slot(shard_state)
//        # Process shard period on the start slot of the next shard period
//        if (shard_state.slot + 1) % (SHARD_SLOTS_PER_EPOCH * EPOCHS_PER_SHARD_PERIOD) == 0:
//            process_shard_period(shard_state)
//        shard_state.slot += ShardSlot(1)
func ProcessShardSlots(shardState *ethpb.ShardState, slot uint64) (*ethpb.ShardState, error) {
	if shardState.Slot > slot {
		return nil, fmt.Errorf("expected shard state.slot %d < slot %d", shardState.Slot, slot)
	}
	for shardState.Slot < slot {
		var err error
		shardState, err = ProcessShardSlot(shardState)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		// Process period on the next slot of the next period
		if (shardState.Slot+1)%(params.ShardConfig().ShardSlotsPerEpoch*params.ShardConfig().EpochsPerShardPeriod) == 0 {
			shardState = ProcessShardPeriod(shardState)
		}
		shardState.Slot++
	}
	return shardState, nil
}

// ProcessShardSlot processes a slot of a shard.
//
// Spec pseudocode definition:
//  def process_shard_slot(shard_state: ShardState) -> None:
//    # Cache state root
//    previous_state_root = hash_tree_root(shard_state)
//    if shard_state.latest_block_header.state_root == Bytes32():
//        shard_state.latest_block_header.state_root = previous_state_root
//    # Cache state root in history accumulator
//    depth = 0
//    while shard_state.slot % 2**depth == 0 and depth < HISTORY_ACCUMULATOR_DEPTH:
//        shard_state.history_accumulator[depth] = previous_state_root
//        depth += 1
func ProcessShardSlot(shardState *ethpb.ShardState) (*ethpb.ShardState, error) {
	// Cache the shard state root
	prevStateRoot, err := ssz.HashTreeRoot(shardState)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash prev state root")
	}
	zeroHash := params.BeaconConfig().ZeroHash
	if bytes.Equal(shardState.LatestBlockHeader.StateRoot, zeroHash[:]) {
		shardState.LatestBlockHeader.StateRoot = prevStateRoot[:]
	}

	// Cache shard state root in history accumulator
	depth := uint64(0)
	twoTotheDepth := uint64(1 << depth)
	for shardState.Slot%twoTotheDepth == 0 && depth < params.ShardConfig().HistoryAccumulatorDepth {
		shardState.HistoryAccumulator[depth] = prevStateRoot[:]
		depth++
	}

	return shardState, nil
}
