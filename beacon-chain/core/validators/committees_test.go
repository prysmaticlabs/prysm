package validators

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGetShardAndCommitteesForSlots(t *testing.T) {
	state := &pb.BeaconState{
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

func TestExceedingMaxValidatorRegistryFails(t *testing.T) {
	// Create more validators than ModuloBias defined in config, this should fail.
	size := 1<<(params.BeaconConfig().RandBytes*8) - 1

	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{Status: pb.ValidatorRecord_ACTIVE}
	for i := 0; i < size; i++ {
		validators[i] = validator
	}

	// ValidatorRegistryBySlotShard should fail the same.
	if _, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1); err == nil {
		t.Errorf("ValidatorRegistryBySlotShard should have failed")
	}
}

func BenchmarkMaxValidatorRegistry(b *testing.B) {
	var validators []*pb.ValidatorRecord
	validator := &pb.ValidatorRecord{}
	size := 1<<(params.BeaconConfig().RandBytes*8) - 1

	for i := 0; i < size; i++ {
		validators = append(validators, validator)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1)
	}
}

func TestInitialShardAndCommiteeForSlots(t *testing.T) {
	// Create 1000 validators in ActiveValidatorRegistry.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{}
		validators = append(validators, validator)
	}
	shardAndCommitteeArray, err := InitialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("unable to get initial shard committees %v", err)
	}

	if uint64(len(shardAndCommitteeArray)) != 3*params.BeaconConfig().CycleLength {
		t.Errorf("shard committee slots are not as expected: %d instead of %d", len(shardAndCommitteeArray), 3*params.BeaconConfig().CycleLength)
	}

}
func TestShuffleActiveValidatorRegistry(t *testing.T) {
	// Create 1000 validators in ActiveValidatorRegistry.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(params.BeaconConfig().CycleLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %v", params.BeaconConfig().CycleLength, len(indices))
	}
}

func TestSmallSampleValidatorRegistry(t *testing.T) {
	// Create a small number of validators validators in ActiveValidatorRegistry.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 20; i++ {
		validator := &pb.ValidatorRecord{}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(params.BeaconConfig().CycleLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %d", params.BeaconConfig().CycleLength, len(indices))
	}
}

func TestGetCommitteesPerSlotSmallValidatorSet(t *testing.T) {
	numValidatorRegistry := params.BeaconConfig().CycleLength * params.BeaconConfig().TargetCommitteeSize / 4

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotRegularValidatorSet(t *testing.T) {
	numValidatorRegistry := params.BeaconConfig().CycleLength * params.BeaconConfig().TargetCommitteeSize

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotLargeValidatorSet(t *testing.T) {
	numValidatorRegistry := params.BeaconConfig().CycleLength * params.BeaconConfig().TargetCommitteeSize * 8

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 5 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 5, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotSmallShardCount(t *testing.T) {
	config := params.BeaconConfig()
	oldShardCount := config.ShardCount
	config.ShardCount = config.CycleLength - 1

	numValidatorRegistry := params.BeaconConfig().CycleLength * params.BeaconConfig().TargetCommitteeSize

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}

	config.ShardCount = oldShardCount
}

func TestValidatorRegistryBySlotShardRegularValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidatorRegistry := int(params.BeaconConfig().CycleLength * params.BeaconConfig().TargetCommitteeSize)
	for i := 0; i < numValidatorRegistry; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != int(params.BeaconConfig().CycleLength) {
		t.Fatalf("Expected length %d: got %d", params.BeaconConfig().CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", params.BeaconConfig().TargetCommitteeSize, 1)
		}

		committeeSize := len(shardAndCommittees[0].Committee)
		if committeeSize != int(params.BeaconConfig().TargetCommitteeSize) {
			t.Fatalf("Expected committee size %d: got %d", params.BeaconConfig().TargetCommitteeSize, committeeSize)
		}
	}
}

func TestValidatorRegistryBySlotShardLargeValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidatorRegistry := int(params.BeaconConfig().CycleLength*params.BeaconConfig().TargetCommitteeSize) * 2
	for i := 0; i < numValidatorRegistry; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != int(params.BeaconConfig().CycleLength) {
		t.Fatalf("Expected length %d: got %d", params.BeaconConfig().CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 2 {
			t.Fatalf("Expected %d committee per slot: got %d", params.BeaconConfig().TargetCommitteeSize, 2)
		}

		t.Logf("slot %d", i)
		for j := 0; j < len(shardAndCommittees); j++ {
			shardCommittee := shardAndCommittees[j]
			t.Logf("shard %d", shardCommittee.Shard)
			t.Logf("committee: %v", shardCommittee.Committee)
			if len(shardCommittee.Committee) != int(params.BeaconConfig().TargetCommitteeSize) {
				t.Fatalf("Expected committee size %d: got %d", params.BeaconConfig().TargetCommitteeSize, len(shardCommittee.Committee))
			}
		}

	}
}

func TestValidatorRegistryBySlotShardSmallValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidatorRegistry := int(params.BeaconConfig().CycleLength*params.BeaconConfig().TargetCommitteeSize) / 2
	for i := 0; i < numValidatorRegistry; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	shardAndCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(shardAndCommitteeArray) != int(params.BeaconConfig().CycleLength) {
		t.Fatalf("Expected length %d: got %d", params.BeaconConfig().CycleLength, len(shardAndCommitteeArray))
	}

	for i := 0; i < len(shardAndCommitteeArray); i++ {
		shardAndCommittees := shardAndCommitteeArray[i].ArrayShardAndCommittee
		if len(shardAndCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", params.BeaconConfig().TargetCommitteeSize, 1)
		}

		for j := 0; j < len(shardAndCommittees); j++ {
			shardCommittee := shardAndCommittees[j]
			if len(shardCommittee.Committee) != int(params.BeaconConfig().TargetCommitteeSize/2) {
				t.Fatalf("Expected committee size %d: got %d", params.BeaconConfig().TargetCommitteeSize/2, len(shardCommittee.Committee))
			}
		}
	}
}

func TestAttestationParticipants_ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var committeeIndices []uint32
	for i := uint32(0); i < 8; i++ {
		committeeIndices = append(committeeIndices, i)
	}

	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: i},
				{Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}

	attestationData := &pb.AttestationData{}

	tests := []struct {
		attestationSlot uint64
		stateSlot       uint64
		shard           uint64
		bitfield        []byte
		wanted          []uint32
	}{
		{
			attestationSlot: 2,
			stateSlot:       5,
			shard:           0,
			bitfield:        []byte{'A'},
			wanted:          []uint32{1, 7},
		},
		{
			attestationSlot: 1,
			stateSlot:       10,
			shard:           0,
			bitfield:        []byte{1},
			wanted:          []uint32{7},
		},
		{
			attestationSlot: 10,
			stateSlot:       20,
			shard:           0,
			bitfield:        []byte{2},
			wanted:          []uint32{6},
		},
		{
			attestationSlot: 64,
			stateSlot:       100,
			shard:           0,
			bitfield:        []byte{3},
			wanted:          []uint32{6, 7},
		},
		{
			attestationSlot: 999,
			stateSlot:       1000,
			shard:           0,
			bitfield:        []byte{'F'},
			wanted:          []uint32{1, 5, 6},
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot
		attestationData.Slot = tt.attestationSlot
		attestationData.Shard = tt.shard

		result, err := AttestationParticipants(state, attestationData, tt.bitfield)
		if err != nil {
			t.Errorf("Failed to get attestation participants: %v", err)
		}

		if !reflect.DeepEqual(tt.wanted, result) {
			t.Errorf(
				"Result indices was an unexpected value. Wanted %d, got %d",
				tt.wanted,
				result,
			)
		}
	}
}

func TestAttestationParticipants_IncorrectBitfield(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: i},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}

	attestationData := &pb.AttestationData{}

	if _, err := AttestationParticipants(state, attestationData, []byte{1}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}
