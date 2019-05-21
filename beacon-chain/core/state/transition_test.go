package state_test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	epoch := uint64(0)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.DomainVersion(beaconState, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal()
}

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
		Slot: 0,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
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
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	var attesterSlashings []*pb.AttesterSlashing
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings+1; i++ {
		attesterSlashings = append(attesterSlashings, &pb.AttesterSlashing{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: 0,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
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
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}

	var blockAttestations []*pb.Attestation
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations+1; i++ {
		blockAttestations = append(blockAttestations, &pb.Attestation{})
	}
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block := &pb.BeaconBlock{
		Slot: 0,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
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
	deposits := make([]*pb.Deposit, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: []byte(strconv.Itoa(i)),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRoots = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			TargetEpoch: 0,
			SourceEpoch: 0,
			SourceRoot:  []byte("tron-sucks"),
			Crosslink: &pb.Crosslink{
				Shard: 0,
				Epoch: 0,
			},
		},
		AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
		CustodyBitfield:     []byte{},
	}
	attestations := []*pb.Attestation{blockAtt}
	var exits []*pb.VoluntaryExit
	for i := uint64(0); i < params.BeaconConfig().MaxVoluntaryExits+1; i++ {
		exits = append(exits, &pb.VoluntaryExit{})
	}
	block := &pb.BeaconBlock{
		Slot: 4,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      []byte{},
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
			Epoch: 0,
		},
	}
	beaconState.CurrentJustifiedRoot = []byte("tron-sucks")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	encoded, err := ssz.TreeHash(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = params.BeaconConfig().ZeroHash[:]
	want := "could not process validator exits"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	deposits := make([]*pb.Deposit, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: []byte(strconv.Itoa(i)),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestSlashedBalances = make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Slot:        5,
					SourceEpoch: 0,
					TargetEpoch: 0,
					Shard:       4,
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.LatestBlockRoots = blockRoots
	beaconState.LatestCrosslinks = []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = (params.BeaconConfig().PersistentCommitteePeriod * slotsPerEpoch) + params.BeaconConfig().MinAttestationInclusionDelay
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			TargetEpoch: helpers.SlotToEpoch(beaconState.Slot),
			SourceEpoch: 0,
			SourceRoot:  []byte("tron-sucks"),
			Crosslink: &pb.Crosslink{
				Shard: 0,
				Epoch: helpers.SlotToEpoch(beaconState.Slot),
			},
		},
		AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
		CustodyBitfield:     []byte{},
	}
	attestations := []*pb.Attestation{blockAtt}
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 10,
			Epoch:          0,
		},
	}
	block := &pb.BeaconBlock{
		Slot: beaconState.Slot,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{2},
			BlockRoot:   []byte{3},
		},
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      []byte{},
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
		},
	}
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
			Epoch: helpers.SlotToEpoch(beaconState.Slot),
		},
	}
	beaconState.CurrentJustifiedRoot = []byte("tron-sucks")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	encoded, err := ssz.TreeHash(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = params.BeaconConfig().ZeroHash[:]
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); err != nil {
		t.Errorf("Expected block to pass processing conditions: %v", err)
	}
}

func TestProcessEpoch_PassesProcessingConditions(t *testing.T) {
	t.Skip()
	// TODO(#2307) unskip after ProcessCrosslinks is finished
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
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch,
				Shard:                    2,
				JustifiedEpoch:           1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().SlotsPerEpoch + 1,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestRandaoMixesLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	newState := &pb.BeaconState{
		Slot:               params.BeaconConfig().SlotsPerEpoch + 1,
		LatestAttestations: attestations,
		Balances:           validatorBalances,
		ValidatorRegistry:  validatorRegistry,
		LatestBlockRoots:   blockRoots,
		LatestCrosslinks:   crosslinkRecord,
		LatestRandaoMixes:  randaoHashes,
		LatestActiveIndexRoots: make([][]byte,
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
	t.Skip()
	// TODO(#2307) unskip after ProcessCrosslinks is finished
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
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch,
				Shard:                    2,
				JustifiedEpoch:           1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().SlotsPerEpoch + 1,
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
		Slot:               params.BeaconConfig().SlotsPerEpoch + 1,
		LatestAttestations: attestations,
		Balances:           validatorBalances,
		ValidatorRegistry:  validatorRegistry,
		LatestBlockRoots:   blockRoots,
		LatestCrosslinks:   crosslinkRecord,
		LatestRandaoMixes:  randaoHashes,
		LatestActiveIndexRoots: make([][]byte,
			params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances: make([]uint64,
			params.BeaconConfig().LatestSlashedExitLength),
		ValidatorRegistryUpdateEpoch: 0,
		FinalizedEpoch:               1,
	}

	newState, err := state.ProcessEpoch(context.Background(), newState, nil, state.DefaultConfig())
	if err != nil {
		t.Errorf("Expected epoch transition to pass processing conditions: %v", err)
	}
	if newState.ValidatorRegistryUpdateEpoch != 0 {
		t.Errorf(
			"Expected registry to not have been updated, received update epoch: %v",
			newState.ValidatorRegistryUpdateEpoch,
		)
	}
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func TestProcessEpoch_InactiveConditions(t *testing.T) {
	t.Skip()
	// TODO(#2307) unskip after ProcessCrosslinks is finished
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
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch,
				Shard:                    1,
				JustifiedEpoch:           1,
				JustifiedBlockRootHash32: []byte{0},
			},
			AggregationBitfield: []byte{},
			InclusionSlot:       i + params.BeaconConfig().SlotsPerEpoch + 1,
		})
	}

	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().SlotsPerEpoch; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestRandaoMixesLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	crosslinkRecord := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)

	newState := &pb.BeaconState{
		Slot:               params.BeaconConfig().SlotsPerEpoch + 1,
		LatestAttestations: attestations,
		Balances:           validatorBalances,
		ValidatorRegistry:  validatorRegistry,
		LatestBlockRoots:   blockRoots,
		LatestCrosslinks:   crosslinkRecord,
		LatestRandaoMixes:  randaoHashes,
		LatestActiveIndexRoots: make([][]byte,
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
	t.Skip()
	// TODO(#2307) unskip after ProcessCrosslinks is finished
	newState := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{Slot: 100}},
		}}

	want := fmt.Sprintf(
		"slot %d is not within expected range of %d to %d",
		newState.Slot,
		0,
		newState.Slot,
	)
	if _, err := state.ProcessEpoch(context.Background(), newState, &pb.BeaconBlock{}, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected: %s, received: %v", want, err)
	}
}

func TestProcessEpoch_CantGetCurrentValidatorIndices(t *testing.T) {
	t.Skip()
	// TODO(#2307) unskip after ProcessCrosslinks is finished
	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = params.BeaconConfig().ZeroHash[:]
	}

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     1,
				Shard:                    1,
				JustifiedBlockRootHash32: make([]byte, 32),
			},
			AggregationBitfield: []byte{0xff},
		})
	}

	newState := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch,
		LatestAttestations:     attestations,
		LatestBlockRoots:       latestBlockRoots,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	wanted := fmt.Sprintf("could not process justification and finalization of state: slot %d is not within expected range of %d to %d",
		64, 0, 64)
	if _, err := state.ProcessEpoch(context.Background(), newState, &pb.BeaconBlock{}, state.DefaultConfig()); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %v", wanted, err)
	}
}
