package state_test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func setupInitialDeposits(t *testing.T, numDeposits uint64) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		depositInput := &pb.DepositInput{
			Pubkey: priv.PublicKey().Marshal(),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func createRandaoReveal(t *testing.T, beaconState *pb.BeaconState, privKeys []*bls.SecretKey) []byte {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	epoch := helpers.SlotToEpoch(params.BeaconConfig().GenesisSlot)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal()
}

func TestProcessBlock_IncorrectSlot(t *testing.T) {
	beaconState := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 5,
	}
	block := &pb.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot + 4,
	}
	want := fmt.Sprintf(
		"block.slot != state.slot, block.slot = %d, state.slot = %d",
		4,
		5,
	)
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	var slashings []*pb.ProposerSlashing
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings+1; i++ {
		slashings = append(slashings, &pb.ProposerSlashing{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot:         params.BeaconConfig().GenesisSlot,
		RandaoReveal: randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "could not verify block proposer slashing"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectAttesterSlashing(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
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
	var attesterSlashings []*pb.AttesterSlashing
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings+1; i++ {
		attesterSlashings = append(attesterSlashings, &pb.AttesterSlashing{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot:         params.BeaconConfig().GenesisSlot,
		RandaoReveal: randaoReveal,
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
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessBlockAttestations(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
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
		Slot:           params.BeaconConfig().GenesisSlot + 5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 5,
	}
	att2 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 4,
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

	var blockAttestations []*pb.Attestation
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations+1; i++ {
		blockAttestations = append(blockAttestations, &pb.Attestation{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot:         params.BeaconConfig().GenesisSlot,
		RandaoReveal: randaoReveal,
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
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessExits(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            params.BeaconConfig().GenesisSlot + 1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            params.BeaconConfig().GenesisSlot + 1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
		},
	}
	att1 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 5,
	}
	att2 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 4,
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
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRootHash32S = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + 10
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     params.BeaconConfig().GenesisSlot,
			JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
			LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
			CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	var exits []*pb.VoluntaryExit
	for i := uint64(0); i < params.BeaconConfig().MaxVoluntaryExits+1; i++ {
		exits = append(exits, &pb.VoluntaryExit{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot:         params.BeaconConfig().GenesisSlot + 10,
		RandaoReveal: randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}
	want := "could not process validator exits"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, params.BeaconConfig().SlotsPerEpoch)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
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
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 5,
	}
	att2 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 4,
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
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRootHash32S = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + 10
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     params.BeaconConfig().GenesisSlot,
			JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
			LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
			CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{blockAtt}
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 10,
			Epoch:          params.BeaconConfig().GenesisEpoch,
		},
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot:         params.BeaconConfig().GenesisSlot + 10,
		RandaoReveal: randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); err != nil {
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
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
				Shard:                    1,
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch + 1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().SlotsPerEpoch + 1 + params.BeaconConfig().GenesisSlot,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := make([]*pb.Crosslink, 64)
	newState := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1,
		LatestAttestations:     attestations,
		ValidatorBalances:      validatorBalances,
		ValidatorRegistry:      validatorRegistry,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       crosslinkRecord,
		LatestRandaoMixes:      randaoHashes,
		LatestIndexRootHash32S: make([][]byte,
			params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances: make([]uint64,
			params.BeaconConfig().LatestSlashedExitLength),
	}

	_, err := state.ProcessEpoch(context.Background(), newState, &pb.BeaconBlock{}, state.DefaultConfig())
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_PreventsRegistryUpdateOnNilBlock(t *testing.T) {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: false,
	})
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
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
				Shard:                    1,
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch + 1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().SlotsPerEpoch + 1 + params.BeaconConfig().GenesisSlot,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := make([]*pb.Crosslink, 64)
	newState := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1,
		LatestAttestations:     attestations,
		ValidatorBalances:      validatorBalances,
		ValidatorRegistry:      validatorRegistry,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       crosslinkRecord,
		LatestRandaoMixes:      randaoHashes,
		LatestIndexRootHash32S: make([][]byte,
			params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances: make([]uint64,
			params.BeaconConfig().LatestSlashedExitLength),
		ValidatorRegistryUpdateEpoch: params.BeaconConfig().GenesisEpoch,
		FinalizedEpoch:               params.BeaconConfig().GenesisEpoch + 1,
	}

	newState, err := state.ProcessEpoch(context.Background(), newState, nil, state.DefaultConfig())
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
	if newState.ValidatorRegistryUpdateEpoch != params.BeaconConfig().GenesisEpoch {
		t.Errorf(
			"Expected registry to not have been updated, received update epoch: %v",
			newState.ValidatorRegistryUpdateEpoch-params.BeaconConfig().GenesisEpoch,
		)
	}
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
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
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
				Shard:                    1,
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch + 1,
				JustifiedBlockRootHash32: []byte{0},
			},
			AggregationBitfield: []byte{},
			InclusionSlot:       i + params.BeaconConfig().SlotsPerEpoch + 1 + params.BeaconConfig().GenesisSlot,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().SlotsPerEpoch; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < 5*params.BeaconConfig().SlotsPerEpoch; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := make([]*pb.Crosslink, 64)

	newState := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1,
		LatestAttestations:     attestations,
		ValidatorBalances:      validatorBalances,
		ValidatorRegistry:      validatorRegistry,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       crosslinkRecord,
		LatestRandaoMixes:      randaoHashes,
		LatestIndexRootHash32S: make([][]byte,
			params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances: make([]uint64,
			params.BeaconConfig().LatestSlashedExitLength),
	}

	_, err := state.ProcessEpoch(context.Background(), newState, &pb.BeaconBlock{}, state.DefaultConfig())
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_CantGetBoundaryAttestation(t *testing.T) {
	newState := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot + 100}},
		}}

	want := fmt.Sprintf(
		"slot %d is not within expected range of %d to %d",
		newState.Slot-params.BeaconConfig().GenesisSlot,
		0,
		newState.Slot-params.BeaconConfig().GenesisSlot,
	)
	if _, err := state.ProcessEpoch(context.Background(), newState, &pb.BeaconBlock{}, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantGetCurrentValidatorIndices(t *testing.T) {
	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = params.BeaconConfig().ZeroHash[:]
	}

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     params.BeaconConfig().GenesisSlot + 1,
				Shard:                    1,
				JustifiedBlockRootHash32: make([]byte, 32),
			},
			AggregationBitfield: []byte{0xff},
		})
	}

	newState := &pb.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
		LatestAttestations:     attestations,
		LatestBlockRootHash32S: latestBlockRoots,
	}

	wanted := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 0, 1)
	if _, err := state.ProcessEpoch(context.Background(), newState, &pb.BeaconBlock{}, state.DefaultConfig()); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %v", wanted, err)
	}
}
