package transition

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessShardPeriod processes a period of a shard.
//
// Spec pseudocode definition:
//  def process_shard_period(shard_state: ShardState) -> None:
//    # Rotate committee deltas
//    shard_state.older_committee_positive_deltas = shard_state.newer_committee_positive_deltas
//    shard_state.older_committee_negative_deltas = shard_state.newer_committee_negative_deltas
//    shard_state.newer_committee_positive_deltas = [Gwei(0) for _ in range(MAX_PERIOD_COMMITTEE_SIZE)]
//    shard_state.newer_committee_negative_deltas = [Gwei(0) for _ in range(MAX_PERIOD_COMMITTEE_SIZE)]
func ProcessShardPeriod(shardState *ethpb.ShardState) *ethpb.ShardState {
	shardState.OlderCommitteePositiveDeltas = shardState.NewerCommitteePositiveDeltas
	shardState.OlderCommitteeNegativeDeltas = shardState.NewerCommitteeNegativeDeltas
	shardState.NewerCommitteePositiveDeltas = make([]uint64, params.ShardConfig().MaxPeriodCommitteeSize)
	shardState.NewerCommitteeNegativeDeltas = make([]uint64, params.ShardConfig().MaxPeriodCommitteeSize)

	return shardState
}
