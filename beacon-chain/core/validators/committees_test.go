package validators

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGetShardCommitteesAtSlots(t *testing.T) {
	state := &pb.BeaconState{
		ShardCommitteesAtSlots: []*pb.ShardCommitteeArray{
			{ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4}},
				{Shard: 2, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
			{ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 3, Committee: []uint32{0, 1, 2, 3, 4}},
				{Shard: 4, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
		}}
	if _, err := ShardCommitteesAtSlot(state, 1000); err == nil {
		t.Error("getShardCommitteesForSlot should have failed with invalid slot")
	}
	committee, err := ShardCommitteesAtSlot(state, 0)
	if err != nil {
		t.Errorf("getShardCommitteesForSlot failed: %v", err)
	}
	if committee.ArrayShardCommittee[0].Shard != 1 {
		t.Errorf("getShardCommitteesForSlot returns Shard should be 1, got: %v", committee.ArrayShardCommittee[0].Shard)
	}
	committee, _ = ShardCommitteesAtSlot(state, 1)
	if committee.ArrayShardCommittee[0].Shard != 3 {
		t.Errorf("getShardCommitteesForSlot returns Shard should be 3, got: %v", committee.ArrayShardCommittee[0].Shard)
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
	if _, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1, 0); err == nil {
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
		ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1, 0)
	}
}

func TestShuffleActiveValidatorRegistry(t *testing.T) {
	// Create 1000 validators in ActiveValidatorRegistry.
	var validators []*pb.ValidatorRecord
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{}
		validators = append(validators, validator)
	}

	indices, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1, 0)
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

	indices, err := ShuffleValidatorRegistryToCommittees(common.Hash{'A'}, validators, 1, 0)
	if err != nil {
		t.Errorf("validatorsBySlotShard failed with %v:", err)
	}
	if len(indices) != int(config.EpochLength) {
		t.Errorf("incorret length for validator indices. Want: %d. Got: %d", config.EpochLength, len(indices))
	}
}

func TestGetCommitteesPerSlotSmallValidatorSet(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize / 4

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 0 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 0, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotRegularValidatorSet(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotLargeValidatorSet(t *testing.T) {
	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize * 8

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 8 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 8, committesPerSlot)
	}
}

func TestGetCommitteesPerSlotSmallShardCount(t *testing.T) {
	config := config
	oldShardCount := config.ShardCount
	config.ShardCount = config.EpochLength - 1

	numValidatorRegistry := config.EpochLength * config.TargetCommitteeSize

	committesPerSlot := getCommitteesPerSlot(numValidatorRegistry)
	if committesPerSlot != 1 {
		t.Fatalf("Expected committeesPerSlot to equal %d: got %d", 1, committesPerSlot)
	}

	config.ShardCount = oldShardCount
}

func TestValidatorRegistryBySlotShardRegularValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidatorRegistry := int(config.EpochLength * config.TargetCommitteeSize)
	for i := 0; i < numValidatorRegistry; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	ShardCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(ShardCommitteeArray) != int(config.EpochLength) {
		t.Fatalf("Expected length %d: got %d", config.EpochLength, len(ShardCommitteeArray))
	}

	for i := 0; i < len(ShardCommitteeArray); i++ {
		ShardCommittees := ShardCommitteeArray[i].ArrayShardCommittee
		if len(ShardCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", config.TargetCommitteeSize, 1)
		}

		committeeSize := len(ShardCommittees[0].Committee)
		if committeeSize != int(config.TargetCommitteeSize) {
			t.Fatalf("Expected committee size %d: got %d", config.TargetCommitteeSize, committeeSize)
		}
	}
}

func TestValidatorRegistryBySlotShardLargeValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidatorRegistry := int(config.EpochLength*config.TargetCommitteeSize) * 2
	for i := 0; i < numValidatorRegistry; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	ShardCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(ShardCommitteeArray) != int(config.EpochLength) {
		t.Fatalf("Expected length %d: got %d", config.EpochLength, len(ShardCommitteeArray))
	}

	for i := 0; i < len(ShardCommitteeArray); i++ {
		ShardCommittees := ShardCommitteeArray[i].ArrayShardCommittee
		if len(ShardCommittees) != 2 {
			t.Fatalf("Expected %d committee per slot: got %d", config.TargetCommitteeSize, 2)
		}

		t.Logf("slot %d", i)
		for j := 0; j < len(ShardCommittees); j++ {
			shardCommittee := ShardCommittees[j]
			if len(shardCommittee.Committee) != int(config.TargetCommitteeSize) {
				t.Fatalf("Expected committee size %d: got %d", config.TargetCommitteeSize, len(shardCommittee.Committee))
			}
		}

	}
}

func TestValidatorRegistryBySlotShardSmallValidatorSet(t *testing.T) {
	validatorIndices := []uint32{}
	numValidatorRegistry := int(config.EpochLength * config.TargetCommitteeSize)
	for i := 0; i < numValidatorRegistry; i++ {
		validatorIndices = append(validatorIndices, uint32(i))
	}

	ShardCommitteeArray := splitBySlotShard(validatorIndices, 0)

	if len(ShardCommitteeArray) != int(config.EpochLength) {
		t.Fatalf("Expected length %d: got %d", config.EpochLength, len(ShardCommitteeArray))
	}

	for i := 0; i < len(ShardCommitteeArray); i++ {
		ShardCommittees := ShardCommitteeArray[i].ArrayShardCommittee
		if len(ShardCommittees) != 1 {
			t.Fatalf("Expected %d committee per slot: got %d", config.TargetCommitteeSize,
				len(ShardCommittees))
		}

		for j := 0; j < len(ShardCommittees); j++ {
			shardCommittee := ShardCommittees[j]
			if len(shardCommittee.Committee) != int(config.TargetCommitteeSize) {
				t.Fatalf("Expected committee size %d: got %d", config.TargetCommitteeSize, len(shardCommittee.Committee))
			}
		}
	}
}

func TestAttestationParticipants_ok(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var committeeIndices []uint32
	for i := uint32(0); i < 8; i++ {
		committeeIndices = append(committeeIndices, i)
	}

	var ShardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		ShardCommittees = append(ShardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: i},
				{Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
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
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var ShardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		ShardCommittees = append(ShardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: i},
			},
		})
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	attestationData := &pb.AttestationData{}

	if _, err := AttestationParticipants(state, attestationData, []byte{1}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}

func TestCommitteeCountPerSlot_Ok(t *testing.T) {
	// this defines the # of validators required to have 1 committee
	// per slot for epoch length.
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	tests := []struct {
		validatorCount uint64
		committeeCount uint64
	}{
		{0, 1},
		{1000, 1},
		{2 * validatorsPerEpoch, 2},
		{5 * validatorsPerEpoch, 5},
		{16 * validatorsPerEpoch, 16},
		{32 * validatorsPerEpoch, 16},
	}
	for _, test := range tests {
		if test.committeeCount != committeeCountPerSlot(test.validatorCount) {
			t.Errorf("wanted: %d, got: %d",
				test.committeeCount, committeeCountPerSlot(test.validatorCount))
		}
	}
}

func TestCurrCommitteesCountPerSlot_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(8)
	// set curr epoch total validators count to 8 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if CurrCommitteesCountPerSlot(state) != committeesPerEpoch {
		t.Errorf("Incorrect current epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, CurrCommitteesCountPerSlot(state))
	}
}

func TestPrevCommitteesCountPerSlot_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(3)
	// set prev epoch total validators count to 3 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if prevCommitteesCountPerSlot(state) != committeesPerEpoch {
		t.Errorf("Incorrect prev epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, prevCommitteesCountPerSlot(state))
	}
}

func TestShuffling_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	randaoSeed := [32]byte{'A'}
	slot := uint64(10)
	committees, err := Shuffling(randaoSeed, validators, slot)
	if err != nil {
		t.Fatalf("Could not shuffle validators: %v", err)
	}

	// Verify shuffled list is correctly split into
	// epoch_length * committees_per_slot pieces.
	committeesPerSlot := committeeCountPerSlot(uint64(len(validators)))
	if len(committees) != int(committeesPerSlot*config.EpochLength) {
		t.Errorf("Incorrect committee count after splitting. Wanted: %d, got: %d",
			committeesPerSlot*config.EpochLength, len(committees))
	}

	// Verify each shuffled committee is TARGET_COMMITTEE_SIZE.
	for i := 0; i < len(committees); i++ {
		if len(committees[i]) != int(config.TargetCommitteeSize) {
			t.Errorf("Incorrect validator count per committee. Wanted: %d, got: %d",
				config.TargetCommitteeSize, len(committees[i]))
		}
	}

}

//func TestShuffling_OutOfBound(t *testing.T) {
//	size := 1<<(config.RandBytes*8) - 1
//	validators := make([]*pb.ValidatorRecord, size)
//	for i := 0; i < len(validators); i++ {
//		validators[i] = &pb.ValidatorRecord{
//			ExitSlot: config.FarFutureSlot,
//		}
//	}
//
//	if _, err := Shuffling([32]byte{}, validators, 0); err == nil {
//		t.Fatalf("Shuffling should have failed with exceeded upper bound")
//	}
//}

func TestCrosslinkCommitteesAtSlot_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              200,
	}
	committees, err := CrosslinkCommitteesAtSlot(state, 132)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}
	if len(committees) != int(committeesPerEpoch) {
		t.Errorf("Incorrect committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, len(committees))
	}

	newCommittees, err := CrosslinkCommitteesAtSlot(state, 180)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}

	if reflect.DeepEqual(committees, newCommittees) {
		t.Error("Committees from different slot shall not be equal")
	}
}

func TestCrosslinkCommitteesAtSlot_OutOfBound(t *testing.T) {
	want := fmt.Sprintf(
		"input committee slot %d out of bounds: %d <= slot < %d",
		config.EpochLength+1, 0, config.EpochLength,
	)

	if _, err := CrosslinkCommitteesAtSlot(&pb.BeaconState{}, config.EpochLength+1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

//func TestCrosslinkCommitteesAtPrevSlot_ShuffleFailed(t *testing.T) {
//	size := 1<<(config.RandBytes*8) - 1
//	validators := make([]*pb.ValidatorRecord, size)
//	for i := 0; i < len(validators); i++ {
//		validators[i] = &pb.ValidatorRecord{
//			ExitSlot: config.FarFutureSlot,
//		}
//	}
//
//	state := &pb.BeaconState{
//		ValidatorRegistry: validators,
//		Slot:              100}
//
//	want := fmt.Sprint(
//		"could not shuffle prev epoch validators: " +
//			"input list exceeded upper bound and reached modulo bias",
//	)
//
//	if _, err := CrosslinkCommitteesAtSlot(state, 1); !strings.Contains(err.Error(), want) {
//		t.Errorf("Expected: %s, received: %v", want, err)
//	}
//}

//func TestCrosslinkCommitteesAtCurrSlot_ShuffleFailed(t *testing.T) {
//	size := 1<<(config.RandBytes*8) - 1
//	validators := make([]*pb.ValidatorRecord, size)
//	for i := 0; i < len(validators); i++ {
//		validators[i] = &pb.ValidatorRecord{
//			ExitSlot: config.FarFutureSlot,
//		}
//	}
//
//	state := &pb.BeaconState{
//		ValidatorRegistry: validators,
//		Slot:              100}
//
//	want := fmt.Sprint(
//		"could not shuffle current epoch validators: " +
//			"input list exceeded upper bound and reached modulo bias",
//	)
//
//	if _, err := CrosslinkCommitteesAtSlot(state, 99); !strings.Contains(err.Error(), want) {
//		t.Errorf("Expected: %s, received: %v", want, err)
//	}
//}
