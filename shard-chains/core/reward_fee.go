package core

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// AddReward adds reward to input validator index.
//
// Spec pseudocode definition:
//  def add_reward(state: BeaconState, shard_state: ShardState, index: ValidatorIndex, delta: Gwei) -> None:
//    epoch = compute_epoch_of_shard_slot(state.slot)
//    older_committee = get_period_committee(state, shard_state.shard, compute_shard_period_start_epoch(epoch, 2))
//    newer_committee = get_period_committee(state, shard_state.shard, compute_shard_period_start_epoch(epoch, 1))
//    if index in older_committee:
//        shard_state.older_committee_rewards[older_committee.index(index)] += delta
//    elif index in newer_committee:
//        shard_state.newer_committee_rewards[newer_committee.index(index)] += delta
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

// AddFee adds fee to input validator index.
//
// Spec pseudocode definition:
//  def add_fee(state: BeaconState, shard_state: ShardState, index: ValidatorIndex, delta: Gwei) -> None:
//	    epoch = compute_epoch_of_shard_slot(state.slot)
//	    older_committee = get_period_committee(state, shard_state.shard, compute_shard_period_start_epoch(epoch, 2))
//	    newer_committee = get_period_committee(state, shard_state.shard, compute_shard_period_start_epoch(epoch, 1))
//	    if index in older_committee:
//	        shard_state.older_committee_fees[older_committee.index(index)] += delta
//	    elif index in newer_committee:
//	        shard_state.newer_committee_fees[newer_committee.index(index)] += delta
func AddFee(beaconState *pb.BeaconState, shardState *ethpb.ShardState, index uint64, delta uint64) (*ethpb.ShardState, error) {
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
			shardState.OlderCommitteeFees[i] += delta
			return shardState, nil
		}
	}

	for _, i := range newerCommittee {
		if i == index {
			shardState.NewerCommitteeFees[i] += delta
			return shardState, nil
		}
	}

	return shardState, nil
}
