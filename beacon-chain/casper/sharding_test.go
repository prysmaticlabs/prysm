package casper

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGetIndicesForHeight(t *testing.T) {
	state := &pb.CrystallizedState{
		LastStateRecalc: 1,
		IndicesForHeights: []*pb.ShardAndCommitteeArray{
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{ShardId: 1, Committee: []uint32{0, 1, 2, 3, 4}},
				{ShardId: 2, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{ShardId: 3, Committee: []uint32{0, 1, 2, 3, 4}},
				{ShardId: 4, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
		}}
	if _, err := GetIndicesForHeight(state.IndicesForHeights, state.LastStateRecalc, 1000); err == nil {
		t.Error("getIndicesForHeight should have failed with invalid height")
	}
	committee, err := GetIndicesForHeight(state.IndicesForHeights, state.LastStateRecalc, 1)
	if err != nil {
		t.Errorf("getIndicesForHeight failed: %v", err)
	}
	if committee.ArrayShardAndCommittee[0].ShardId != 1 {
		t.Errorf("getIndicesForHeight returns shardID should be 1, got: %v", committee.ArrayShardAndCommittee[0].ShardId)
	}
	committee, _ = GetIndicesForHeight(state.IndicesForHeights, state.LastStateRecalc, 2)
	if committee.ArrayShardAndCommittee[0].ShardId != 3 {
		t.Errorf("getIndicesForHeight returns shardID should be 3, got: %v", committee.ArrayShardAndCommittee[0].ShardId)
	}
}

func TestSampleAttestersAndProposers(t *testing.T) {
	// Create validators more than params.MaxValidators, this should fail.
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.MaxValidators+1; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 1, EndDynasty: 100}
		validators = append(validators, validator)
	}

	if _, _, err := SampleAttestersAndProposers(common.Hash{'A'}, validators, 1); err == nil {
		t.Errorf("GetAttestersProposer should have failed")
	}

	// ValidatorsByHeightShard should fail the same.
	if _, err := ValidatorsByHeightShard(common.Hash{'A'}, validators, 1, 0); err == nil {
		t.Errorf("ValidatorsByHeightShard should have failed")
	}

	// Create 1000 validators in ActiveValidators.
	validators = validators[:0]
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

	indices, err := ValidatorsByHeightShard(common.Hash{'A'}, validators, 1, 0)
	if err != nil {
		t.Errorf("validatorsByHeightShard failed with %v:", err)
	}
	if len(indices) != 8192 {
		t.Errorf("incorret length for validator indices. Want: 8192. Got: %v", len(indices))
	}

	// Create a small number of validators validators in ActiveValidators.
	validators = validators[:0]
	for i := 0; i < 20; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 1, EndDynasty: 100}
		validators = append(validators, validator)
	}

	attesters, proposer, err = SampleAttestersAndProposers(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("GetAttestersProposer function failed: %v", err)
	}

	activeValidators = ActiveValidatorIndices(validators, 1)

	validatorList, err = utils.ShuffleIndices(common.Hash{'A'}, activeValidators)
	if err != nil {
		t.Errorf("Shuffle function function failed: %v", err)
	}

	if !reflect.DeepEqual(proposer, validatorList[len(validatorList)-1]) {
		t.Errorf("Get proposer failed, expected: %v got: %v", validatorList[len(validatorList)-1], proposer)
	}
	if !reflect.DeepEqual(attesters, validatorList[:len(attesters)]) {
		t.Errorf("Get attesters failed, expected: %v got: %v", validatorList[:len(attesters)], attesters)
	}

	indices, err = ValidatorsByHeightShard(common.Hash{'A'}, validators, 1, 0)
	if err != nil {
		t.Errorf("validatorsByHeightShard failed with %v:", err)
	}
	if len(indices) != 8192 {
		t.Errorf("incorret length for validator indices. Want: 8192. Got: %v", len(indices))
	}
}
