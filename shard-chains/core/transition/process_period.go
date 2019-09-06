package transition

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessShardPeriod processes a period of a shard.
//
// Spec pseudocode definition:
//  def process_shard_period(state: BeaconState, shard_state: ShardState) -> None:
//    # Rotate rewards and fees
//    state.older_committee_deltas = state.newer_committee_deltas
//    state.newer_committee_deltas = [GweiDelta(0) for _ in range(MAX_PERIOD_COMMITTEE_SIZE)]
func ProcessShardPeriod(shardState *ethpb.ShardState) *ethpb.ShardState {

	shardState.OlderCommitteeDeltas = shardState.NewerCommitteeDeltas
	shardState.NewerCommitteeDeltas = make([]uint64, params.ShardConfig().MaxPeriodCommitteeSize)

	return shardState
}
