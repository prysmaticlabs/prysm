package blocks_test

import (
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestProcessDeposits_SameValidatorMultipleDepositsSameBlock(t *testing.T) {
	// Same validator created 3 valid deposits within the same block
	testutil.ResetCache()
	dep, _, err := testutil.DeterministicDepositsAndKeysSameValidator(3)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(dep))
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			// 3 deposits from the same validator
			Deposits: []*ethpb.Deposit{dep[0], dep[1], dep[2]},
		},
	}
	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}

	if len(newState.Validators()) != 2 {
		t.Errorf("Incorrect validator count. Wanted %d, got %d", 2, len(newState.Validators()))
	}
}

func TestProcessDeposits_MerkleBranchFailsVerification(t *testing.T) {
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: []byte{1, 2, 3},
			Signature: make([]byte, 96),
		},
	}
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit.Proof = proof
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{deposit},
		},
	}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: []byte{0},
			BlockHash:   []byte{1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "deposit root did not verify"
	_, err = blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessDeposits_AddsNewValidatorDeposit(t *testing.T) {
	dep, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(dep))
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{dep[0]},
		},
	}
	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}
	if newState.Balances()[1] != dep[0].Data.Amount {
		t.Errorf(
			"Expected state validator balances index 0 to equal %d, received %d",
			dep[0].Data.Amount,
			newState.Balances()[1],
		)
	}
}

func TestProcessDeposits_RepeatedDeposit_IncreasesValidatorBalance(t *testing.T) {
	sk := bls.RandKey()
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: sk.PublicKey().Marshal(),
			Amount:    1000,
		},
	}
	sr, err := helpers.ComputeSigningRoot(deposit.Data, bytesutil.ToBytes(3, 8))
	if err != nil {
		t.Fatal(err)
	}
	sig := sk.Sign(sr[:])
	deposit.Data.Signature = sig.Marshal()
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit.Proof = proof
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{deposit},
		},
	}
	registry := []*ethpb.Validator{
		{
			PublicKey: []byte{1, 2, 3},
		},
		{
			PublicKey:             sk.PublicKey().Marshal(),
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{0, 50}
	root := depositTrie.Root()
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: root[:],
			BlockHash:   root[:],
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.Balances()[1] != 1000+50 {
		t.Errorf("Expected balance at index 1 to be 1050, received %d", newState.Balances()[1])
	}
}

func TestProcessDeposit_AddsNewValidatorDeposit(t *testing.T) {
	//Similar to TestProcessDeposits_AddsNewValidatorDeposit except that this test directly calls ProcessDeposit
	dep, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(dep))
	if err != nil {
		t.Fatal(err)
	}

	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Error(err)
	}
	newState, err := blocks.ProcessDeposit(beaconState, dep[0], true)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.Validators()) != 2 {
		t.Errorf("Expected validator list to have length 2, received: %v", len(newState.Validators()))
	}
	if len(newState.Balances()) != 2 {
		t.Fatalf("Expected validator balances list to have length 2, received: %v", len(newState.Balances()))
	}
	if newState.Balances()[1] != dep[0].Data.Amount {
		t.Errorf(
			"Expected state validator balances index 1 to equal %d, received %d",
			dep[0].Data.Amount,
			newState.Balances()[1],
		)
	}
}

func TestProcessDeposit_SkipsInvalidDeposit(t *testing.T) {
	// Same test settings as in TestProcessDeposit_AddsNewValidatorDeposit, except that we use an invalid signature
	dep, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	dep[0].Data.Signature = make([]byte, 96)
	trie, _, err := testutil.DepositTrieFromDeposits(dep)
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	eth1Data := &ethpb.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: 1,
	}
	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposit(beaconState, dep[0], true)
	if err != nil {
		t.Fatalf("Expected invalid block deposit to be ignored without error, received: %v", err)
	}

	if newState.Eth1DepositIndex() != 1 {
		t.Errorf(
			"Expected Eth1DepositIndex to be increased by 1 after processing an invalid deposit, received change: %v",
			newState.Eth1DepositIndex(),
		)
	}
	if len(newState.Validators()) != 1 {
		t.Errorf("Expected validator list to have length 1, received: %v", len(newState.Validators()))
	}
	if len(newState.Balances()) != 1 {
		t.Errorf("Expected validator balances list to have length 1, received: %v", len(newState.Balances()))
	}
	if newState.Balances()[0] != 0 {
		t.Errorf("Expected validator balance at index 0 to stay 0, received: %v", newState.Balances()[0])
	}
}
