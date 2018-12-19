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

func TestNewValidatorSetRecalculationsInvalid(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	// Negative test case, shuffle validators with more than MaxValidatorRegistry.
	size := 1<<(params.BeaconConfig().RandBytes*8) - 1
	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{Status: pb.ValidatorRecord_ACTIVE}
	for i := 0; i < size; i++ {
		validators[i] = validator
	}
	beaconState.SetValidatorRegistry(validators)
	if _, err := validatorSetRecalculations(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.ValidatorRegistry(),
		[32]byte{'A'},
	); err == nil {
		t.Error("Validator set change calculation should have failed with invalid validator count")
	}
}

func TestNewValidatorSetRecalculations(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	// Create shard committee for every slot.
	var shardCommitteesForSlot []*pb.ShardAndCommitteeArray
	for i := 0; i < int(params.BeaconConfig().CycleLength); i++ {
		// Only 10 shards gets crosslinked by validators this period.
		var shardCommittees []*pb.ShardAndCommittee
		for i := 0; i < 10; i++ {
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{Shard: uint64(i)})
		}
		shardCommitteesForSlot = append(shardCommitteesForSlot, &pb.ShardAndCommitteeArray{ArrayShardAndCommittee: shardCommittees})
	}

	beaconState.SetShardAndCommitteesForSlots(shardCommitteesForSlot)
	beaconState.SetLastStateRecalculationSlot(65)

	_, err = validatorSetRecalculations(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.ValidatorRegistry(),
		[32]byte{'A'},
	)
	if err != nil {
		t.Fatalf("Validator set change failed %v", err)
	}
}
