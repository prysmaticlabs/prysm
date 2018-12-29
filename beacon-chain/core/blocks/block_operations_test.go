package blocks

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessPOWReceiptRoots_SameRootHash(t *testing.T) {
	beaconState := &pb.BeaconState{
		CandidatePowReceiptRoots: []*pb.CandidatePoWReceiptRootRecord{
			{
				CandidatePowReceiptRootHash32: []byte{1},
				VoteCount:                     5,
			},
		},
	}
	block := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1},
	}
	beaconState = ProcessPOWReceiptRoots(beaconState, block)
	newRoots := beaconState.GetCandidatePowReceiptRoots()
	if newRoots[0].GetVoteCount() != 6 {
		t.Errorf("expected votes to increase from 5 to 6, received %d", newRoots[0].GetVoteCount())
	}
}

func TestProcessPOWReceiptRoots_NewCandidateRecord(t *testing.T) {
	beaconState := &pb.BeaconState{
		CandidatePowReceiptRoots: []*pb.CandidatePoWReceiptRootRecord{
			{
				CandidatePowReceiptRootHash32: []byte{0},
				VoteCount:                     5,
			},
		},
	}
	block := &pb.BeaconBlock{
		CandidatePowReceiptRootHash32: []byte{1},
	}
	beaconState = ProcessPOWReceiptRoots(beaconState, block)
	newRoots := beaconState.GetCandidatePowReceiptRoots()
	if len(newRoots) == 1 {
		t.Error("expected new receipt roots to have length > 1")
	}
	if newRoots[1].GetVoteCount() != 1 {
		t.Errorf(
			"expected new receipt roots to have a new element with votes = 1, received votes = %d",
			newRoots[1].GetVoteCount(),
		)
	}
	if !bytes.Equal(newRoots[1].CandidatePowReceiptRootHash32, []byte{1}) {
		t.Errorf(
			"expected new receipt roots to have a new element with root = %#x, received root = %#x",
			[]byte{1},
			newRoots[1].CandidatePowReceiptRootHash32,
		)
	}
}

func TestProcessBlockRandao_UnequalBlockAndProposerRandao(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			RandaoLayers:           0,
			RandaoCommitmentHash32: []byte{},
		},
	}
	block := &pb.BeaconBlock{
		RandaoRevealHash32: []byte{1},
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              1,
		ShardAndCommitteesAtSlots: []*pb.ShardAndCommitteeArray{
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{
						Shard:               0,
						Committee:           []uint32{1, 0},
						TotalValidatorCount: 1,
					},
				},
			},
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{
						Shard:               0,
						Committee:           []uint32{1, 0},
						TotalValidatorCount: 1,
					},
				},
			},
		},
	}

	want := fmt.Sprintf(
		"expected hashed block randao layers to equal proposer randao: received %#x = %#x",
		[32]byte{1},
		[32]byte{0},
	)
	if _, err := ProcessBlockRandao(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockRandao_CreateRandaoMixAndUpdateProposer(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			RandaoLayers:           0,
			RandaoCommitmentHash32: []byte{1},
		},
	}
	block := &pb.BeaconBlock{
		RandaoRevealHash32: []byte{1},
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry:        registry,
		Slot:                     1,
		LatestRandaoMixesHash32S: make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		ShardAndCommitteesAtSlots: []*pb.ShardAndCommitteeArray{
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{
						Shard:               0,
						Committee:           []uint32{1, 0},
						TotalValidatorCount: 1,
					},
				},
			},
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{
						Shard:               0,
						Committee:           []uint32{1, 0},
						TotalValidatorCount: 1,
					},
				},
			},
		},
	}

	newState, err := ProcessBlockRandao(
		beaconState,
		block,
	)
	if err != nil {
		t.Fatalf("Unexpected error processing block randao: %v", err)
	}

	xorRandao := [32]byte{1}
	updatedLatestMix := newState.LatestRandaoMixesHash32S[newState.GetSlot()%params.BeaconConfig().LatestRandaoMixesLength]
	if !bytes.Equal(updatedLatestMix, xorRandao[:]) {
		t.Errorf("Expected randao mix to XOR correctly: wanted %#x, received %#x", xorRandao[:], updatedLatestMix)
	}
	if !bytes.Equal(newState.GetValidatorRegistry()[0].GetRandaoCommitmentHash32(), []byte{1}) {
		t.Errorf(
			"Expected proposer at index 0 to update randao commitment to block randao reveal = %#x, received %#x",
			[]byte{1},
			newState.GetValidatorRegistry()[0].GetRandaoCommitmentHash32(),
		)
	}
}

func TestProcessProposerSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"number of proposer slashings (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxProposerSlashings+1,
		params.BeaconConfig().MaxProposerSlashings,
	)
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	if _, err := ProcessProposerSlashings(
		beaconState,
		block,
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

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "slashing proposal data slots do not match: 1, 0"
	if _, err := ProcessProposerSlashings(
		beaconState,
		block,
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

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "slashing proposal data shards do not match: 0, 1"
	if _, err := ProcessProposerSlashings(
		beaconState,
		block,
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
				Slot:            1,
				Shard:           0,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            1,
				Shard:           0,
				BlockRootHash32: []byte{1, 1, 0},
			},
		},
	}

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"slashing proposal data block roots do not match: %#x, %#x",
		[]byte{0, 1, 0}, []byte{1, 1, 0},
	)

	if _, err := ProcessProposerSlashings(
		beaconState,
		block,
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
				Slot:            1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
		},
	}
	currentSlot := uint64(1)
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := ProcessProposerSlashings(
		beaconState,
		block,
	)
	registry = newState.GetValidatorRegistry()
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

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			CasperSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"number of casper slashings (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxCasperSlashings+1,
		params.BeaconConfig().MaxCasperSlashings,
	)

	if _, err := ProcessCasperSlashings(
		beaconState,
		block,
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

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			CasperSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"exceeded allowed casper votes (%d), received %d",
		params.BeaconConfig().MaxCasperVotes,
		params.BeaconConfig().MaxCasperVotes*2,
	)

	if _, err := ProcessCasperSlashings(
		beaconState,
		block,
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
	beaconState = &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block = &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			CasperSlashings: slashings,
		},
	}
	if _, err := ProcessCasperSlashings(
		beaconState,
		block,
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

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			CasperSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"casper slashing inner vote attestation data should not match: %v, %v",
		att1,
		att1,
	)

	if _, err := ProcessCasperSlashings(
		beaconState,
		block,
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

		beaconState := &pb.BeaconState{
			ValidatorRegistry: registry,
			Slot:              currentSlot,
		}
		block := &pb.BeaconBlock{
			Body: &pb.BeaconBlockBody{
				CasperSlashings: slashings,
			},
		}
		want := fmt.Sprintf(
			`
			Expected the following conditions to hold:
			(vote1.JustifiedSlot < vote2.JustifiedSlot) &&
			(vote2.JustifiedSlot + 1 == vote2.Slot) &&
			(vote2.Slot < vote1.Slot)
			OR
			vote1.Slot == vote.Slot

			Instead, received vote1.JustifiedSlot %d, vote2.JustifiedSlot %d
			and vote1.Slot %d, vote2.Slot %d
			`,
			tt.att1.JustifiedSlot,
			tt.att2.JustifiedSlot,
			tt.att1.Slot,
			tt.att2.Slot,
		)

		if _, err := ProcessCasperSlashings(
			beaconState,
			block,
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

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			CasperSlashings: slashings,
		},
	}
	want := "expected intersection of vote indices to be non-empty"
	if _, err := ProcessCasperSlashings(
		beaconState,
		block,
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
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			CasperSlashings: slashings,
		},
	}
	newState, err := ProcessCasperSlashings(
		beaconState,
		block,
	)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.GetValidatorRegistry()

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

func TestProcessBlockAttestations_ThresholdReached(t *testing.T) {
	attestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations+1)
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{}

	want := fmt.Sprintf(
		"number of attestations in block (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxAttestations+1,
		params.BeaconConfig().MaxAttestations,
	)

	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_InclusionDelayFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot: 5,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot: 5,
	}

	want := fmt.Sprintf(
		"attestation slot (slot %d) + inclusion delay (%d) beyond current beacon state slot (%d)",
		5,
		params.BeaconConfig().MinAttestationInclusionDelay,
		5,
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_EpochDistanceFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot: 5,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot: 5 + 2*params.BeaconConfig().EpochLength,
	}

	want := fmt.Sprintf(
		"attestation slot (slot %d) + epoch length (%d) less than current beacon state slot (%d)",
		5,
		params.BeaconConfig().EpochLength,
		5+2*params.BeaconConfig().EpochLength,
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_JustifiedSlotVerificationFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot:          10,
				JustifiedSlot: 4,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot:          params.BeaconConfig().EpochLength - 1,
		JustifiedSlot: 0,
	}

	want := fmt.Sprintf(
		"expected attestation.JustifiedSlot == state.JustifiedSlot, received %d == %d",
		4,
		0,
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_PreviousJustifiedSlotVerificationFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot:          5,
				JustifiedSlot: 4,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot:                  5 + params.BeaconConfig().EpochLength,
		PreviousJustifiedSlot: 3,
	}

	want := fmt.Sprintf(
		"expected attestation.JustifiedSlot == state.PreviousJustifiedSlot, received %d == %d",
		4,
		3,
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_BlockRootOutOfBounds(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		Slot:                   64,
		PreviousJustifiedSlot:  65,
		LatestBlockRootHash32S: blockRoots,
	}
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot:                     20,
				JustifiedSlot:            65,
				JustifiedBlockRootHash32: []byte{},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	want := "could not get block root for justified slot"
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_BlockRootFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		Slot:                   64,
		PreviousJustifiedSlot:  10,
		LatestBlockRootHash32S: blockRoots,
	}
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot:                     20,
				JustifiedSlot:            10,
				JustifiedBlockRootHash32: []byte{},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	want := fmt.Sprintf(
		"expected JustifiedBlockRoot == getBlockRoot(state, JustifiedSlot): got %#x = %#x",
		[]byte{},
		blockRoots[10],
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CrosslinkRootFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	// If attestation.latest_cross_link_root != state.latest_crosslinks[shard].shard_block_root
	// AND
	// attestation.data.shard_block_root != state.latest_crosslinks[shard].shard_block_root
	// the attestation should be invalid.
	stateLatestCrosslinks := []*pb.CrosslinkRecord{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	state := &pb.BeaconState{
		Slot:                   64,
		PreviousJustifiedSlot:  10,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       stateLatestCrosslinks,
	}
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Shard:                     0,
				Slot:                      20,
				JustifiedSlot:             10,
				JustifiedBlockRootHash32:  blockRoots[10],
				LatestCrosslinkRootHash32: []byte{2},
				ShardBlockRootHash32:      []byte{2},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	want := fmt.Sprintf(
		"attestation.CrossLinkRoot and ShardBlockRoot != %v (state.LatestCrosslinks' ShardBlockRoot)",
		[]byte{1},
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_ShardBlockRootEqualZeroHashFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.CrosslinkRecord{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	state := &pb.BeaconState{
		Slot:                   64,
		PreviousJustifiedSlot:  10,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       stateLatestCrosslinks,
	}
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Shard:                     0,
				Slot:                      20,
				JustifiedSlot:             10,
				JustifiedBlockRootHash32:  blockRoots[10],
				LatestCrosslinkRootHash32: []byte{1},
				ShardBlockRootHash32:      []byte{1},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	want := fmt.Sprintf(
		"expected attestation.ShardBlockRoot == %#x, received %#x instead",
		[]byte{},
		[]byte{1},
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CreatePendingAttestations(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.CrosslinkRecord{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	state := &pb.BeaconState{
		Slot:                   64,
		PreviousJustifiedSlot:  10,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       stateLatestCrosslinks,
	}
	att1 := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                     0,
			Slot:                      20,
			JustifiedSlot:             10,
			JustifiedBlockRootHash32:  blockRoots[10],
			LatestCrosslinkRootHash32: []byte{1},
			ShardBlockRootHash32:      []byte{},
		},
		ParticipationBitfield: []byte{1},
		CustodyBitfield:       []byte{1},
	}
	attestations := []*pb.Attestation{att1}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	newState, err := ProcessBlockAttestations(
		state,
		block,
	)
	pendingAttestations := newState.GetLatestAttestations()
	if err != nil {
		t.Fatalf("Could not produce pending attestations: %v", err)
	}
	if !reflect.DeepEqual(pendingAttestations[0].GetData(), att1.GetData()) {
		t.Errorf(
			"Did not create pending attestation correctly with inner data, wanted %v, received %v",
			att1.GetData(),
			pendingAttestations[0].GetData(),
		)
	}
	if pendingAttestations[0].GetSlotIncluded() != 64 {
		t.Errorf(
			"Pending attestation not included at correct slot: wanted %v, received %v",
			64,
			pendingAttestations[0].GetSlotIncluded(),
		)
	}
}

func TestProcessValidatorExits_ThresholdReached(t *testing.T) {
	exits := make([]*pb.Exit, params.BeaconConfig().MaxExits+1)
	registry := []*pb.ValidatorRecord{}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Exits: exits,
		},
	}

	want := fmt.Sprintf(
		"number of exits (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxExits+1,
		params.BeaconConfig().MaxExits,
	)

	if _, err := ProcessValidatorExits(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_ValidatorNotActive(t *testing.T) {
	exits := []*pb.Exit{
		{
			ValidatorIndex: 0,
		},
	}
	registry := []*pb.ValidatorRecord{
		{
			Status: pb.ValidatorRecord_EXITED_WITH_PENALTY,
		},
	}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Exits: exits,
		},
	}

	want := fmt.Sprintf(
		"expected validator to have active status, received %v",
		pb.ValidatorRecord_EXITED_WITH_PENALTY,
	)

	if _, err := ProcessValidatorExits(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_InvalidExitSlot(t *testing.T) {
	exits := []*pb.Exit{
		{
			Slot: 10,
		},
	}
	registry := []*pb.ValidatorRecord{
		{
			Status: pb.ValidatorRecord_ACTIVE,
		},
	}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Exits: exits,
		},
	}

	want := fmt.Sprintf(
		"expected state.Slot >= exit.Slot, received %d < %d",
		0,
		10,
	)

	if _, err := ProcessValidatorExits(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_InvalidStatusChangeSlot(t *testing.T) {
	exits := []*pb.Exit{
		{
			ValidatorIndex: 0,
			Slot:           0,
		},
	}
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 100,
		},
	}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              10,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Exits: exits,
		},
	}

	want := "expected validator.LatestStatusChangeSlot + PersistentCommitteePeriod >= state.Slot"
	if _, err := ProcessValidatorExits(
		state,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_AppliesCorrectStatus(t *testing.T) {
	exits := []*pb.Exit{
		{
			ValidatorIndex: 0,
			Slot:           0,
		},
	}
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
		},
	}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              10,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Exits: exits,
		},
	}
	newState, err := ProcessValidatorExits(state, block)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.GetValidatorRegistry()
	if newRegistry[0].Status == pb.ValidatorRecord_ACTIVE {
		t.Error("Expected validator status to change, remained ACTIVE")
	}
}
