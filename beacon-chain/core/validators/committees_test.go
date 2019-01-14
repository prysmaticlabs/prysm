package validators

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGetShardCommitteesAtSlots(t *testing.T) {
	validators := make([]*pb.ValidatorRecord, config.EpochLength)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}
	if _, err := ShardCommitteesAtSlot(state, 1000); err == nil {
		t.Error("getShardCommitteesForSlot should have failed with invalid slot")
	}
	committee, err := ShardCommitteesAtSlot(state, 1)
	if err != nil {
		t.Errorf("getShardCommitteesForSlot failed: %v", err)
	}

	if committee[0].Shard != 1 {
		t.Errorf("getShardCommitteesForSlot returns Shard should be 1, got: %v", committee[0].Shard)
	}
	committee, _ = ShardCommitteesAtSlot(state, 3)
	if committee[0].Shard != 3 {
		t.Errorf("getShardCommitteesForSlot returns Shard should be 3, got: %v", committee[0].Shard)
	}
}

func TestExceedingMaxValidatorRegistryFails(t *testing.T) {
	// Create more validators than ModuloBias defined in config, this should fail.
	size := 1<<(config.RandBytes*8) - 1

	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{ExitSlot: config.FarFutureSlot}
	for i := 0; i < size; i++ {
		validators[i] = validator
	}

	// ValidatorRegistryBySlotShard should fail the same.
	if _, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 0); err == nil {
		t.Errorf("ValidatorRegistryBySlotShard should have failed")
	}
}

func BenchmarkMaxValidatorRegistry(b *testing.B) {
	var validators []*pb.ValidatorRecord
	validator := &pb.ValidatorRecord{}
	size := 1<<(config.RandBytes*8) - 1

	for i := 0; i < size; i++ {
		validators = append(validators, validator)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 0)
	}
}

func TestShuffleActiveValidatorRegistry(t *testing.T) {
	// Create 1000 validators in ActiveValidatorRegistry.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 0)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(config.EpochLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %v", config.EpochLength, len(indices))
	}
}

func TestSmallSampleValidatorRegistry(t *testing.T) {
	// Create a small number of validators validators in ActiveValidatorRegistry.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 20; i++ {
		validator := &pb.ValidatorRecord{}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 0)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(config.EpochLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %d", config.EpochLength, len(indices))
	}
}

func TestGetCommitteesPerSlotSmallValidatorSet(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize / 4

	committesPerSlot := committeesCountPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotRegularValidatorSet(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize

	committesPerSlot := committeesCountPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotLargeValidatorSet(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize * 8

	committesPerSlot := committeesCountPerSlot(numValidatorRegistry)
	if committesPerSlot != 8 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 8, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotBoundOnShardCount(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize * 16

	committesPerSlot := committeesCountPerSlot(numValidatorRegistry)
	if committesPerSlot != config.ShardCount/config.EpochLength {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 8, committesPerSlot)
	}
}

func TestValidatorRegistryBySlotShardLargeValidatorSet(t *testing.T) {

	size := config.EpochLength * config.TargetCommitteeSize
	validators := make([]*pb.ValidatorRecord, size)
	for i := uint64(0); i < size; i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}
	committees, err := ShuffleValidatorRegistryToCommittees(
		[32]byte{}, validators, 0)
	if err != nil {
		t.Errorf("could not execute ShuffleValidatorRegistryToCommittees: %v", err)
	}

	if len(committees) != int(config.EpochLength) {
		t.Fatalf("Expected length %d: got %d", config.EpochLength, len(committees))
	}

	for i := 0; i < len(committees); i++ {
		committee := committees[i]
		if len(committee) != int(config.TargetCommitteeSize) {
			t.Fatalf("Expected %d committee per slot: got %d",
				config.TargetCommitteeSize, len(committee))
		}
	}
}

func TestValidatorRegistryBySlotShardSmallValidatorSet(t *testing.T) {
	validators := make([]*pb.ValidatorRecord, config.EpochLength*config.TargetCommitteeSize)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	for i := uint64(0); i < config.EpochLength; i++ {
		committees, err := ShardCommitteesAtSlot(state, i)
		if err != nil {
			t.Errorf("getShardCommitteesForSlot failed: %v", err)
		}

		if len(committees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", 1,
				len(committees))
		}

		if len(committees[0].Committee) != int(config.TargetCommitteeSize) {
			t.Fatalf("Expected committee size %d: got %d",
				config.TargetCommitteeSize, len(committees[0].Committee))
		}

		if committees[0].Shard != i {
			t.Fatalf("Expected shard number %d: got %d",
				i, committees[0].Shard)
		}
	}
}

func TestAttestationParticipants_ok(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.ValidatorRecord, config.EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
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
			shard:           2,
			bitfield:        []byte{0xFF},
			wanted:          []uint32{11, 121},
		},
		{
			attestationSlot: 1,
			stateSlot:       10,
			shard:           1,
			bitfield:        []byte{77},
			wanted:          []uint32{117},
		},
		{
			attestationSlot: 10,
			stateSlot:       20,
			shard:           10,
			bitfield:        []byte{0xFF},
			wanted:          []uint32{14, 30},
		},
		{
			attestationSlot: 64,
			stateSlot:       100,
			shard:           0,
			bitfield:        []byte{0xFF},
			wanted:          []uint32{109, 97},
		},
		{
			attestationSlot: 999,
			stateSlot:       1000,
			shard:           39,
			bitfield:        []byte{99},
			wanted:          []uint32{89},
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
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: config.FarFutureSlot},
		},
	}
	attestationData := &pb.AttestationData{}

	if _, err := AttestationParticipants(state, attestationData, []byte{1}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}
