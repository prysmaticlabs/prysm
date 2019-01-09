package state

import (
	"fmt"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var config = params.BeaconConfig()

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
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{0},
			RandaoLayers:           0,
		},
		{
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{0},
			RandaoLayers:           0,
		},
	}
	beaconState := &pb.BeaconState{
		Slot:              0,
		ValidatorRegistry: registry,
		ShardCommitteesAtSlots: []*pb.ShardCommitteeArray{
			{
				ArrayShardCommittee: []*pb.ShardCommittee{
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
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
	}

	slashings := make([]*pb.ProposerSlashing, config.MaxProposerSlashings+1)
	shardCommittees := make([]*pb.ShardCommitteeArray, 64)
	shardCommittees[5] = &pb.ShardCommitteeArray{
		ArrayShardCommittee: []*pb.ShardCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		ShardCommitteesAtSlots:   shardCommittees,
		Slot:                     5,
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
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			ExitSlot:               config.FarFutureSlot,
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
	casperSlashings := make([]*pb.CasperSlashing, config.MaxCasperSlashings+1)
	shardCommittees := make([]*pb.ShardCommitteeArray, 64)
	shardCommittees[5] = &pb.ShardCommitteeArray{
		ArrayShardCommittee: []*pb.ShardCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        registry,
		ShardCommitteesAtSlots:   shardCommittees,
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
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			ExitSlot:               config.FarFutureSlot,
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
				Data:                att1,
				CustodyBit_0Indices: []uint32{0, 1},
				CustodyBit_1Indices: []uint32{2, 3},
			},
			Votes_2: &pb.SlashableVoteData{
				Data:                att2,
				CustodyBit_0Indices: []uint32{4, 5},
				CustodyBit_1Indices: []uint32{6, 1},
			},
		},
	}

	blockAttestations := make([]*pb.Attestation, config.MaxAttestations+1)
	shardCommittees := make([]*pb.ShardCommitteeArray, 64)
	shardCommittees[5] = &pb.ShardCommitteeArray{
		ArrayShardCommittee: []*pb.ShardCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        registry,
		ShardCommitteesAtSlots:   shardCommittees,
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
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			ExitSlot:               config.FarFutureSlot,
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
				Data:                att1,
				CustodyBit_0Indices: []uint32{0, 1},
				CustodyBit_1Indices: []uint32{2, 3},
			},
			Votes_2: &pb.SlashableVoteData{
				Data:                att2,
				CustodyBit_0Indices: []uint32{4, 5},
				CustodyBit_1Indices: []uint32{6, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
	ShardCommittees := make([]*pb.ShardCommitteeArray, 128)
	ShardCommittees[64] = &pb.ShardCommitteeArray{
		ArrayShardCommittee: []*pb.ShardCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     64,
		PreviousJustifiedSlot:    10,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         stateLatestCrosslinks,
		ShardCommitteesAtSlots:   ShardCommittees,
	}
	exits := make([]*pb.Exit, config.MaxExits+1)
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
			ExitSlot:               config.FarFutureSlot,
			RandaoCommitmentHash32: []byte{1},
			RandaoLayers:           0,
		},
		{
			ExitSlot:               config.FarFutureSlot,
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
				Data:                att1,
				CustodyBit_0Indices: []uint32{0, 1},
				CustodyBit_1Indices: []uint32{2, 3},
			},
			Votes_2: &pb.SlashableVoteData{
				Data:                att2,
				CustodyBit_0Indices: []uint32{4, 5},
				CustodyBit_1Indices: []uint32{6, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
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
	ShardCommittees := make([]*pb.ShardCommitteeArray, 128)
	ShardCommittees[64] = &pb.ShardCommitteeArray{
		ArrayShardCommittee: []*pb.ShardCommittee{
			{
				Shard:     0,
				Committee: []uint32{0, 1},
			},
		},
	}
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     64,
		ShardCommitteesAtSlots:   ShardCommittees,
		PreviousJustifiedSlot:    10,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         stateLatestCrosslinks,
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

func TestProcessEpoch_PassesProcessingConditions(t *testing.T) {
	defaultBalance := config.MaxDepositInGwei

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
			},
		})
	}

	validatorRegistry := []*pb.ValidatorRecord{
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot},
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot},
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot},
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot}}

	validatorBalances := []uint64{
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < config.EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     i + config.EpochLength,
				Shard:                    1,
				JustifiedSlot:            64,
				JustifiedBlockRootHash32: []byte{0},
			},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          i + config.EpochLength + 1,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 16*config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := []*pb.CrosslinkRecord{{}, {}}

	state := &pb.BeaconState{
		Slot:                     config.EpochLength * 16,
		LatestAttestations:       attestations,
		ValidatorBalances:        validatorBalances,
		ValidatorRegistry:        validatorRegistry,
		ShardCommitteesAtSlots:   shardCommittees,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         crosslinkRecord,
		LatestRandaoMixesHash32S: randaoHashes,
	}

	_, err := ProcessEpoch(state)
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_InactiveConditions(t *testing.T) {
	defaultBalance := config.MaxDepositInGwei

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
			},
		})
	}

	validatorRegistry := []*pb.ValidatorRecord{
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot},
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot},
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot},
		{ExitSlot: config.FarFutureSlot}, {ExitSlot: config.FarFutureSlot}}

	validatorBalances := []uint64{
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < config.EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     i + config.EpochLength,
				Shard:                    1,
				JustifiedSlot:            64,
				JustifiedBlockRootHash32: []byte{0},
			},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          i + config.EpochLength + 1,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < 2*config.EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 5*config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := []*pb.CrosslinkRecord{{}, {}}

	state := &pb.BeaconState{
		Slot:                     config.EpochLength * 5,
		LatestAttestations:       attestations,
		ValidatorBalances:        validatorBalances,
		ValidatorRegistry:        validatorRegistry,
		ShardCommitteesAtSlots:   shardCommittees,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         crosslinkRecord,
		LatestRandaoMixesHash32S: randaoHashes,
	}

	_, err := ProcessEpoch(state)
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_CantGetBoundaryAttestation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}},
		}}

	want := fmt.Sprintf(
		"could not get current boundary attestations: slot %d out of bounds: %d <= slot < %d",
		state.LatestAttestations[0].Data.Slot, state.Slot, state.Slot,
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantGetCurrentValidatorIndices(t *testing.T) {
	latestBlockRoots := make([][]byte, config.LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = config.ZeroHash[:]
	}

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{}},
			},
		})
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < config.EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     1,
				Shard:                    1,
				JustifiedBlockRootHash32: make([]byte, 32),
			},
			ParticipationBitfield: []byte{0xff},
		})
	}

	state := &pb.BeaconState{
		Slot:                   config.EpochLength,
		ShardCommitteesAtSlots: shardCommittees,
		LatestAttestations:     attestations,
		LatestBlockRootHash32S: latestBlockRoots,
	}

	want := fmt.Sprintf(
		"could not get current boundary attester indices: wanted participants bitfield length %d, got: %d",
		len(shardCommittees[0].ArrayShardCommittee[0].Committee),
		len(attestations[0].ParticipationBitfield),
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantGetPrevValidatorIndices(t *testing.T) {

	latestBlockRoots := make([][]byte, config.LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = config.ZeroHash[:]
	}

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{}},
			},
		})
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < config.EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     1,
				Shard:                    1,
				JustifiedBlockRootHash32: make([]byte, 32),
			},
			ParticipationBitfield: []byte{0xff},
		})
	}

	state := &pb.BeaconState{
		Slot:                   config.EpochLength * 2,
		ShardCommitteesAtSlots: shardCommittees,
		LatestAttestations:     attestations,
		LatestBlockRootHash32S: latestBlockRoots,
	}

	want := fmt.Sprintf(
		"could not get prev epoch attester indices: slot 1 out of bounds: %d <= slot < %d",
		config.EpochLength,
		config.EpochLength*3,
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Log(err)
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantProcessPrevBoundaryAttestations(t *testing.T) {
	state := &pb.BeaconState{
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}},
		}}

	want := fmt.Sprintf(
		"could not get prev boundary attestations: slot %d out of bounds: %d <= slot < %d",
		state.LatestAttestations[0].Data.Slot, state.Slot, state.Slot,
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantProcessEjections(t *testing.T) {

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{}},
			},
		})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 4*config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		Slot:                     4 * config.EpochLength,
		ValidatorBalances:        []uint64{1e9},
		ShardCommitteesAtSlots:   shardCommittees,
		LatestBlockRootHash32S:   make([][]byte, config.LatestBlockRootsLength),
		ValidatorRegistry:        []*pb.ValidatorRecord{{ExitSlot: 4*config.EpochLength + 1}},
		LatestRandaoMixesHash32S: randaoHashes,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}, ParticipationBitfield: []byte{}},
		}}

	want := fmt.Sprintf(
		"could not process ejections: could not exit validator 0: "+
			"validator 0 could not exit until slot %d", state.Slot+config.EntryExitDelay)

	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantProcessValidators(t *testing.T) {
	defaultBalance := config.MaxDepositInGwei

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Committee: []uint32{}},
			},
		})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 4*config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	size := 1<<(params.BeaconConfig().RandBytes*8) - 1
	validators := make([]*pb.ValidatorRecord, size)
	validatorBalances := make([]uint64, size)
	validator := &pb.ValidatorRecord{ExitSlot: params.BeaconConfig().FarFutureSlot}
	for i := 0; i < size; i++ {
		validators[i] = validator
		validatorBalances[i] = defaultBalance
	}

	state := &pb.BeaconState{
		Slot:                     4 * config.EpochLength,
		ValidatorBalances:        validatorBalances,
		ShardCommitteesAtSlots:   shardCommittees,
		LatestBlockRootHash32S:   make([][]byte, config.LatestBlockRootsLength),
		ValidatorRegistry:        validators,
		LatestRandaoMixesHash32S: randaoHashes,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}, ParticipationBitfield: []byte{}}},
		FinalizedSlot:    1,
		LatestCrosslinks: []*pb.CrosslinkRecord{{Slot: 1}},
	}

	want := fmt.Sprint(
		"could not shuffle validator registry for commtitees: input list exceeded upper bound and reached modulo bias",
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantProcessPartialValidators(t *testing.T) {
	defaultBalance := config.MaxDepositInGwei

	var shardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < config.EpochLength*2; i++ {
		shardCommittees = append(shardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 1, Committee: []uint32{}},
			},
		})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 4*config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	size := 1<<(params.BeaconConfig().RandBytes*8) - 1
	validators := make([]*pb.ValidatorRecord, size)
	validatorBalances := make([]uint64, size)
	validator := &pb.ValidatorRecord{ExitSlot: params.BeaconConfig().FarFutureSlot}
	for i := 0; i < size; i++ {
		validators[i] = validator
		validatorBalances[i] = defaultBalance
	}

	state := &pb.BeaconState{
		Slot:                     4 * config.EpochLength,
		ValidatorBalances:        validatorBalances,
		ShardCommitteesAtSlots:   shardCommittees,
		LatestBlockRootHash32S:   make([][]byte, config.LatestBlockRootsLength),
		ValidatorRegistry:        validators,
		LatestRandaoMixesHash32S: randaoHashes,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}, ParticipationBitfield: []byte{}},
		}}

	want := fmt.Sprint(
		"could not shuffle validator registry for commtitees: input list exceeded upper bound and reached modulo bias",
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}
