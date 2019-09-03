package transition

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ShardStateTransition processes a slots of a shard.
//
// Spec pseudocode definition:
//  def shard_state_transition(state: BeaconState,
//                           shard_state: ShardState,
//                           block: ShardBlock,
//                           validate_state_root: bool=False) -> ShardState:
//    # Process slots (including those with no blocks) since block
//    process_shard_slots(state, shard_state, block.data.slot)
//    # Process block
//    process_shard_block(state, shard_state, block)
//    # Validate state root (`validate_state_root == True` in production)
//    if validate_state_root:
//        assert block.data.state_root == hash_tree_root(shard_state)
//    # Return post-state
//    return shard_state
func ShardStateTransition(beaconState *pb.BeaconState, shardState *ethpb.ShardState, shardBlock *ethpb.ShardBlock) (*ethpb.ShardState, error) {
	var err error
	shardState, err = ProcessShardSlots(beaconState, shardState, shardBlock.Data.Slot)
	if err != nil {
		return nil, errors.Wrapf(err, "could not process shard slots up to %d", shardBlock.Data.Slot)
	}
	shardState, err = ProcessShardBlock(beaconState, shardState, shardBlock)
	if err != nil {
		return nil, errors.Wrapf(err, "could not process shard block", shardBlock.Data.Slot)
	}

	postStateRoot, err := ssz.HashTreeRoot(shardState)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash processed shard state")
	}
	if !bytes.Equal(postStateRoot[:], shardBlock.Data.StateRoot) {
		return nil, fmt.Errorf("validate shard state root failed, wanted: %#x, received: %#x",
			postStateRoot[:], shardBlock.Data.StateRoot)
	}

	return shardState, nil
}
