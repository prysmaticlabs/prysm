package helpers

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// GenesisShardState returns the genesis state of a particular shard.
//
// Spec pseudocode definition:
//  def get_genesis_shard_state(state: BeaconState, shard: Shard) -> ShardState:
//    return ShardState(
//        shard=shard,
//        slot=ShardSlot(SHARD_GENESIS_EPOCH * SHARD_SLOTS_PER_EPOCH),
//        block_body_price=MIN_BLOCK_BODY_PRICE,
//    )
func GenesisShardState(state *pb.BeaconState, shard uint64) (*ethpb.ShardState, error) {
	return &ethpb.ShardState{
		Shard:          shard,
		Slot:           params.ShardConfig().ShardGenesisEpoch * params.ShardConfig().ShardSlotsPerEpoch,
		BlockSizePrice: params.ShardConfig().MinBlockSizePrices,
	}, nil
}

// GenesisShardBlock returns the genesis block of a particular shard.
//
// Spec pseudocode definition:
//  def get_genesis_shard_block(state: BeaconState, shard: Shard) -> ShardBlock:
//    return ShardBlock(
//        shard=shard,
//        slot=ShardSlot(SHARD_GENESIS_EPOCH * SHARD_SLOTS_PER_EPOCH),
//        state_root=hash_tree_root(get_genesis_shard_state(state, shard)),
//    )
func GenesisShardBlock(state *pb.BeaconState, shard uint64) (*ethpb.ShardBlock, error) {
	genesisState, err := GenesisShardState(state, shard)
	if err != nil {
		return nil, errors.Wrap(err, "could not get shard genesis state")
	}
	stateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash shard genesis state")
	}
	return &ethpb.ShardBlock{
		Shard:     shard,
		Slot:      params.ShardConfig().ShardGenesisEpoch * params.ShardConfig().ShardSlotsPerEpoch,
		StateRoot: stateRoot[:],
	}, nil
}
