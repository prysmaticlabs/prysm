package casper

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGetShardAndCommitteesForSlots(t *testing.T) {
	state := &pb.CrystallizedState{
		LastStateRecalc: 65,
		ShardAndCommitteesForSlots: []*pb.ShardAndCommitteeArray{
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{ShardId: 1, Committee: []uint32{0, 1, 2, 3, 4}},
				{ShardId: 2, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{ShardId: 3, Committee: []uint32{0, 1, 2, 3, 4}},
				{ShardId: 4, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
		}}
	if _, err := GetShardAndCommitteesForSlot(state.ShardAndCommitteesForSlots, state.LastStateRecalc, 1000); err == nil {
		t.Error("getShardAndCommitteesForSlot should have failed with invalid slot")
	}
	committee, err := GetShardAndCommitteesForSlot(state.ShardAndCommitteesForSlots, state.LastStateRecalc, 1)
	if err != nil {
		t.Errorf("getShardAndCommitteesForSlot failed: %v", err)
	}
	if committee.ArrayShardAndCommittee[0].ShardId != 1 {
		t.Errorf("getShardAndCommitteesForSlot returns shardID should be 1, got: %v", committee.ArrayShardAndCommittee[0].ShardId)
	}
	committee, _ = GetShardAndCommitteesForSlot(state.ShardAndCommitteesForSlots, state.LastStateRecalc, 2)
	if committee.ArrayShardAndCommittee[0].ShardId != 3 {
		t.Errorf("getShardAndCommitteesForSlot returns shardID should be 3, got: %v", committee.ArrayShardAndCommittee[0].ShardId)
	}
}

func TestMaxValidators(t *testing.T) {
	// Create more validators than params.MaxValidators, this should fail.
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.MaxValidators+1; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 1, EndDynasty: 100}
		validators = append(validators, validator)
	}

	if _, _, err := SampleAttestersAndProposers(common.Hash{'A'}, validators, 1); err == nil {
		t.Errorf("GetAttestersProposer should have failed")
	}

	// ValidatorsBySlotShard should fail the same.
	if _, err := ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1, 0); err == nil {
		t.Errorf("ValidatorsBySlotShard should have failed")
	}
}

func Test1000ActiveValidators(t *testing.T) {
	// Create 1000 validators in ActiveValidators.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 1, EndDynasty: 100}
		validators = append(validators, validator)
	}

	attesters, proposer, err := SampleAttestersAndProposers(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("GetAttestersProposer function failed: %v", err)
	}

	activeValidators := ActiveValidatorIndices(validators, 1)

	validatorList, err := utils.ShuffleIndices(common.Hash{'A'}, activeValidators)
	if err != nil {
		t.Errorf("Shuffle function function failed: %v", err)
	}

	if !reflect.DeepEqual(proposer, validatorList[len(validatorList)-1]) {
		t.Errorf("Get proposer failed, expected: %v got: %v", validatorList[len(validatorList)-1], proposer)
	}
	if !reflect.DeepEqual(attesters, validatorList[:len(attesters)]) {
		t.Errorf("Get attesters failed, expected: %v got: %v", validatorList[:len(attesters)], attesters)
	}

	indices, err := ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1, 0)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != params.CycleLength {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %v", params.CycleLength, len(indices))
	}
}

func TestSmallSampleValidators(t *testing.T) {
	// Create a small number of validators validators in ActiveValidators.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 20; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 1, EndDynasty: 100}
		validators = append(validators, validator)
	}

	attesters, proposer, err := SampleAttestersAndProposers(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("GetAttestersProposer function failed: %v", err)
	}

	activeValidators := ActiveValidatorIndices(validators, 1)

	validatorList, err := utils.ShuffleIndices(common.Hash{'A'}, activeValidators)
	if err != nil {
		t.Errorf("Shuffle function function failed: %v", err)
	}

	if !reflect.DeepEqual(proposer, validatorList[len(validatorList)-1]) {
		t.Errorf("Get proposer failed, expected: %v got: %v", validatorList[len(validatorList)-1], proposer)
	}
	if !reflect.DeepEqual(attesters, validatorList[:len(attesters)]) {
		t.Errorf("Get attesters failed, expected: %v got: %v", validatorList[:len(attesters)], attesters)
	}

	indices, err := ShuffleValidatorsToCommittees(common.Hash{'A'}, validators, 1, 0)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != params.CycleLength {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %d", params.CycleLength, len(indices))
	}
}

func TestGetCommitteeParamsSmallValidatorSet(t *testing.T) {
	numValidators := params.CycleLength * params.MinCommiteeSize / 4

	committesPerSlot, slotsPerCommittee := getCommitteeParams(numValidators)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}

	if slotsPerCommittee != 4 {
		t.Fatalf("Expected slotsPerCommittee to equal %d: got %d", 4, slotsPerCommittee)
	}
}

func TestGetCommitteeParamsRegularValidatorSet(t *testing.T) {
	numValidators := params.CycleLength * params.MinCommiteeSize

	committesPerSlot, slotsPerCommittee := getCommitteeParams(numValidators)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}

	if slotsPerCommittee != 1 {
		t.Fatalf("Expected slotsPerCommittee to equal %d: got %d", 1, slotsPerCommittee)
	}
}

func TestGetCommitteeParamsLargeValidatorSet(t *testing.T) {
	numValidators := params.CycleLength * params.MinCommiteeSize * 8

	committesPerSlot, slotsPerCommittee := getCommitteeParams(numValidators)
	if committesPerSlot != 5 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 5, committesPerSlot)
	}

	if slotsPerCommittee != 1 {
		t.Fatalf("Expected slotsPerCommittee to equal %d: got %d", 1, slotsPerCommittee)
	}
}

func TestValidatorsBySlotShardRegularValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidators := params.CycleLength * params.MinCommiteeSize
	for i := 0; i < numValidators; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != params.CycleLength {
		t.Fatalf("Expected length %d: got %d", params.CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", params.MinCommiteeSize, 1)
		}

		committeeSize := len(shardAndCommittees[0].Committee)
		if committeeSize != params.MinCommiteeSize {
			t.Fatalf("Expected committee size %d: got %d", params.MinCommiteeSize, committeeSize)
		}
	}
}

func TestValidatorsBySlotShardLargeValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidators := params.CycleLength * params.MinCommiteeSize * 2
	for i := 0; i < numValidators; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != params.CycleLength {
		t.Fatalf("Expected length %d: got %d", params.CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 2 {
			t.Fatalf("Expected %d committee per slot: got %d", params.MinCommiteeSize, 2)
		}

		t.Logf("slot %d", i)
		for j := 0; j < len(shardAndCommittees); j++ {
			shardCommittee := shardAndCommittees[j]
			t.Logf("shard %d", shardCommittee.ShardId)
			t.Logf("committee: %v", shardCommittee.Committee)
			if len(shardCommittee.Committee) != params.MinCommiteeSize {
				t.Fatalf("Expected committee size %d: got %d", params.MinCommiteeSize, len(shardCommittee.Committee))
			}
		}

	}
}

func TestValidatorsBySlotShardSmallValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidators := params.CycleLength * params.MinCommiteeSize / 2
	for i := 0; i < numValidators; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != params.CycleLength {
		t.Fatalf("Expected length %d: got %d", params.CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", params.MinCommiteeSize, 1)
		}

		for j := 0; j < len(shardAndCommittees); j++ {
			shardCommittee := shardAndCommittees[j]
			if len(shardCommittee.Committee) != params.MinCommiteeSize/2 {
				t.Fatalf("Expected committee size %d: got %d", params.MinCommiteeSize/2, len(shardCommittee.Committee))
			}
		}
	}
}
