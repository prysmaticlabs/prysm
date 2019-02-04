package validators

import (
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"reflect"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestationParticipants_ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.ValidatorRecord, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
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
			attestationSlot: 2,
			stateSlot:       5,
			shard:           256,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{766, 752},
		},
		{
			attestationSlot: 1,
			stateSlot:       10,
			shard:           128,
			bitfield:        []byte{77},
			wanted:          []uint64{511},
		},
		{
			attestationSlot: 10,
			stateSlot:       20,
			shard:           383,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{3069, 2608},
		},
		{
			attestationSlot: 64,
			stateSlot:       100,
			shard:           0,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{237, 224},
		},
		{
			attestationSlot: 999,
			stateSlot:       1000,
			shard:           1023,
			bitfield:        []byte{99},
			wanted:          []uint64{10494},
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

	validators := make([]*pb.ValidatorRecord, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
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

func TestShuffling_Ok(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().EpochLength * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
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
	committeesPerEpoch = helpers.EpochCommitteeCount(uint64(len(validators)))
	committeesPerSlot := committeesPerEpoch / params.BeaconConfig().EpochLength
	if committeesPerSlot != committeesPerSlot {
		t.Errorf("Incorrect committee count after splitting. Wanted: %d, got: %d",
			committeesPerSlot, len(committees))
	}

	// Verify each shuffled committee is TARGET_COMMITTEE_SIZE.
	for i := 0; i < len(committees); i++ {
		committeeCount := uint64(len(committees[i]))
		if committeeCount*params.BeaconConfig().EpochLength != params.BeaconConfig().TargetCommitteeSize {
			t.Errorf("Incorrect validator count per committee. Wanted: %d, got: %d",
				params.BeaconConfig().TargetCommitteeSize, committeeCount*params.BeaconConfig().EpochLength)
		}
	}

}

func TestShuffling_OutOfBound(t *testing.T) {
	populateValidatorsMax()
	if _, err := Shuffling([32]byte{}, validatorsUpperBound, 0); err == nil {
		t.Fatalf("Shuffling should have failed with exceeded upper bound")
	}
}

func TestCrosslinkCommitteesAtSlot_Ok(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().EpochLength * params.BeaconConfig().TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// Set epoch total validators count to 6 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
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
	if len(committees) != int(committeesPerEpoch*params.BeaconConfig().EpochLength) {
		t.Errorf("Incorrect committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch*params.BeaconConfig().EpochLength, len(committees))
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
		"input committee epoch %d out of bounds: %d <= epoch < %d",
		1, 0, 0,
	)

	if _, err := CrosslinkCommitteesAtSlot(&pb.BeaconState{}, params.BeaconConfig().EpochLength+1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestCrosslinkCommitteesAtPrevSlot_ShuffleFailed(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: validatorsUpperBound,
		Slot:              100,
	}

	want := fmt.Sprint(
		"could not shuffle prev epoch validators: " +
			"input list exceeded upper bound and reached modulo bias",
	)

	if _, err := CrosslinkCommitteesAtSlot(state, 1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestCrosslinkCommitteesAtCurrSlot_ShuffleFailed(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: validatorsUpperBound,
		Slot:              100,
	}

	want := fmt.Sprint(
		"could not shuffle current epoch validators: " +
			"input list exceeded upper bound and reached modulo bias",
	)

	if _, err := CrosslinkCommitteesAtSlot(state, 99); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}
