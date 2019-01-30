package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/trie"
)

func TestProcessPOWReceiptRoots_SameRootHash(t *testing.T) {
	beaconState := &pb.BeaconState{
		DepositRootVotes: []*pb.DepositRootVote{
			{
				DepositRootHash32: []byte{1},
				VoteCount:         5,
			},
		},
	}
	block := &pb.BeaconBlock{
		DepositRootHash32: []byte{1},
	}
	beaconState = ProcessDepositRoots(beaconState, block)
	newRoots := beaconState.DepositRootVotes
	if newRoots[0].VoteCount != 6 {
		t.Errorf("expected votes to increase from 5 to 6, received %d", newRoots[0].VoteCount)
	}
}

func TestProcessPOWReceiptRoots_NewCandidateRecord(t *testing.T) {
	beaconState := &pb.BeaconState{
		DepositRootVotes: []*pb.DepositRootVote{
			{
				DepositRootHash32: []byte{0},
				VoteCount:         5,
			},
		},
	}
	block := &pb.BeaconBlock{
		DepositRootHash32: []byte{1},
	}
	beaconState = ProcessDepositRoots(beaconState, block)
	newRoots := beaconState.DepositRootVotes
	if len(newRoots) == 1 {
		t.Error("expected new receipt roots to have length > 1")
	}
	if newRoots[1].VoteCount != 1 {
		t.Errorf(
			"expected new receipt roots to have a new element with votes = 1, received votes = %d",
			newRoots[1].VoteCount,
		)
	}
	if !bytes.Equal(newRoots[1].DepositRootHash32, []byte{1}) {
		t.Errorf(
			"expected new receipt roots to have a new element with root = %#x, received root = %#x",
			[]byte{1},
			newRoots[1].DepositRootHash32,
		)
	}
}

func TestProcessBlockRandao_UnequalBlockAndProposerRandao(t *testing.T) {
	validators := make([]*pb.ValidatorRecord, config.EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot: config.FarFutureSlot,
		}
	}

	block := &pb.BeaconBlock{
		RandaoRevealHash32: []byte{1},
	}

	beaconState := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              1,
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
	randaoCommit := hashutil.RepeatHash([32]byte{}, 1)
	validators := make([]*pb.ValidatorRecord, config.EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: randaoCommit[:],
		}
	}

	block := &pb.BeaconBlock{
		RandaoRevealHash32: randaoCommit[:],
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry:        validators,
		Slot:                     1,
		LatestRandaoMixesHash32S: make([][]byte, config.LatestRandaoMixesLength),
	}

	newState, err := ProcessBlockRandao(
		beaconState,
		block,
	)
	if err != nil {
		t.Fatalf("Unexpected error processing block randao: %v", err)
	}

	updatedLatestMix := newState.LatestRandaoMixesHash32S[newState.Slot%config.LatestRandaoMixesLength]
	if !bytes.Equal(updatedLatestMix, randaoCommit[:]) {
		t.Errorf("Expected randao mix to XOR correctly: wanted %#x, received %#x", randaoCommit[:], updatedLatestMix)
	}
	if !bytes.Equal(newState.ValidatorRegistry[0].RandaoCommitmentHash32, randaoCommit[:]) {
		t.Errorf(
			"Expected proposer at index 0 to update randao commitment to block randao reveal = %#x, received %#x",
			[]byte{1},
			newState.ValidatorRegistry[0].RandaoCommitmentHash32,
		)
	}
}

func TestProcessProposerSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.ProposerSlashing, config.MaxProposerSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"number of proposer slashings (%d) exceeds allowed threshold of %d",
		config.MaxProposerSlashings+1,
		config.MaxProposerSlashings,
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
		false,
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
		false,
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
		false,
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
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.

	validators := make([]*pb.ValidatorRecord, config.EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot:      config.FarFutureSlot,
			PenalizedSlot: 2,
		}
	}
	validatorBalances := make([]uint64, config.EpochLength*2)
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = config.MaxDeposit
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
		ValidatorRegistry:       validators,
		Slot:                    currentSlot,
		ValidatorBalances:       validatorBalances,
		LatestPenalizedBalances: []uint64{0},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := ProcessProposerSlashings(
		beaconState,
		block,
		false,
	)
	validators = newState.ValidatorRegistry
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if validators[1].ExitSlot !=
		beaconState.Slot+config.EntryExitDelay {
		t.Errorf("Proposer with index 1 did not correctly exit,"+"wanted slot:%d, got:%d",
			beaconState.Slot+config.EntryExitDelay, validators[1].ExitSlot)
	}
}

func TestProcessAttesterSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.AttesterSlashing, config.MaxAttesterSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"number of attester slashings (%d) exceeds allowed threshold of %d",
		config.MaxAttesterSlashings+1,
		config.MaxAttesterSlashings,
	)

	if _, err := ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_EmptyCustodyFields(t *testing.T) {
	slashings := []*pb.AttesterSlashing{
		{
			SlashableVote_1: &pb.SlashableVote{
				ValidatorIndices: make(
					[]uint64,
					config.MaxIndicesPerSlashableVote,
				),
			},
			SlashableVote_2: &pb.SlashableVote{
				ValidatorIndices: make(
					[]uint64,
					config.MaxIndicesPerSlashableVote,
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
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("custody bit field can't all be 0")

	if _, err := ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	// Perform the same check for SlashableVoteData_2.
	slashings = []*pb.AttesterSlashing{
		{
			SlashableVote_1: &pb.SlashableVote{
				ValidatorIndices: make(
					[]uint64,
					config.MaxIndicesPerSlashableVote,
				),
			},
			SlashableVote_2: &pb.SlashableVote{
				ValidatorIndices: make(
					[]uint64,
					config.MaxIndicesPerSlashableVote,
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
			AttesterSlashings: slashings,
		},
	}
	if _, err := ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_UnmatchedAttestations(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot: 5,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableVote_1: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{1},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableVote_2: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{2},
				CustodyBitfield:  []byte{0xFF},
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
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"attester slashing inner slashable vote data attestation should not match: %v, %v",
		att1,
		att1,
	)

	if _, err := ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_SlotsInequalities(t *testing.T) {
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
		slashings := []*pb.AttesterSlashing{
			{
				SlashableVote_1: &pb.SlashableVote{
					Data:             tt.att1,
					ValidatorIndices: []uint64{1},
					CustodyBitfield:  []byte{0xFF},
				},
				SlashableVote_2: &pb.SlashableVote{
					Data:             tt.att2,
					ValidatorIndices: []uint64{2},
					CustodyBitfield:  []byte{0xFF},
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
				AttesterSlashings: slashings,
			},
		}
		want := fmt.Sprintf(
			`
			Expected the following conditions to hold:
			(slashableVoteData1.JustifiedSlot <
			slashableVoteData2.JustifiedSlot) &&
			(slashableVoteData2.JustifiedSlot + 1
			== slashableVoteData1.Slot) &&
			(slashableVoteData2.Slot < slashableVoteData1.Slot)
			OR
			slashableVoteData1.Slot == slashableVoteData2.Slot

			Instead, received slashableVoteData1.JustifiedSlot %d,
			slashableVoteData2.JustifiedSlot %d
			and slashableVoteData1.Slot %d, slashableVoteData2.Slot %d
			`,
			tt.att1.JustifiedSlot,
			tt.att2.JustifiedSlot,
			tt.att1.Slot,
			tt.att2.Slot,
		)

		if _, err := ProcessAttesterSlashings(
			beaconState,
			block,
			false,
		); !strings.Contains(err.Error(), want) {
			t.Errorf("Expected %s, received %v", want, err)
		}
	}
}

func TestProcessAttesterSlashings_EmptyVoteIndexIntersection(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableVote_1: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{1},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableVote_2: &pb.SlashableVote{
				Data:             att2,
				ValidatorIndices: []uint64{2},
				CustodyBitfield:  []byte{0xFF},
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
			AttesterSlashings: slashings,
		},
	}
	want := "expected intersection of vote indices to be non-empty"
	if _, err := ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	validators := make([]*pb.ValidatorRecord, config.EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot:      config.FarFutureSlot,
			PenalizedSlot: 6,
		}
	}

	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableVote_1: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{1},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableVote_2: &pb.SlashableVote{
				Data:             att2,
				ValidatorIndices: []uint64{1},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}

	currentSlot := uint64(5)
	beaconState := &pb.BeaconState{
		ValidatorRegistry:       validators,
		Slot:                    currentSlot,
		ValidatorBalances:       []uint64{32, 32, 32, 32, 32, 32},
		LatestPenalizedBalances: []uint64{0},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	newState, err := ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.ValidatorRegistry

	// Given the intersection of slashable indices is [1], only validator
	// at index 1 should be penalized and exited. We confirm this below.
	if newRegistry[1].ExitSlot != config.EntryExitDelay+currentSlot {
		t.Errorf(
			`
			Expected validator at index 1's exit slot to change to
			%d, received %d instead
			`,
			config.EntryExitDelay+currentSlot, newRegistry[1].ExitSlot,
		)
	}
	if newRegistry[0].ExitSlot != config.FarFutureSlot {
		t.Errorf(
			`
			Expected validator at index 0's exit slot to not change,
			received %d instead
			`,
			newRegistry[0].ExitSlot,
		)
	}
}

func TestProcessBlockAttestations_ThresholdReached(t *testing.T) {
	attestations := make([]*pb.Attestation, config.MaxAttestations+1)
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{}

	want := fmt.Sprintf(
		"number of attestations in block (%d) exceeds allowed threshold of %d",
		config.MaxAttestations+1,
		config.MaxAttestations,
	)

	if _, err := ProcessBlockAttestations(
		state,
		block,
		false,
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
		config.MinAttestationInclusionDelay,
		5,
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
		false,
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
		Slot: 5 + 2*config.EpochLength,
	}

	want := fmt.Sprintf(
		"attestation slot (slot %d) + epoch length (%d) less than current beacon state slot (%d)",
		5,
		config.EpochLength,
		5+2*config.EpochLength,
	)
	if _, err := ProcessBlockAttestations(
		state,
		block,
		false,
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
		Slot:          config.EpochLength - 1,
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
		false,
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
		Slot:                  5 + config.EpochLength,
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
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_BlockRootOutOfBounds(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_BlockRootFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CrosslinkRootFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_ShardBlockRootEqualZeroHashFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CreatePendingAttestations(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
		false,
	)
	pendingAttestations := newState.LatestAttestations
	if err != nil {
		t.Fatalf("Could not produce pending attestations: %v", err)
	}
	if !reflect.DeepEqual(pendingAttestations[0].Data, att1.Data) {
		t.Errorf(
			"Did not create pending attestation correctly with inner data, wanted %v, received %v",
			att1.Data,
			pendingAttestations[0].Data,
		)
	}
	if pendingAttestations[0].SlotIncluded != 64 {
		t.Errorf(
			"Pending attestation not included at correct slot: wanted %v, received %v",
			64,
			pendingAttestations[0].SlotIncluded,
		)
	}
}

func TestProcessValidatorDeposits_ThresholdReached(t *testing.T) {
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: make([]*pb.Deposit, config.MaxDeposits+1),
		},
	}
	beaconState := &pb.BeaconState{}
	want := "exceeds allowed threshold"
	if _, err := ProcessValidatorDeposits(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_DepositDataSizeTooSmall(t *testing.T) {
	data := []byte{1, 2, 3}
	deposit := &pb.Deposit{
		DepositData: data,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{}
	want := "deposit data slice too small"
	if _, err := ProcessValidatorDeposits(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_DepositInputDecodingFails(t *testing.T) {
	data := make([]byte, 16)
	deposit := &pb.Deposit{
		DepositData: data,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{}
	want := "ssz decode failed"
	if _, err := ProcessValidatorDeposits(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_MerkleBranchFailsVerification(t *testing.T) {
	// We create a correctly encoded deposit data using Simple Serialize.
	depositInput := &pb.DepositInput{
		Pubkey: []byte{1, 2, 3},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}
	value := make([]byte, 8)
	timestamp := make([]byte, 8)
	data = append(data, encodedInput...)
	data = append(data, value...)
	data = append(data, timestamp...)

	// We then create a merkle branch for the test.
	depositTrie := trie.NewDepositTrie()
	depositTrie.UpdateDepositTrie(data)
	branch := depositTrie.GenerateMerkleBranch(0)

	deposit := &pb.Deposit{
		DepositData:         data,
		MerkleBranchHash32S: branch,
		MerkleTreeIndex:     0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{
		LatestDepositRootHash32: []byte{},
	}
	want := "merkle branch of deposit root did not verify"
	if _, err := ProcessValidatorDeposits(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_ProcessDepositHelperFuncFails(t *testing.T) {
	// Having mismatched withdrawal credentials will cause the process deposit
	// validator helper function to fail with error when the public key
	// currently exists in the validator registry.
	depositInput := &pb.DepositInput{
		Pubkey:                      []byte{1},
		WithdrawalCredentialsHash32: []byte{1, 2, 3},
		ProofOfPossession:           []byte{},
		RandaoCommitmentHash32:      []byte{0},
		CustodyCommitmentHash32:     []byte{0},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, uint64(1000))

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * config.SlotDuration
	binary.BigEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	// We then create a merkle branch for the test.
	depositTrie := trie.NewDepositTrie()
	depositTrie.UpdateDepositTrie(data)
	branch := depositTrie.GenerateMerkleBranch(0)

	deposit := &pb.Deposit{
		DepositData:         data,
		MerkleBranchHash32S: branch,
		MerkleTreeIndex:     0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	// The validator will have a mismatched withdrawal credential than
	// the one specified in the deposit input, causing a failure.
	registry := []*pb.ValidatorRecord{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{4, 5, 6},
		},
	}
	balances := []uint64{0}
	root := depositTrie.Root()
	beaconState := &pb.BeaconState{
		ValidatorRegistry:       registry,
		ValidatorBalances:       balances,
		LatestDepositRootHash32: root[:],
		Slot:                    currentSlot,
		GenesisTime:             uint64(genesisTime),
	}
	want := "expected withdrawal credentials to match"
	if _, err := ProcessValidatorDeposits(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_ProcessCorrectly(t *testing.T) {
	depositInput := &pb.DepositInput{
		Pubkey:                      []byte{1},
		WithdrawalCredentialsHash32: []byte{1, 2, 3},
		ProofOfPossession:           []byte{},
		RandaoCommitmentHash32:      []byte{0},
		CustodyCommitmentHash32:     []byte{0},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	depositValue := uint64(1000)
	binary.BigEndian.PutUint64(value, depositValue)

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * config.SlotDuration
	binary.BigEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	// We then create a merkle branch for the test.
	depositTrie := trie.NewDepositTrie()
	depositTrie.UpdateDepositTrie(data)
	branch := depositTrie.GenerateMerkleBranch(0)

	deposit := &pb.Deposit{
		DepositData:         data,
		MerkleBranchHash32S: branch,
		MerkleTreeIndex:     0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	registry := []*pb.ValidatorRecord{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	root := depositTrie.Root()
	beaconState := &pb.BeaconState{
		ValidatorRegistry:       registry,
		ValidatorBalances:       balances,
		LatestDepositRootHash32: root[:],
		Slot:                    currentSlot,
		GenesisTime:             uint64(genesisTime),
	}
	newState, err := ProcessValidatorDeposits(
		beaconState,
		block,
	)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}
	if newState.ValidatorBalances[0] != depositValue {
		t.Errorf(
			"Expected state validator balances index 0 to equal %d, received %d",
			depositValue,
			newState.ValidatorBalances[0],
		)
	}
}

func TestProcessValidatorExits_ThresholdReached(t *testing.T) {
	exits := make([]*pb.Exit, config.MaxExits+1)
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
		config.MaxExits+1,
		config.MaxExits,
	)

	if _, err := ProcessValidatorExits(
		state,
		block,
		false,
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
			ExitSlot: 0,
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
		"expected exit.Slot > state.Slot + EntryExitDelay, received 0 < %d",
		config.EntryExitDelay,
	)

	if _, err := ProcessValidatorExits(
		state,
		block,
		false,
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
			ExitSlot: config.FarFutureSlot,
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
		false,
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
			ExitSlot: 1,
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

	want := "expected exit.Slot > state.Slot + EntryExitDelay"
	if _, err := ProcessValidatorExits(
		state,
		block,
		false,
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
			ExitSlot: config.FarFutureSlot,
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
	newState, err := ProcessValidatorExits(state, block, false)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.ValidatorRegistry
	if newRegistry[0].StatusFlags == pb.ValidatorRecord_INITIAL {
		t.Error("Expected validator status to change, remained INITIAL")
	}
}
