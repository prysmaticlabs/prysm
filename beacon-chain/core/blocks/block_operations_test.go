package blocks

import (
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestIncorrectProcessProposerSlashings(t *testing.T) {
	// We test exceeding the proposer slashing threshold.
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); err == nil {
		want := "number of proposer slashings exceeds threshold"
		t.Errorf("Expected %s, received nil", want)
	}
	currentSlot++

	// We now test the case with unmatched slot numbers.
	slashings = []*pb.ProposerSlashing{
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
	currentSlot++

	// We now test the case with unmatched shard IDs.
	slashings = []*pb.ProposerSlashing{
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
	currentSlot++

	// We now test the case with unmatched block hashes.
	slashings = []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:        1,
				Shard:       0,
				BlockHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:        1,
				Shard:       0,
				BlockHash32: []byte{1, 1, 0},
			},
		},
	}

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); err == nil {
		want := fmt.Sprintf(
			"slashing proposal data block hashes do not match: %x, %x",
			[]byte{0, 1, 0}, []byte{1, 1, 0},
		)
		t.Errorf("Expected %s, received nil", want)
	}
}

func TestCorrectlyProcessProposerSlashings(t *testing.T) {
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
				Slot:        1,
				Shard:       1,
				BlockHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:        1,
				Shard:       1,
				BlockHash32: []byte{0, 1, 0},
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
