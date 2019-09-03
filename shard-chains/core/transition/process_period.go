package transition

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shard-chains/core/helpers"
)

// ProcessShardPeriod processes a period of a shard.
//
// Spec pseudocode definition:
//  def process_shard_period(shard_state: ShardState, state: BeaconState) -> None:
//    # Rotate rewards and fees
//    epoch = compute_epoch_of_shard_slot(state.slot)
//    newer_committee = get_period_committee(state, state.shard, compute_shard_period_start_epoch(epoch, 1))
//    state.older_committee_rewards = state.newer_committee_rewards
//    state.newer_committee_rewards = [Gwei(0) for _ in range(len(newer_committee))]
//    state.older_committee_fees = state.newer_committee_fees
//    state.newer_committee_fees = [Gwei(0) for _ in range(len(newer_committee))]
func ProcessShardPeriod(beaconState *pb.BeaconState, shardState *ethpb.ShardState) (*ethpb.ShardState, error) {

	epoch := helpers.ShardSlotToEpoch(shardState.Slot)

	newerCommittee, err := helpers.PeriodCommittee(beaconState, shardState.Shard, epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get period committee for epoch %d", epoch)
	}
	shardState.OlderCommitteeFees = shardState.NewerCommitteeFees
	shardState.NewerCommitteeFees = make([]uint64, len(newerCommittee))
	shardState.OlderCommitteeRewards = shardState.NewerCommitteeRewards
	shardState.NewerCommitteeRewards = make([]uint64, len(newerCommittee))

	return shardState, err
}
