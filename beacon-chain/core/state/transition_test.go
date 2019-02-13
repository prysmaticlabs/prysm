package state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
	validators := make([]*pb.Validator, 1000)
	for i := uint64(0); i < 1000; i++ {
		pubkey := hashutil.Hash([]byte{byte(i)})
		validators[i] = &pb.Validator{
			ExitEpoch:      params.BeaconConfig().FarFutureEpoch,
			Pubkey:         pubkey[:],
			PenalizedEpoch: 10,
		}
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
		Slot:           5,
		JustifiedEpoch: 5,
	}
	att2 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: 4,
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}

	blockAttestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations+1)
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		Slot:                     5,
		ValidatorRegistry:        validators,
		ValidatorBalances:        make([]uint64, len(validators)),
		LatestPenalizedBalances:  make([]uint64, params.BeaconConfig().LatestPenalizedExitLength),
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
	validators := make([]*pb.Validator, 1000)
	for i := uint64(0); i < 1000; i++ {
		pubkey := hashutil.Hash([]byte{byte(i)})
		validators[i] = &pb.Validator{
			ExitEpoch:      params.BeaconConfig().FarFutureEpoch,
			Pubkey:         pubkey[:],
			PenalizedEpoch: 10,
		}
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
		Slot:           5,
		JustifiedEpoch: 5,
	}
	att2 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: 4,
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.Crosslink{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     20,
			JustifiedBlockRootHash32: blockRoots[0],
			LatestCrosslink:          &pb.Crosslink{ShardBlockRootHash32: []byte{1}},
			ShardBlockRootHash32:     []byte{},
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        validators,
		Slot:                     64,
		PreviousJustifiedEpoch:   0,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         stateLatestCrosslinks,
		ValidatorBalances:        make([]uint64, len(validators)),
		LatestPenalizedBalances:  make([]uint64, params.BeaconConfig().LatestPenalizedExitLength),
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
	validators := make([]*pb.Validator, 1000)
	for i := uint64(0); i < 1000; i++ {
		pubkey := hashutil.Hash([]byte{byte(i)})
		validators[i] = &pb.Validator{
			ExitEpoch:      params.BeaconConfig().FarFutureEpoch,
			Pubkey:         pubkey[:],
			PenalizedEpoch: 10,
		}
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
		Slot:           5,
		JustifiedEpoch: 5,
	}
	att2 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: 4,
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().EpochLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.Crosslink{
		{
			ShardBlockRootHash32: []byte{1},
		},
	}
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     20,
			JustifiedBlockRootHash32: blockRoots[0],
			LatestCrosslink:          &pb.Crosslink{ShardBlockRootHash32: []byte{1}},
			ShardBlockRootHash32:     []byte{},
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	latestMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	beaconState := &pb.BeaconState{
		LatestRandaoMixesHash32S: latestMixes,
		ValidatorRegistry:        validators,
		Slot:                     64,
		PreviousJustifiedEpoch:   0,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         stateLatestCrosslinks,
		ValidatorBalances:        make([]uint64, len(validators)),
		LatestPenalizedBalances:  make([]uint64, params.BeaconConfig().LatestPenalizedExitLength),
	}
	exits := []*pb.Exit{
		{
			ValidatorIndex: 10,
			Epoch:          0,
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
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().EpochLength,
				Shard:                    1,
				JustifiedEpoch:           1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().EpochLength + 1,
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

	crosslinkRecord := []*pb.Crosslink{{}, {}}

	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().EpochLength,
		LatestAttestations:       attestations,
		ValidatorBalances:        validatorBalances,
		ValidatorRegistry:        validatorRegistry,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         crosslinkRecord,
		LatestRandaoMixesHash32S: randaoHashes,
		LatestIndexRootHash32S: make([][]byte,
			params.BeaconConfig().LatestIndexRootsLength),
		LatestPenalizedBalances: make([]uint64,
			params.BeaconConfig().LatestPenalizedExitLength),
	}

	_, err := ProcessEpoch(state)
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_InactiveConditions(t *testing.T) {
	defaultBalance := params.BeaconConfig().MaxDepositAmount

	validatorRegistry := []*pb.Validator{
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch}, {ExitEpoch: params.BeaconConfig().FarFutureEpoch}}

	validatorBalances := []uint64{
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
		defaultBalance, defaultBalance, defaultBalance, defaultBalance,
	}

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().EpochLength,
				Shard:                    1,
				JustifiedEpoch:           1,
				JustifiedBlockRootHash32: []byte{0},
			},
			AggregationBitfield: []byte{},
			InclusionSlot:       i + params.BeaconConfig().EpochLength + 1,
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

	crosslinkRecord := []*pb.Crosslink{{}, {}}

	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().EpochLength,
		LatestAttestations:       attestations,
		ValidatorBalances:        validatorBalances,
		ValidatorRegistry:        validatorRegistry,
		LatestBlockRootHash32S:   blockRoots,
		LatestCrosslinks:         crosslinkRecord,
		LatestRandaoMixesHash32S: randaoHashes,
		LatestIndexRootHash32S: make([][]byte,
			params.BeaconConfig().LatestIndexRootsLength),
		LatestPenalizedBalances: make([]uint64,
			params.BeaconConfig().LatestPenalizedExitLength),
	}

	_, err := ProcessEpoch(state)
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_CantGetBoundaryAttestation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5,
		LatestAttestations: []*pb.PendingAttestation{
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

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
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
		LatestAttestations: []*pb.PendingAttestation{
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
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
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
		LatestCrosslinks:         []*pb.Crosslink{{}},
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{}, AggregationBitfield: participationBitfield},
		}}

	want := fmt.Sprintf("could not process inclusion distance: 0")

	if _, err := ProcessEpoch(state); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}
