package helpers

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ProcessDelta adds fee to input validator index.
//
// Spec pseudocode definition:
//  def process_delta(beacon_state: BeaconState,
//                  shard_state: ShardState,
//                  index: ValidatorIndex,
//                  delta: Gwei,
//                  positive: bool=True) -> None:
//    epoch = compute_epoch_of_shard_slot(beacon_state.slot)
//    older_committee = get_period_committee(beacon_state, shard_state.shard, compute_shard_period_start_epoch(epoch, 2))
//    newer_committee = get_period_committee(beacon_state, shard_state.shard, compute_shard_period_start_epoch(epoch, 1))
//    if index in older_committee:
//        if positive:
//            shard_state.older_committee_positive_deltas[older_committee.index(index)] += delta
//        else:
//            shard_state.older_committee_negative_deltas[older_committee.index(index)] += delta
//    elif index in newer_committee:
//        if positive:
//            shard_state.newer_committee_positive_deltas[newer_committee.index(index)] += delta
//        else:
//            shard_state.newer_committee_negative_deltas[newer_committee.index(index)] += delta
func ProcessDelta(
	beaconState *pb.BeaconState,
	shardState *ethpb.ShardState,
	index uint64,
	delta uint64,
	positive bool,
) (*ethpb.ShardState, error) {
	epoch := ComputeEpochOfShardSlot(beaconState.Slot)
	olderEpoch := ComputeShardPeriodStartEpoch(epoch, 2)
	newerEpoch := ComputeShardPeriodStartEpoch(epoch, 1)
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
			if positive {
				shardState.OlderCommitteePositiveDeltas[olderCommittee[index]] += delta
			} else {
				shardState.OlderCommitteeNegativeDeltas[olderCommittee[index]] += delta
			}
			return shardState, nil
		}
	}

	for _, i := range newerCommittee {
		if i == index {
			if positive {
				shardState.NewerCommitteePositiveDeltas[newerCommittee[index]] += delta
			} else {
				shardState.NewerCommitteeNegativeDeltas[newerCommittee[index]] += delta
			}
			return shardState, nil
		}
	}

	return shardState, nil
}
