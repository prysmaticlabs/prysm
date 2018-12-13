package blocks

import (
	"fmt"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessProposerSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"number of proposer slashings (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxProposerSlashings+1,
		params.BeaconConfig().MaxProposerSlashings,
	)

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
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

	want := "slashing proposal data slots do not match: 1, 0"
	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
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

	want := "slashing proposal data shards do not match: 0, 1"
	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
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

	want := fmt.Sprintf(
		"slashing proposal data block roots do not match: %#x, %#x",
		[]byte{0, 1, 0}, []byte{1, 1, 0},
	)

	if _, err := ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
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

func TestProcessCasperSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.CasperSlashing, params.BeaconConfig().MaxCasperSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"number of casper slashings (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxCasperSlashings+1,
		params.BeaconConfig().MaxCasperSlashings,
	)

	if _, err := ProcessCasperSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessCasperSlashings_VoteThresholdReached(t *testing.T) {
	slashings := []*pb.CasperSlashing{
		{
			Votes_1: &pb.SlashableVoteData{
				AggregateSignaturePoc_0Indices: make(
					[]uint32,
					params.BeaconConfig().MaxCasperVotes,
				),
				AggregateSignaturePoc_1Indices: make(
					[]uint32,
					params.BeaconConfig().MaxCasperVotes,
				),
			},
		},
	}
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"exceeded allowed casper votes (%d), received %d",
		params.BeaconConfig().MaxCasperVotes,
		params.BeaconConfig().MaxCasperVotes*2,
	)

	if _, err := ProcessCasperSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	// Perform the same check for Votes_2.
	slashings = []*pb.CasperSlashing{
		{
			Votes_2: &pb.SlashableVoteData{
				AggregateSignaturePoc_0Indices: make(
					[]uint32,
					params.BeaconConfig().MaxCasperVotes,
				),
				AggregateSignaturePoc_1Indices: make(
					[]uint32,
					params.BeaconConfig().MaxCasperVotes,
				),
			},
		},
	}
	if _, err := ProcessCasperSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessCasperSlashings_UnmatchedAttestations(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot: 5,
	}
	slashings := []*pb.CasperSlashing{
		{
			Votes_1: &pb.SlashableVoteData{
				Data: att1,
			},
			Votes_2: &pb.SlashableVoteData{
				Data: att1,
			},
		},
	}
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"casper slashing inner vote attestation data should not match: %v, %v",
		att1,
		att1,
	)

	if _, err := ProcessCasperSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessCasperSlashings_SlotsInequalities(t *testing.T) {
	testCases := []struct {
		att1 *pb.AttestationData
		att2 *pb.AttestationData
	}{
		{
			// Case 0: vote1.JustifiedSlot < vote2.JustifiedSlot is false
			// vote2.JustifiedSlot + 1 == vote2.Slot is true
			// vote2.Slot < vote1.Slot is true
			// and slots are unequal.
			att1: &pb.AttestationData{
				JustifiedSlot: 4,
				Slot:          6,
			},
			att2: &pb.AttestationData{
				JustifiedSlot: 4,
				Slot:          5,
			},
		},
		{
			// Case 1: vote1.JustifiedSlot < vote2.JustifiedSlot is false
			// vote2.JustifiedSlot + 1 == vote2.Slot is false
			// vote2.Slot < vote1.Slot is true
			// and slots are unequal.
			att1: &pb.AttestationData{
				JustifiedSlot: 4,
				Slot:          8,
			},
			att2: &pb.AttestationData{
				JustifiedSlot: 4,
				Slot:          7,
			},
		},
		{
			// Case 2: vote1.JustifiedSlot < vote2.JustifiedSlot is false
			// vote2.JustifiedSlot + 1 == vote2.Slot is false
			// vote2.Slot < vote1.Slot is false
			// and slots are unequal.
			att1: &pb.AttestationData{
				JustifiedSlot: 4,
				Slot:          6,
			},
			att2: &pb.AttestationData{
				JustifiedSlot: 4,
				Slot:          7,
			},
		},
	}
	for _, tt := range testCases {
		slashings := []*pb.CasperSlashing{
			{
				Votes_1: &pb.SlashableVoteData{
					Data: tt.att1,
				},
				Votes_2: &pb.SlashableVoteData{
					Data: tt.att2,
				},
			},
		}
		registry := []*pb.ValidatorRecord{}
		currentSlot := uint64(0)

		want := fmt.Sprintf(
			`
			Expected the following conditions to hold:
			(vote1.JustifiedSlot < vote2.JustifiedSlot) &&
			(vote2.JustifiedSlot + 1 == vote2.Slot) &&
			(vote2.Slot < vote1.Slot)
			OR
			vote1.Slot == vote.Slot

			Instead, received vote1.JustifiedSlot %d, vote.JustifiedSlot %d
			and vote1.Slot %d, vote2.Slot %d
			`,
			tt.att1.JustifiedSlot,
			tt.att2.JustifiedSlot,
			tt.att1.Slot,
			tt.att2.Slot,
		)

		if _, err := ProcessCasperSlashings(
			registry,
			slashings,
			currentSlot,
		); !strings.Contains(err.Error(), want) {
			t.Errorf("Expected %s, received %v", want, err)
		}
	}
}

func TestProcessCasperSlashings_EmptyVoteIndexIntersection(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	slashings := []*pb.CasperSlashing{
		{
			Votes_1: &pb.SlashableVoteData{
				Data:                           att1,
				AggregateSignaturePoc_0Indices: []uint32{1, 2},
				AggregateSignaturePoc_1Indices: []uint32{3, 4},
			},
			Votes_2: &pb.SlashableVoteData{
				Data:                           att2,
				AggregateSignaturePoc_0Indices: []uint32{5, 6},
				AggregateSignaturePoc_1Indices: []uint32{7, 8},
			},
		},
	}
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := "expected intersection of vote indices to be non-empty"
	if _, err := ProcessCasperSlashings(
		registry,
		slashings,
		currentSlot,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessCasperSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
		},
	}

	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	slashings := []*pb.CasperSlashing{
		{
			Votes_1: &pb.SlashableVoteData{
				Data:                           att1,
				AggregateSignaturePoc_0Indices: []uint32{0, 1},
				AggregateSignaturePoc_1Indices: []uint32{2, 3},
			},
			Votes_2: &pb.SlashableVoteData{
				Data:                           att2,
				AggregateSignaturePoc_0Indices: []uint32{4, 5},
				AggregateSignaturePoc_1Indices: []uint32{6, 1},
			},
		},
	}

	currentSlot := uint64(5)
	newRegistry, err := ProcessCasperSlashings(
		registry,
		slashings,
		currentSlot,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Given the intersection of slashable indices is [1], only validator
	// at index 1 should be penalized and change Status. We confirm this below.
	if newRegistry[1].Status != pb.ValidatorRecord_EXITED_WITH_PENALTY {
		t.Errorf(
			`
			Expected validator at index 1's status to change to 
			EXITED_WITH_PENALTY, received %v instead
			`,
			newRegistry[1].Status,
		)
	}
	if newRegistry[0].Status != pb.ValidatorRecord_ACTIVE {
		t.Errorf(
			`
			Expected validator at index 0's status to remain 
			ACTIVE, received %v instead
			`,
			newRegistry[1].Status,
		)
	}
}
