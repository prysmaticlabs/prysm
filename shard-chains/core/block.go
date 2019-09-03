package core

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ProcessShardBlockSizeFee processes the block fee based on the size of the block.
//
// Spec pseudocode definition:
//  def process_shard_block_size_fee(state: BeaconState, shard_state: ShardState, block: ShardBlock) -> None:
//    # Charge proposer block size fee
//    proposer_index = get_shard_proposer_index(state, state.shard, block.data.slot)
//    block_size = SHARD_HEADER_SIZE + len(block.data.body)
//    add_fee(state, shard_state, proposer_index, state.block_size_price * block_size // SHARD_BLOCK_SIZE_LIMIT)
//    # Calculate new block size price
//    if block_size > SHARD_BLOCK_SIZE_TARGET:
//        size_delta = block_size - SHARD_BLOCK_SIZE_TARGET
//        price_delta = Gwei(state.block_size_price * size_delta // SHARD_BLOCK_SIZE_LIMIT // BLOCK_SIZE_PRICE_QUOTIENT)
//        # The maximum gas price caps the amount burnt on gas fees within a period to 32 ETH
//        MAX_BLOCK_SIZE_PRICE = MAX_EFFECTIVE_BALANCE // EPOCHS_PER_SHARD_PERIOD // SHARD_SLOTS_PER_EPOCH
//        state.block_size_price = min(MAX_BLOCK_SIZE_PRICE, state.block_size_price + price_delta)
//    else:
//        size_delta = SHARD_BLOCK_SIZE_TARGET - block_size
//        price_delta = Gwei(state.block_size_price * size_delta // SHARD_BLOCK_SIZE_LIMIT // BLOCK_SIZE_PRICE_QUOTIENT)
//        state.block_size_price = max(MIN_BLOCK_SIZE_PRICE, state.block_size_price - price_delta)
func AddReward(beaconState *pb.BeaconState, shardState *ethpb.ShardState, index uint64, delta uint64) (*ethpb.ShardState, error) {
	epoch := ShardSlotToEpoch(beaconState.Slot)
	olderEpoch := ShardPeriodStartEpoch(epoch, 2)
	newerEpoch := ShardPeriodStartEpoch(epoch, 1)
	olderCommittee, err := PeriodCommittee(beaconState, shardState.Shard, olderEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get older committee")
	}
	newerCommittee, err := PeriodCommittee(beaconState, shardState.Shard, newerEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get newer committee")
	}

	for _, i := range olderCommittee {
		if i == index {
			shardState.OlderCommitteeRewards[i] += delta
			return shardState, nil
		}
	}

	for _, i := range newerCommittee {
		if i == index {
			shardState.NewerCommitteeRewards[i] += delta
			return shardState, nil
		}
	}

	return shardState, nil
}