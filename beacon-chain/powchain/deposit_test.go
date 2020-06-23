package powchain

import (
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

const pubKeyErr = "could not convert bytes to public key"

func TestProcessDeposit_OK(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}

	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	if err := web3Service.processDeposit(eth1Data, deposits[0]); err != nil {
		t.Fatalf("Could not process deposit %v", err)
	}

	valcount, err := helpers.ActiveValidatorCount(web3Service.preGenesisState, 0)
	if err != nil {
		t.Fatal(err)
	}
	if valcount != 1 {
		t.Errorf("Did not get correct active validator count received %d, but wanted %d", valcount, 1)
	}
}

func TestProcessDeposit_InvalidMerkleBranch(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}

	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	deposits[0].Proof = [][]byte{{'f', 'a', 'k', 'e'}}

	err = web3Service.processDeposit(eth1Data, deposits[0])
	if err == nil {
		t.Fatal("No errors, when an error was expected")
	}

	want := "deposit merkle branch of deposit root did not verify for root"

	if !strings.Contains(err.Error(), want) {
		t.Errorf("Did not get expected error. Wanted: '%s' but got '%s'", want, err.Error())
	}

}

func TestProcessDeposit_InvalidPublicKey(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	deposits[0].Data.PublicKey = []byte("junk")

	leaf, err := ssz.HashTreeRoot(deposits[0].Data)
	if err != nil {
		t.Fatalf("Could not hash deposit %v", err)
	}
	trie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		log.Error(err)
	}
	deposits[0].Proof, err = trie.MerkleProof(0)
	if err != nil {
		t.Fatal(err)
	}

	root := trie.Root()

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}

	if err := web3Service.processDeposit(eth1Data, deposits[0]); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, pubKeyErr)
}

func TestProcessDeposit_InvalidSignature(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	var fakeSig [96]byte
	copy(fakeSig[:], []byte{'F', 'A', 'K', 'E'})
	deposits[0].Data.Signature = fakeSig[:]

	leaf, err := ssz.HashTreeRoot(deposits[0].Data)
	if err != nil {
		t.Fatalf("Could not hash deposit %v", err)
	}

	trie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		log.Error(err)
	}

	root := trie.Root()

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}

	if err := web3Service.processDeposit(eth1Data, deposits[0]); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, pubKeyErr)
}

func TestProcessDeposit_UnableToVerify(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	testutil.ResetCache()

	deposits, keys, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	sig := keys[0].Sign([]byte{'F', 'A', 'K', 'E'})
	deposits[0].Data.Signature = sig.Marshal()[:]

	trie, _, err := testutil.DepositTrieFromDeposits(deposits)
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}
	proof, err := trie.MerkleProof(0)
	if err != nil {
		t.Fatal(err)
	}
	deposits[0].Proof = proof
	if err := web3Service.processDeposit(eth1Data, deposits[0]); err != nil {
		t.Fatal(err)
	}
	want := "signature did not verify"

	testutil.AssertLogsContain(t, hook, want)

}

func TestProcessDeposit_IncompleteDeposit(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)

	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			Amount:                params.BeaconConfig().EffectiveBalanceIncrement, // incomplete deposit
			WithdrawalCredentials: []byte("testing"),
		},
	}

	sk := bls.RandKey()
	deposit.Data.PublicKey = sk.PublicKey().Marshal()
	d, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	signedRoot, err := helpers.ComputeSigningRoot(deposit.Data, d)
	if err != nil {
		t.Fatal(err)
	}

	sig := sk.Sign(signedRoot[:])
	deposit.Data.Signature = sig.Marshal()

	trie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}
	proof, err := trie.MerkleProof(0)
	if err != nil {
		t.Fatal(err)
	}
	dataRoot, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}
	deposit.Proof = proof

	factor := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().EffectiveBalanceIncrement
	// deposit till 31e9
	for i := 0; i < int(factor-1); i++ {
		trie.Insert(dataRoot[:], i)

		trieRoot := trie.HashTreeRoot()
		eth1Data.DepositRoot = trieRoot[:]
		eth1Data.DepositCount = uint64(i + 1)

		deposit.Proof, err = trie.MerkleProof(i)
		if err != nil {
			t.Fatal(err)
		}
		if err := web3Service.processDeposit(eth1Data, deposit); err != nil {
			t.Fatalf("Could not process deposit at %d %v", i, err)
		}

		valcount, err := helpers.ActiveValidatorCount(web3Service.preGenesisState, 0)
		if err != nil {
			t.Fatal(err)
		}

		if valcount == 1 {
			t.Errorf("Did not get correct active validator count received %d, but wanted %d", valcount, 0)
		}
	}
}

func TestProcessDeposit_AllDepositedSuccessfully(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	testutil.ResetCache()

	deposits, keys, err := testutil.DeterministicDepositsAndKeys(10)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	for i := range keys {
		eth1Data.DepositCount = uint64(i + 1)
		if err := web3Service.processDeposit(eth1Data, deposits[i]); err != nil {
			t.Fatalf("Could not process deposit %v", err)
		}

		valCount, err := helpers.ActiveValidatorCount(web3Service.preGenesisState, 0)
		if err != nil {
			t.Fatal(err)
		}

		if valCount != uint64(i+1) {
			t.Errorf("Did not get correct active validator count received %d, but wanted %d", valCount, uint64(i+1))
		}
		val, err := web3Service.preGenesisState.ValidatorAtIndex(uint64(i))
		if err != nil {
			t.Fatal(err)
		}
		if val.EffectiveBalance != params.BeaconConfig().MaxEffectiveBalance {
			t.Errorf(
				"Wanted a full deposit of %d but got %d",
				params.BeaconConfig().MaxEffectiveBalance,
				val.EffectiveBalance,
			)
		}
	}
}
