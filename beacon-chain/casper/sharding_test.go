package casper

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGetShardAndCommitteesForSlots(t *testing.T) {
	state := &pb.CrystallizedState{
		LastStateRecalculationSlot: 64,
		ShardAndCommitteesForSlots: []*pb.ShardAndCommitteeArray{
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4}},
				{Shard: 2, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 3, Committee: []uint32{0, 1, 2, 3, 4}},
				{Shard: 4, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
		}}
	if _, err := GetShardAndCommitteesForSlot(state.ShardAndCommitteesForSlots, state.LastStateRecalculationSlot, 1000); err == nil {
		t.Error("getShardAndCommitteesForSlot should have failed with invalid slot")
	}
	committee, err := GetShardAndCommitteesForSlot(state.ShardAndCommitteesForSlots, state.LastStateRecalculationSlot, 0)
	if err != nil {
		t.Errorf("getShardAndCommitteesForSlot failed: %v", err)
	}
	if committee.ArrayShardAndCommittee[0].Shard != 1 {
		t.Errorf("getShardAndCommitteesForSlot returns Shard should be 1, got: %v", committee.ArrayShardAndCommittee[0].Shard)
	}
	committee, _ = GetShardAndCommitteesForSlot(state.ShardAndCommitteesForSlots, state.LastStateRecalculationSlot, 1)
	if committee.ArrayShardAndCommittee[0].Shard != 3 {
		t.Errorf("getShardAndCommitteesForSlot returns Shard should be 3, got: %v", committee.ArrayShardAndCommittee[0].Shard)
	}
}

func TestExceedingMaxValidatorsFails(t *testing.T) {
	// Create more validators than ModuloBias defined in config, this should fail.
	size := params.GetConfig().ModuloBias + 1
	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{WithdrawalShard: 0, Status: uint64(params.Active)}
	for i := uint64(0); i < size; i++ {
		validators[i] = validator
	}

	// ValidatorsBySlotShard should fail the same.
	if _, err := ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1); err == nil {
		t.Errorf("ValidatorsBySlotShard should have failed")
	}
}

func BenchmarkMaxValidators(b *testing.B) {
	var validators []*pb.ValidatorRecord
	validator := &pb.ValidatorRecord{WithdrawalShard: 0}
	for i := uint64(0); i < params.GetConfig().ModuloBias; i++ {
		validators = append(validators, validator)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1)
	}
}

func TestInitialShardAndCommiteeForSlots(t *testing.T) {
	// Create 1000 validators in ActiveValidators.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{WithdrawalShard: 0}
		validators = append(validators, validator)
	}
	shardAndCommitteeArray, err := InitialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("unable to get initial shard committees %v", err)
	}

	if uint64(len(shardAndCommitteeArray)) != 3*params.GetConfig().CycleLength {
		t.Errorf("shard committee slots are not as expected: %d instead of %d", len(shardAndCommitteeArray), 2*params.GetConfig().CycleLength)
	}

}
func TestShuffleActiveValidators(t *testing.T) {
	// Create 1000 validators in ActiveValidators.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{WithdrawalShard: 0}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(params.GetConfig().CycleLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %v", params.GetConfig().CycleLength, len(indices))
	}
}

func TestSmallSampleValidators(t *testing.T) {
	// Create a small number of validators validators in ActiveValidators.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 20; i++ {
		validator := &pb.ValidatorRecord{WithdrawalShard: 0}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(params.GetConfig().CycleLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %d", params.GetConfig().CycleLength, len(indices))
	}
}

func TestGetCommitteesPerSlotSmallValidatorSet(t *testing.T) {
	numValidators := params.GetConfig().CycleLength * params.GetConfig().MinCommiteeSize / 4

	committesPerSlot := getCommitteesPerSlot(numValidators)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotRegularValidatorSet(t *testing.T) {
	numValidators := params.GetConfig().CycleLength * params.GetConfig().MinCommiteeSize

	committesPerSlot := getCommitteesPerSlot(numValidators)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotLargeValidatorSet(t *testing.T) {
	numValidators := params.GetConfig().CycleLength * params.GetConfig().MinCommiteeSize * 8

	committesPerSlot := getCommitteesPerSlot(numValidators)
	if committesPerSlot != 5 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 5, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotSmallShardCount(t *testing.T) {
	config := params.GetConfig()
	oldShardCount := config.ShardCount
	config.ShardCount = config.CycleLength - 1

	numValidators := params.GetConfig().CycleLength * params.GetConfig().MinCommiteeSize

	committesPerSlot := getCommitteesPerSlot(numValidators)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}

	config.ShardCount = oldShardCount
}

func TestValidatorsBySlotShardRegularValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidators := int(params.GetConfig().CycleLength * params.GetConfig().MinCommiteeSize)
	for i := 0; i < numValidators; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != int(params.GetConfig().CycleLength) {
		t.Fatalf("Expected length %d: got %d", params.GetConfig().CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", params.GetConfig().MinCommiteeSize, 1)
		}

		committeeSize := len(shardAndCommittees[0].Committee)
		if committeeSize != int(params.GetConfig().MinCommiteeSize) {
			t.Fatalf("Expected committee size %d: got %d", params.GetConfig().MinCommiteeSize, committeeSize)
		}
	}
}

func TestValidatorsBySlotShardLargeValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidators := int(params.GetConfig().CycleLength*params.GetConfig().MinCommiteeSize) * 2
	for i := 0; i < numValidators; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != int(params.GetConfig().CycleLength) {
		t.Fatalf("Expected length %d: got %d", params.GetConfig().CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 2 {
			t.Fatalf("Expected %d committee per slot: got %d", params.GetConfig().MinCommiteeSize, 2)
		}

		t.Logf("slot %d", i)
		for j := 0; j < len(shardAndCommittees); j++ {
			shardCommittee := shardAndCommittees[j]
			t.Logf("shard %d", shardCommittee.Shard)
			t.Logf("committee: %v", shardCommittee.Committee)
			if len(shardCommittee.Committee) != int(params.GetConfig().MinCommiteeSize) {
				t.Fatalf("Expected committee size %d: got %d", params.GetConfig().MinCommiteeSize, len(shardCommittee.Committee))
			}
		}

	}
}

func TestValidatorsBySlotShardSmallValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidators := int(params.GetConfig().CycleLength*params.GetConfig().MinCommiteeSize) / 2
	for i := 0; i < numValidators; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != int(params.GetConfig().CycleLength) {
		t.Fatalf("Expected length %d: got %d", params.GetConfig().CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", params.GetConfig().MinCommiteeSize, 1)
		}

		for j := 0; j < len(shardAndCommittees); j++ {
			shardCommittee := shardAndCommittees[j]
			if len(shardCommittee.Committee) != int(params.GetConfig().MinCommiteeSize/2) {
				t.Fatalf("Expected committee size %d: got %d", params.GetConfig().MinCommiteeSize/2, len(shardCommittee.Committee))
			}
		}
	}
}
