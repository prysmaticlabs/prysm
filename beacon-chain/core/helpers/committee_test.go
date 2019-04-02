package helpers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var size = 1<<(params.BeaconConfig().RandBytes*8) - 1
var validatorsUpperBound = make([]*pb.Validator, size)
var validator = &pb.Validator{
	ExitEpoch: params.BeaconConfig().FarFutureEpoch,
}

func populateValidatorsMax() {
	for i := 0; i < len(validatorsUpperBound); i++ {
		validatorsUpperBound[i] = validator
	}
}

func TestEpochCommitteeCount_OK(t *testing.T) {
	// this defines the # of validators required to have 1 committee
	// per slot for epoch length.
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	tests := []struct {
		validatorCount uint64
		committeeCount uint64
	}{
		{0, params.BeaconConfig().SlotsPerEpoch},
		{1000, params.BeaconConfig().SlotsPerEpoch},
		{2 * validatorsPerEpoch, 2 * params.BeaconConfig().SlotsPerEpoch},
		{5 * validatorsPerEpoch, 5 * params.BeaconConfig().SlotsPerEpoch},
		{16 * validatorsPerEpoch, 16 * params.BeaconConfig().SlotsPerEpoch},
		{32 * validatorsPerEpoch, 16 * params.BeaconConfig().SlotsPerEpoch},
	}
	for _, test := range tests {
		if test.committeeCount != EpochCommitteeCount(test.validatorCount) {
			t.Errorf("wanted: %d, got: %d",
				test.committeeCount, EpochCommitteeCount(test.validatorCount))
		}
	}
}

func TestEpochCommitteeCount_LessShardsThanEpoch(t *testing.T) {
	validatorCount := uint64(8)
	productionConfig := params.BeaconConfig()
	testConfig := &params.BeaconChainConfig{
		ShardCount:          1,
		SlotsPerEpoch:       4,
		TargetCommitteeSize: 2,
	}
	params.OverrideBeaconConfig(testConfig)
	if EpochCommitteeCount(validatorCount) != validatorCount/testConfig.TargetCommitteeSize {
		t.Errorf("wanted: %d, got: %d",
			validatorCount/testConfig.TargetCommitteeSize, EpochCommitteeCount(validatorCount))
	}
	params.OverrideBeaconConfig(productionConfig)
}

func TestCurrentEpochCommitteeCount_OK(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(8)
	// set curr epoch total validators count to 8 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if CurrentEpochCommitteeCount(state) != committeesPerEpoch*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Incorrect current epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, CurrentEpochCommitteeCount(state))
	}
}

func TestPrevEpochCommitteeCount_OK(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(3)
	// set prev epoch total validators count to 3 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if PrevEpochCommitteeCount(state) != committeesPerEpoch*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Incorrect prev epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, PrevEpochCommitteeCount(state))
	}
}

func TestNextEpochCommitteeCount_OK(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// set prev epoch total validators count to 3 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}
	if NextEpochCommitteeCount(state) != committeesPerEpoch*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Incorrect next epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, NextEpochCommitteeCount(state))
	}
}

func TestShuffling_OK(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	randaoSeed := [32]byte{'A'}
	slot := uint64(10)
	committees, err := Shuffling(randaoSeed, validators, slot)
	if err != nil {
		t.Fatalf("Could not shuffle validators: %v", err)
	}

	// Verify shuffled list is correctly split into committees_per_slot pieces.
	committeesPerEpoch = EpochCommitteeCount(uint64(len(validators)))
	committeesPerSlot := committeesPerEpoch / params.BeaconConfig().SlotsPerEpoch
	if committeesPerSlot != committeesPerSlot {
		t.Errorf("Incorrect committee count after splitting. Wanted: %d, got: %d",
			committeesPerSlot, len(committees))
	}

	// Verify each shuffled committee is TARGET_COMMITTEE_SIZE.
	for i := 0; i < len(committees); i++ {
		committeeCount := uint64(len(committees[i]))
		if committeeCount != params.BeaconConfig().TargetCommitteeSize {
			t.Errorf("Incorrect validator count per committee. Wanted: %d, got: %d",
				params.BeaconConfig().TargetCommitteeSize, committeeCount)
		}
	}

}

func TestShuffling_OutOfBound(t *testing.T) {
	populateValidatorsMax()
	if _, err := Shuffling([32]byte{}, validatorsUpperBound, 0); err == nil {
		t.Fatalf("Shuffling should have failed with exceeded upper bound")
	}
}

func TestCrosslinkCommitteesAtSlot_OK(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              params.BeaconConfig().GenesisSlot + 200,
	}
	committees, err := CrosslinkCommitteesAtSlot(state, params.BeaconConfig().GenesisSlot+132, false)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}
	if len(committees) != int(committeesPerEpoch) {
		t.Errorf("Incorrect committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, len(committees))
	}

	newCommittees, err := CrosslinkCommitteesAtSlot(state, params.BeaconConfig().GenesisSlot+180, false)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}

	if reflect.DeepEqual(committees, newCommittees) {
		t.Error("Committees from different slot shall not be equal")
	}
}

func TestCrosslinkCommitteesAtSlot_RegistryChange(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(4)
	// Set epoch total validators count to 4 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		Slot:                   params.BeaconConfig().GenesisSlot,
		LatestIndexRootHash32S: [][]byte{{'A'}, {'B'}},
		LatestRandaoMixes:      [][]byte{{'C'}, {'D'}},
	}

	committees, err := CrosslinkCommitteesAtSlot(state, params.BeaconConfig().GenesisSlot+100, true)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}
	if len(committees) != int(committeesPerEpoch) {
		t.Errorf("Incorrect committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, len(committees))
	}
}

func TestCrosslinkCommitteesAtSlot_EpochSinceLastUpdatePow2(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(5)
	// Set epoch total validators count to 5 committees per slot.
	validators := make([]*pb.Validator, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:            validators,
		Slot:                         params.BeaconConfig().GenesisSlot + 128,
		LatestIndexRootHash32S:       [][]byte{{'A'}, {'B'}, {'C'}},
		LatestRandaoMixes:            [][]byte{{'D'}, {'E'}, {'F'}},
		ValidatorRegistryUpdateEpoch: params.BeaconConfig().GenesisEpoch,
	}

	committees, err := CrosslinkCommitteesAtSlot(state, params.BeaconConfig().GenesisSlot+192, false)
	if err != nil {
		t.Fatalf("Could not get crosslink committee: %v", err)
	}
	if len(committees) != int(committeesPerEpoch) {
		t.Errorf("Incorrect committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, len(committees))
	}
}

func TestCrosslinkCommitteesAtSlot_OutOfBound(t *testing.T) {
	want := fmt.Sprintf(
		"input committee epoch %d out of bounds: %d <= epoch <= %d",
		0,
		1,
		2,
	)
	slot := params.BeaconConfig().GenesisSlot
	beaconState := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch*2,
	}

	if _, err := CrosslinkCommitteesAtSlot(beaconState, slot, false); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestCrosslinkCommitteesAtSlot_ShuffleFailed(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: validatorsUpperBound,
		Slot:              params.BeaconConfig().GenesisSlot + 100,
	}

	want := fmt.Sprint(
		"input list exceeded upper bound and reached modulo bias",
	)

	if _, err := CrosslinkCommitteesAtSlot(state, params.BeaconConfig().GenesisSlot+1, false); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestAttestationParticipants_OK(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
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
		wanted          []uint64
	}{
		{
			attestationSlot: params.BeaconConfig().GenesisSlot + 2,
			stateSlot:       params.BeaconConfig().GenesisSlot + 5,
			shard:           2,
			bitfield:        []byte{0xC0},
			wanted:          []uint64{11, 14},
		},
		{
			attestationSlot: params.BeaconConfig().GenesisSlot + 1,
			stateSlot:       params.BeaconConfig().GenesisSlot + 10,
			shard:           1,
			bitfield:        []byte{0x80},
			wanted:          []uint64{5},
		},
		{
			attestationSlot: params.BeaconConfig().GenesisSlot + 10,
			stateSlot:       params.BeaconConfig().GenesisSlot + 10,
			shard:           10,
			bitfield:        []byte{0xC0},
			wanted:          []uint64{55, 105},
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
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}
	attestationData := &pb.AttestationData{}

	if _, err := AttestationParticipants(state, attestationData, []byte{}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}

func TestVerifyBitfield_OK(t *testing.T) {
	bitfield := []byte{0xFF}
	committeeSize := 8

	isValidated, err := VerifyBitfield(bitfield, committeeSize)
	if err != nil {
		t.Fatal(err)
	}

	if !isValidated {
		t.Error("bitfield is not validated when it was supposed to be")
	}

	bitfield = []byte{0xff, 0x01}
	committeeSize = 9

	isValidated, err = VerifyBitfield(bitfield, committeeSize)
	if err != nil {
		t.Fatal(err)
	}

	if isValidated {
		t.Error("bitfield is validated when it was supposed to be")
	}

	bitfield = []byte{0xff, 0x80}
	committeeSize = 10
	isValidated, err = VerifyBitfield(bitfield, committeeSize)
	if err != nil {
		t.Fatal(err)
	}

	if !isValidated {
		t.Error("bitfield is not validated when it was supposed to be")
	}
}

func TestCommitteeAssignment_CanRetrieve(t *testing.T) {
	// Initialize test with 128 validators, each slot and each shard gets 2 validators.
	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
	}

	tests := []struct {
		index      uint64
		slot       uint64
		committee  []uint64
		shard      uint64
		isProposer bool
	}{
		// Next epoch
		{
			index:      0,
			slot:       params.BeaconConfig().GenesisSlot + 159,
			committee:  []uint64{0, 123},
			shard:      31,
			isProposer: false,
		},
		{
			index:      105,
			slot:       params.BeaconConfig().GenesisSlot + 185,
			committee:  []uint64{1, 105},
			shard:      57,
			isProposer: true,
		},
		{
			index:      64,
			slot:       params.BeaconConfig().GenesisSlot + 176,
			committee:  []uint64{64, 12},
			shard:      48,
			isProposer: true,
		},
		{
			index:      11,
			slot:       params.BeaconConfig().GenesisSlot + 188,
			committee:  []uint64{112, 11},
			shard:      60,
			isProposer: false,
		},
		// Fetch current epoch
		{
			index:      3,
			slot:       params.BeaconConfig().GenesisSlot + 69,
			committee:  []uint64{2, 3},
			shard:      5,
			isProposer: true,
		},
		// Fetch previous epoch
		{
			index:      70,
			slot:       params.BeaconConfig().GenesisSlot + 59,
			committee:  []uint64{40, 70},
			shard:      59,
			isProposer: true,
		},
	}

	// Update cache for assignments
	if err := RecalculateAssignmentsCache(state, state.Slot); err != nil {
		t.Fatalf("failed to recalculate assignments cache: %v", err)
	}

	for _, tt := range tests {
		committee, shard, slot, isProposer, err := CommitteeAssignment(
			state, tt.slot, tt.index, false)
		if err != nil {
			t.Fatalf("failed to execute NextEpochCommitteeAssignment: %v", err)
		}
		if shard != tt.shard {
			t.Errorf("wanted shard %d, got shard %d",
				tt.shard, shard)
		}
		if slot != tt.slot {
			t.Errorf("wanted slot %d, got slot %d",
				tt.slot, slot)
		}
		if isProposer != tt.isProposer {
			t.Errorf("wanted isProposer %v, got isProposer %v",
				tt.isProposer, isProposer)
		}
		if !reflect.DeepEqual(committee, tt.committee) {
			t.Errorf("wanted committee %v, got committee %v",
				tt.committee, committee)
		}
	}
}

func TestCommitteeAssignment_CachesOnTransition(t *testing.T) {
	// Initialize test with 128 validators, each slot and each shard gets 2 validators.
	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		Slot:                   params.BeaconConfig().GenesisSlot,
		LatestIndexRootHash32S: [][]byte{{'A'}, {'B'}, {'C'}},
		LatestRandaoMixes:      [][]byte{{'D'}, {'E'}, {'F'}},
	}

	if err := RecalculateAssignmentsCache(state, state.Slot); err != nil {
		t.Fatalf("failed to recalculate assignments cache: %v", err)
	}

	if _, ok := assignmentCache.prevAssignments[0]; ok {
		t.Fatal("previous assignments should not be cached in genesis state")
	}

	state.Slot += params.BeaconConfig().SlotsPerEpoch
	if err := RecalculateAssignmentsCache(state, state.Slot); err != nil {
		t.Fatalf("failed to recalculate assignments cache: %v", err)
	}

	if _, ok := assignmentCache.prevAssignments[0]; !ok {
		t.Fatal("previous assignments should be cached after transition")
	}
}

func TestCommitteeAssignment_CantFindValidator(t *testing.T) {
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
	}
	index := uint64(10000)
	_, _, _, _, err := CommitteeAssignment(state, state.Slot, index, false)
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatal(err)
	}
	if statusErr.Code() != codes.NotFound {
		t.Errorf("Unexpected error: %v", err)
	}
}

func BenchmarkCommitteeAssignment(b *testing.B) {
	// Initialize test with 128 validators, each slot and each shard gets 2 validators.
	validators := make([]*pb.Validator, 8192)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
	}

	slot := params.BeaconConfig().GenesisSlot + 146

	if err := RecalculateAssignmentsCache(beaconState, beaconState.Slot); err != nil {
		b.Fatal(err)
	}

	b.N = 5000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _, err := CommitteeAssignment(beaconState, slot, 800, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}
