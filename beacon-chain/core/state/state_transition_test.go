package state

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessLatestCrosslinks(t *testing.T) {
	// Set up crosslink record for every shard.
	var clRecords []*pb.CrosslinkRecord
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		clRecord := &pb.CrosslinkRecord{ShardBlockRootHash32: []byte{'A'}, Slot: 1}
		clRecords = append(clRecords, clRecord)
	}

	// Set up validators.
	var validators []*pb.ValidatorRecord

	for i := 0; i < 20; i++ {
		validators = append(validators, &pb.ValidatorRecord{
			Balance: 1e18,
			Status:  pb.ValidatorRecord_ACTIVE,
		})
	}

	// Set up pending attestations.
	pAttestations := []*pb.AggregatedAttestation{
		{
			Slot:             0,
			Shard:            1,
			ShardBlockHash:   []byte{'a'},
			AttesterBitfield: []byte{224},
		},
	}

	// Process crosslinks happened at slot 50.
	shardAndCommitteesForSlots, err := v.InitialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("failed to initialize indices for slots: %v", err)
	}

	committee := []uint32{0, 4, 6}

	shardAndCommitteesForSlots[0].ArrayShardAndCommittee[0].Committee = committee

	beaconState := types.NewBeaconState(&pb.BeaconState{
		LatestCrosslinks:           clRecords,
		ValidatorRegistry:          validators,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
	})
	newLatestCrosslinks, err := crossLinkCalculations(beaconState, pAttestations, 100)
	if err != nil {
		t.Fatalf("process crosslink failed %v", err)
	}

	if newLatestCrosslinks[1].Slot != params.BeaconConfig().CycleLength {
		t.Errorf("Slot did not change for new cross link. Wanted: %d. Got: %d", params.BeaconConfig().CycleLength, newLatestCrosslinks[0].Slot)
	}
	if !bytes.Equal(newLatestCrosslinks[1].ShardBlockRootHash32, []byte{'a'}) {
		t.Errorf("ShardBlockHash did not change for new cross link. Wanted a. Got: %s", newLatestCrosslinks[0].ShardBlockRootHash32)
	}
	//TODO(#538) Implement tests on balances of the validators in committee once big.Int is introduced.
}

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

func TestPenalizedETH(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	beaconState.SetLatestPenalizedExitBalances([]uint64{100, 200, 300, 400, 500})
	beaconState.PenalizedETH(2)

	tests := []struct {
		a uint64
		b uint64
	}{
		{a: 0, b: 100},
		{a: 1, b: 300},
		{a: 2, b: 600},
		{a: 3, b: 900},
		{a: 4, b: 1200},
	}
	for _, tt := range tests {
		if beaconState.PenalizedETH(tt.a) != tt.b {
			t.Errorf("PenalizedETH(%d) = %v, want = %d", tt.a, beaconState.PenalizedETH(tt.a), tt.b)
		}
	}
}
