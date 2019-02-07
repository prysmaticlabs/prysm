package state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
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
	if _, err := ProcessBlock(beaconState, block, false); !strings.Contains(err.Error(), want) {
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
	if _, err := ProcessBlock(beaconState, block, false); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {
	registry := validators.InitialValidatorRegistry()

	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     5,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "could not verify block proposer slashing"
	if _, err := ProcessBlock(beaconState, block, false); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectAttesterSlashing(t *testing.T) {
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
	attesterSlashings := make([]*pb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings+1)
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        registry,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
			AttesterSlashings: attesterSlashings,
		},
	}
	want := "could not verify block attester slashing"
	if _, err := ProcessBlock(beaconState, block, false); !strings.Contains(err.Error(), want) {
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
	attesterSlashings := []*pb.AttesterSlashing{
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

	blockAttestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations+1)
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        registry,
	}
	block := &pb.BeaconBlock{
		Slot:               5,
		RandaoRevealHash32: []byte{},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      blockAttestations,
		},
	}
	want := "could not process block attestations"
	if _, err := ProcessBlock(beaconState, block, false); !strings.Contains(err.Error(), want) {
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
	attesterSlashings := []*pb.AttesterSlashing{
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
			JustifiedBlockRootHash32:  blockRoots[0],
			LatestCrosslinkRootHash32: []byte{1},
			ShardBlockRootHash32:      []byte{},
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     64,
		PreviousJustifiedEpoch:   0,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         stateLatestCrosslinks,
	}
	exits := make([]*pb.Exit, params.BeaconConfig().MaxExits+1)
	block := &pb.BeaconBlock{
		Slot:               64,
		RandaoRevealHash32: []byte{},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			Exits:             exits,
		},
	}
	want := "could not process validator exits"
	if _, err := ProcessBlock(beaconState, block, false); !strings.Contains(err.Error(), want) {
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
	attesterSlashings := []*pb.AttesterSlashing{
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
			JustifiedBlockRootHash32:  blockRoots[0],
			LatestCrosslinkRootHash32: []byte{1},
			ShardBlockRootHash32:      []byte{},
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        registry,
		Slot:                     64,
		PreviousJustifiedEpoch:   0,
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
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			Exits:             exits,
		},
	}
	if _, err := ProcessBlock(beaconState, block, false); err != nil {
		t.Errorf("Expected block to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_PassesProcessingConditions(t *testing.T) {
	var validatorRegistry []*pb.Validator
	for i := uint64(0); i < 10; i++ {
		validatorRegistry = append(validatorRegistry,
			&pb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			})
	}
	validatorBalances := make([]uint64, len(validatorRegistry))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDeposit
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().EpochLength,
				Shard:                    1,
				JustifiedSlot:            64,
				JustifiedBlockRootHash32: []byte{0},
			},
			SlotIncluded: i + params.BeaconConfig().EpochLength + 1,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := []*pb.CrosslinkRecord{{}, {}}

	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().EpochLength,
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
	defaultBalance := params.BeaconConfig().MaxDeposit

	validatorRegistry := []*pb.Validator{
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch}}

	validatorBalances := []uint64{
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().EpochLength,
				Shard:                    1,
				JustifiedSlot:            64,
				JustifiedBlockRootHash32: []byte{0},
			},
			AggregationBitfield: []byte{},
			SlotIncluded:        i + params.BeaconConfig().EpochLength + 1,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 5*params.BeaconConfig().EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := []*pb.CrosslinkRecord{{}, {}}

	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().EpochLength,
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
		Slot: 5,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Slot: 4}},
		}}

	want := fmt.Sprintf(
		"could not get current boundary attestations: slot %d is not within expected range of %d to %d",
		0, state.Slot, state.Slot-1,
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantGetCurrentValidatorIndices(t *testing.T) {
	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = params.BeaconConfig().ZeroHash[:]
	}

	var attestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                     1,
				Shard:                    1,
				JustifiedBlockRootHash32: make([]byte, 32),
			},
			AggregationBitfield: []byte{0xff},
		})
	}

	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().EpochLength,
		LatestAttestations:     attestations,
		LatestBlockRootHash32S: latestBlockRoots,
	}

	wanted := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 0, 1)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %v", wanted, err)
	}
}

func TestProcessEpoch_CantProcessCurrentBoundaryAttestations(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 100,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}},
		}}

	want := fmt.Sprintf(
		"could not get prev boundary attestations: slot %d is not within expected range of %d to %d",
		0, state.Slot, state.Slot-1,
	)
	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantProcessEjections(t *testing.T) {
	validatorRegistries := validators.InitialValidatorRegistry()
	validatorBalances := make([]uint64, len(validatorRegistries))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDeposit
	}
	var randaoHashes [][]byte
	for i := uint64(0); i < 4*params.BeaconConfig().EpochLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	ExitEpoch := 4*params.BeaconConfig().EpochLength + 1
	validatorRegistries[0].ExitEpoch = ExitEpoch
	validatorBalances[0] = params.BeaconConfig().EjectionBalance - 1
	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().EpochLength,
		ValidatorBalances:        validatorBalances,
		LatestBlockRootHash32S:   make([][]byte, params.BeaconConfig().LatestBlockRootsLength),
		ValidatorRegistry:        validatorRegistries,
		LatestRandaoMixesHash32S: randaoHashes,
		LatestCrosslinks:         []*pb.CrosslinkRecord{{}},
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{}, AggregationBitfield: participationBitfield},
		}}

	want := fmt.Sprintf("could not process inclusion distance: 0")

	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}
