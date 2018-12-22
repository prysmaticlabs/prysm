package state

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestIsNewValidatorSetTransition(t *testing.T) {
	beaconState, err := NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	beaconState.ValidatorRegistryLastChangeSlot = 1
	if IsValidatorSetChange(beaconState, 0) {
		t.Errorf("Is new validator set change should be false, last changed slot greater than finalized slot")
	}
	beaconState.FinalizedSlot = 2
	if IsValidatorSetChange(beaconState, 2) {
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
	beaconState.ShardAndCommitteesAtSlots = shardCommitteeForSlots

	crosslinks := []*pb.CrosslinkRecord{
		{Slot: 1},
		{Slot: 1},
		{Slot: 1},
	}
	beaconState.LatestCrosslinks = crosslinks

	if IsValidatorSetChange(beaconState, params.BeaconConfig().MinValidatorSetChangeInterval+1) {
		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
	}

	crosslinks = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	beaconState.LatestCrosslinks = crosslinks

	if !IsValidatorSetChange(beaconState, params.BeaconConfig().MinValidatorSetChangeInterval+1) {
		t.Errorf("New validator set change failed should have been true")
	}
}
