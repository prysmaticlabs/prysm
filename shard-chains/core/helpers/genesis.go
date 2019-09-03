package helpers

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/terencechain/prysm-phase2/shared/ssz"
)

// GenesisShardState returns the genesis state of a particular shard.
//
// Spec pseudocode definition:
//  def GenesisShardState(state: BeaconState, shard: Shard) -> ShardState:
//    older_committee = get_period_committee(state, shard, compute_shard_period_start_epoch(SHARD_GENESIS_EPOCH, 2))
//    newer_committee = get_period_committee(state, shard, compute_shard_period_start_epoch(SHARD_GENESIS_EPOCH, 1))
//    return ShardState(
//        shard=shard,
//        slot=ShardSlot(SHARD_GENESIS_EPOCH * SHARD_SLOTS_PER_EPOCH),
//        block_size_price=MIN_BLOCK_SIZE_PRICE,
//        older_committee_rewards=[Gwei(0) for _ in range(len(older_committee))],
//        newer_committee_rewards=[Gwei(0) for _ in range(len(newer_committee))],
//        older_committee_fees=[Gwei(0) for _ in range(len(older_committee))],
//        newer_committee_fees=[Gwei(0) for _ in range(len(newer_committee))],
//    )
func GenesisShardState(state *pb.BeaconState, shard uint64) (*ethpb.ShardState, error) {
	olderEpoch := ShardPeriodStartEpoch(params.BeaconConfig().ShardGenesisEpoch, 2)
	newerEpoch := ShardPeriodStartEpoch(params.BeaconConfig().ShardGenesisEpoch, 1)
	olderCommittee, err := PeriodCommittee(state, shard, olderEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get older committee")
	}
	newerCommittee, err := PeriodCommittee(state, shard, newerEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get newer committee")
	}
	return &ethpb.ShardState{
		Shard:                 shard,
		Slot:                  params.BeaconConfig().ShardGenesisEpoch * params.BeaconConfig().ShardSlotsPerEpoch,
		BlockSizePrice:        params.BeaconConfig().MinBlockSizePrices,
		OlderCommitteeRewards: make([]uint64, len(olderCommittee)),
		NewerCommitteeRewards: make([]uint64, len(newerCommittee)),
		OlderCommitteeFees:    make([]uint64, len(olderCommittee)),
		NewerCommitteeFees:    make([]uint64, len(newerCommittee)),
	}, nil
}

// GenesisShardBlock returns the genesis block of a particular shard.
//
// Spec pseudocode definition:
//  def get_genesis_shard_block(state: BeaconState, shard: Shard) -> ShardBlock:
//    return ShardBlock(data=ShardBlockData(
//        shard=shard,
//        slot=ShardSlot(SHARD_GENESIS_EPOCH * SHARD_SLOTS_PER_EPOCH),
//        state_root=hash_tree_root(get_genesis_shard_state(state, shard)),
//    ))
func GenesisShardBlock(state *pb.BeaconState, shard uint64) (*ethpb.ShardBlock, error) {
	genesisState, err := GenesisShardState(state, shard)
	if err != nil {
		return nil, errors.Wrap(err, "could not get shard genesis state")
	}
	stateRoot, err := ssz.TreeHash(genesisState)
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash shard genesis state")
	}
	return &ethpb.ShardBlock{
		Data: &ethpb.ShardBlockData{
			Slot:      params.BeaconConfig().ShardGenesisEpoch * params.BeaconConfig().ShardSlotsPerEpoch,
			StateRoot: stateRoot[:],
		},
	}, nil
}
