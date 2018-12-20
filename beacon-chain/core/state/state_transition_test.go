package state

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestIsNewValidatorSetTransition(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	beaconState.SetValidatorRegistryLastChangeSlot(1)
	if beaconState.IsValidatorSetChange(0) {
		t.Errorf("Is new validator set change should be false, last changed slot greater than finalized slot")
	}
	beaconState.SetLastFinalizedSlot(2)
	if beaconState.IsValidatorSetChange(1) {
		t.Errorf("Is new validator set change should be false, MinValidatorSetChangeInterval has not reached")
	}
	shardCommitteeForSlots := []*pb.ShardAndCommitteeArray{{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0},
			{Shard: 1},
			{Shard: 2},
		},
	},
	}
	beaconState.SetShardAndCommitteesForSlots(shardCommitteeForSlots)

	crosslinks := []*pb.CrosslinkRecord{
		{Slot: 1},
		{Slot: 1},
		{Slot: 1},
	}
	beaconState.SetCrossLinks(crosslinks)

	if beaconState.IsValidatorSetChange(params.BeaconConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
	}

	crosslinks = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	beaconState.SetCrossLinks(crosslinks)

	if !beaconState.IsValidatorSetChange(params.BeaconConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("New validator set changen failed should have been true")
	}
}
