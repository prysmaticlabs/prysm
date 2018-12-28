package state

import (
	"fmt"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessBlock_IncorrectSlot(t *testing.T) {
	beaconState := &pb.BeaconState{
		Slot: 5,
	}
	block := &pb.BeaconBlock{
		Slot: 4,
	}
	want := fmt.Sprintf(
		"block.slot != state.slot, block.slot = %d, state.slot = %d",
		4,
		5,
	)
	if _, err := ProcessBlock(beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectBlockRandao(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{0},
			RandaoLayers:           0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{0},
			RandaoLayers:           0,
		},
	}
	beaconState := &pb.BeaconState{
		Slot:              0,
		ValidatorRegistry: registry,
		ShardAndCommitteesAtSlots: []*pb.ShardAndCommitteeArray{
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{
						Shard:     0,
						Committee: []uint32{0, 1},
					},
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Slot:               0,
		RandaoRevealHash32: []byte{1},
		Body:               &pb.BeaconBlockBody{},
	}
	want := "could not verify and process block randao"
	if _, err := ProcessBlock(beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
	}
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, 64)
	shardAndCommittees[5] = &pb.ShardAndCommitteeArray{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S:  latestMixes,
		ValidatorRegistry:         registry,
		ShardAndCommitteesAtSlots: shardAndCommittees,
		Slot:                      5,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{1},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "could not verify block proposer slashing"
	if _, err := ProcessBlock(beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectCasperSlashing(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
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
	casperSlashings := make([]*pb.CasperSlashing, params.BeaconConfig().MaxCasperSlashings+1)
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, 64)
	shardAndCommittees[5] = &pb.ShardAndCommitteeArray{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S:  latestMixes,
		Slot:                      5,
		ValidatorRegistry:         registry,
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{1},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
			CasperSlashings:   casperSlashings,
		},
	}
	want := "could not verify block casper slashing"
	if _, err := ProcessBlock(beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessBlockAttestations(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
	}
	proposerSlashings := []*pb.ProposerSlashing{
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
	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	casperSlashings := []*pb.CasperSlashing{
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
	blockAttestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations+1)
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, 64)
	shardAndCommittees[5] = &pb.ShardAndCommitteeArray{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S:  latestMixes,
		Slot:                      5,
		ValidatorRegistry:         registry,
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{1},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			CasperSlashings:   casperSlashings,
			Attestations:      blockAttestations,
		},
	}
	want := "could not process block attestations"
	if _, err := ProcessBlock(beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessExits(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
	}
	proposerSlashings := []*pb.ProposerSlashing{
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
	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	casperSlashings := []*pb.CasperSlashing{
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
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.CrosslinkRecord{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	blockAtt := &pb.Attestation{
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
	attestations := []*pb.Attestation{blockAtt}
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, 128)
	shardAndCommittees[64] = &pb.ShardAndCommitteeArray{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S:  latestMixes,
		ValidatorRegistry:         registry,
		Slot:                      64,
		PreviousJustifiedSlot:     10,
		LatestBlockRootHash32S:    blockRoots,
		LatestCrosslinks:          stateLatestCrosslinks,
		ShardAndCommitteesAtSlots: shardAndCommittees,
	}
	exits := make([]*pb.Exit, params.BeaconConfig().MaxExits+1)
	block := &pb.BeaconBlock{
		Slot:               64,
		RandaoRevealHash32: []byte{1},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			CasperSlashings:   casperSlashings,
			Attestations:      attestations,
			Exits:             exits,
		},
	}
	want := "could not process validator exits"
	if _, err := ProcessBlock(beaconState, block); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			Status:                 pb.ValidatorRecord_ACTIVE,
			LatestStatusChangeSlot: 0,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
	}
	proposerSlashings := []*pb.ProposerSlashing{
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
	att1 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 5,
	}
	att2 := &pb.AttestationData{
		Slot:          5,
		JustifiedSlot: 4,
	}
	casperSlashings := []*pb.CasperSlashing{
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
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.CrosslinkRecord{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	blockAtt := &pb.Attestation{
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
	attestations := []*pb.Attestation{blockAtt}
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, 128)
	shardAndCommittees[64] = &pb.ShardAndCommitteeArray{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S:  latestMixes,
		ValidatorRegistry:         registry,
		Slot:                      64,
		ShardAndCommitteesAtSlots: shardAndCommittees,
		PreviousJustifiedSlot:     10,
		LatestBlockRootHash32S:    blockRoots,
		LatestCrosslinks:          stateLatestCrosslinks,
	}
	exits := []*pb.Exit{
		{
			ValidatorIndex: 0,
			Slot:           0,
		},
	}
	block := &pb.BeaconBlock{
		Slot:               64,
		RandaoRevealHash32: []byte{1},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			CasperSlashings:   casperSlashings,
			Attestations:      attestations,
			Exits:             exits,
		},
	}
	if _, err := ProcessBlock(beaconState, block); err != nil {
		t.Errorf("Expected block to pass processing conditions: %v", err)
	}
}

func TestIsNewValidatorSetTransition(t *testing.T) {
	beaconState, err := NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	beaconState.ValidatorRegistryLastChangeSlot = 1
	if IsValidatorSetChange(beaconState, 0) {
		t.Errorf("Is new validator set change should be false, last changed slot greater than finalized slot")
	}
	beaconState.FinalizedSlot = 2
	if IsValidatorSetChange(beaconState, 2) {
		t.Errorf("Is new validator set change should be false, MinValidatorSetChangeInterval has not reached")
	}
	shardCommitteeForSlots := []*pb.ShardAndCommitteeArray{{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0},
			{Shard: 1},
			{Shard: 2},
		},
	},
	}
	beaconState.ShardAndCommitteesAtSlots = shardCommitteeForSlots

	crosslinks := []*pb.CrosslinkRecord{
		{Slot: 1},
		{Slot: 1},
		{Slot: 1},
	}
	beaconState.LatestCrosslinks = crosslinks

	if IsValidatorSetChange(beaconState, params.BeaconConfig().MinValidatorSetChangeInterval+1) {
		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
	}

	crosslinks = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	beaconState.LatestCrosslinks = crosslinks

	if !IsValidatorSetChange(beaconState, params.BeaconConfig().MinValidatorSetChangeInterval+1) {
		t.Errorf("New validator set change failed should have been true")
	}
}
