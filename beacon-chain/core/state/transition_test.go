package state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	validators := validators.InitialValidatorRegistry()

	beaconState := &pb.BeaconState{
		Slot:              0,
		ValidatorRegistry: validators,
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
	registry := validators.InitialValidatorRegistry()

	slashings := make([]*pb.ProposerSlashing, config.MaxProposerSlashings+1)
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     5,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{},
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
	registry := validators.InitialValidatorRegistry()

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
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        registry,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{},
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
	registry := validators.InitialValidatorRegistry()
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
			SlashableVote_1: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{0, 1, 2, 3},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableVote_2: &pb.SlashableVote{
				Data:             att2,
				ValidatorIndices: []uint64{4, 5, 6, 1},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}

	blockAttestations := make([]*pb.Attestation, config.MaxAttestations+1)
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        registry,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{},
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
	registry := validators.InitialValidatorRegistry()
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
			SlashableVote_1: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{0, 1, 2, 3},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableVote_2: &pb.SlashableVote{
				Data:             att2,
				ValidatorIndices: []uint64{4, 5, 6, 1},
				CustodyBitfield:  []byte{0xFF},
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
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     64,
		PreviousJustifiedSlot:    10,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         stateLatestCrosslinks,
	}
	exits := make([]*pb.Exit, config.MaxExits+1)
	block := &pb.BeaconBlock{
		Slot:               64,
		RandaoRevealHash32: []byte{},
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
	registry := validators.InitialValidatorRegistry()
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
			SlashableVote_1: &pb.SlashableVote{
				Data:             att1,
				ValidatorIndices: []uint64{0, 1, 2, 3},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableVote_2: &pb.SlashableVote{
				Data:             att2,
				ValidatorIndices: []uint64{4, 5, 6, 1},
				CustodyBitfield:  []byte{0xFF},
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
	latestMixes := make([][]byte, config.LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     64,
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
		RandaoRevealHash32: []byte{},
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
	validatorRegistry := validators.InitialValidatorRegistry()
	validatorBalances := make([]uint64, len(validatorRegistry))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = config.MaxDepositInGwei
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
	for i := uint64(0); i < config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := []*pb.CrosslinkRecord{{}, {}}

	state := &pb.BeaconState{
		Slot:                     config.EpochLength,
		LatestAttestations:       attestations,
		ValidatorBalances:        validatorBalances,
		ValidatorRegistry:        validatorRegistry,
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
		Slot:                     config.EpochLength,
		LatestAttestations:       attestations,
		ValidatorBalances:        validatorBalances,
		ValidatorRegistry:        validatorRegistry,
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
		LatestAttestations:     attestations,
		LatestBlockRootHash32S: latestBlockRoots,
	}

	//want := fmt.Sprintf(
	//	"could not get current boundary attester indices: wanted participants bitfield length %d, got: %d",
	//	len(shardCommittees[0].ArrayShardCommittee[0].Committee),
	//	len(attestations[0].ParticipationBitfield),
	//)
	want := "test"
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantGetPrevValidatorIndices(t *testing.T) {
	latestBlockRoots := make([][]byte, config.LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = config.ZeroHash[:]
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
		LatestAttestations:     attestations,
		LatestBlockRootHash32S: latestBlockRoots,
	}

	want := fmt.Sprintf(
		"input committee slot 1 out of bounds: %d <= slot < %d",
		config.EpochLength,
		config.EpochLength*3,
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
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
	validatorRegistries := validators.InitialValidatorRegistry()
	validatorBalances := make([]uint64, len(validatorRegistries))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = config.MaxDepositInGwei
	}
	var randaoHashes [][]byte
	for i := uint64(0); i < 4*config.EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}
	var participationBitfield []byte
	for i := 0; i < int(config.TargetCommitteeSize/8); i++ {
		participationBitfield = append(participationBitfield, byte(255))
	}
	exitSlot := 4*config.EpochLength + 1
	validatorRegistries[0].ExitSlot = exitSlot
	validatorBalances[0] = config.EjectionBalanceInGwei - 1
	state := &pb.BeaconState{
		Slot:                     config.EpochLength,
		ValidatorBalances:        validatorBalances,
		LatestBlockRootHash32S:   make([][]byte, config.LatestBlockRootsLength),
		ValidatorRegistry:        validatorRegistries,
		LatestRandaoMixesHash32S: randaoHashes,
		LatestCrosslinks:         []*pb.CrosslinkRecord{{}},
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}, ParticipationBitfield: participationBitfield},
		}}

	want := fmt.Sprintf(
		"validator 0 could not exit until slot %d", 320)

	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}
