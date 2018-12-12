package blocks

import (
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessProposerSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); err == nil {
		want := fmt.Sprintf(
			"number of proposer slashings (%d) exceeds allowed threshold of %d",
			params.BeaconConfig().MaxProposerSlashings+1,
			params.BeaconConfig().MaxProposerSlashings,
		)
		t.Errorf("Expected %s, received nil", want)
	}
}

func TestProcessProposerSlashings_UnmatchedSlotNumbers(t *testing.T) {
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot: 1,
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot: 0,
			},
		},
	}

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); err == nil {
		want := "slashing proposal data slots do not match: 1, 0"
		t.Errorf("Expected %s, received nil", want)
	}
}

func TestProcessProposerSlashings_UnmatchedShards(t *testing.T) {
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:  1,
				Shard: 0,
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:  1,
				Shard: 1,
			},
		},
	}

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); err == nil {
		want := "slashing proposal data shards do not match: 0, 1"
		t.Errorf("Expected %s, received nil", want)
	}
}

func TestProcessProposerSlashings_UnmatchedBlockRoots(t *testing.T) {
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:      1,
				Shard:     0,
				BlockRoot: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:      1,
				Shard:     0,
				BlockRoot: []byte{1, 1, 0},
			},
		},
	}

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); err == nil {
		want := fmt.Sprintf(
			"slashing proposal data block roots do not match: %x, %x",
			[]byte{0, 1, 0}, []byte{1, 1, 0},
		)
		t.Errorf("Expected %s, received nil", want)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_EXITED_WITH_PENALTY,
			LatestStatusChangeSlot: 0,
		},
		{
			Status:                 pb.ValidatorRecord_EXITED_WITH_PENALTY,
			LatestStatusChangeSlot: 0,
		},
	}
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:      1,
				Shard:     1,
				BlockRoot: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:      1,
				Shard:     1,
				BlockRoot: []byte{0, 1, 0},
			},
		},
	}
	currentSlot := uint64(1)

	registry, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if registry[1].Status != pb.ValidatorRecord_EXITED_WITH_PENALTY {
		t.Errorf("Proposer with index 1 did not ExitWithPenalty in validator registry: %v", registry[1].Status)
	}
}

func TestIntersection(t *testing.T) {
	testCases := []struct {
		setA []uint32
		setB []uint32
		out  []uint32
	}{
		{[]uint32{2, 3, 5}, []uint32{3}, []uint32{3}},
		{[]uint32{2, 3, 5}, []uint32{3, 5}, []uint32{3, 5}},
		{[]uint32{2, 3, 5}, []uint32{5, 3, 2}, []uint32{5, 3, 2}},
		{[]uint32{2, 3, 5}, []uint32{2, 3, 5}, []uint32{2, 3, 5}},
		{[]uint32{2, 3, 5}, []uint32{}, []uint32{}},
		{[]uint32{}, []uint32{2, 3, 5}, []uint32{}},
		{[]uint32{}, []uint32{}, []uint32{}},
		{[]uint32{1}, []uint32{1}, []uint32{1}},
	}
	for _, tt := range testCases {
		result := intersection(tt.setA, tt.setB)
		if !testEq(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func testEq(a, b []uint32) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
