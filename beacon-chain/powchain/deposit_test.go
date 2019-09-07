package powchain

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

const pubKeyErr = "could not deserialize validator public key"

func TestProcessDeposit_OK(t *testing.T) {
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 1)

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
		t.Fatalf("Could not process deposit %v", err)
	}

	if web3Service.activeValidatorCount != 1 {
		t.Errorf("Did not get correct active validator count received %d, but wanted %d", web3Service.activeValidatorCount, 1)
	}
}

func TestProcessDeposit_InvalidMerkleBranch(t *testing.T) {
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 1)

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
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 1)
	deposits[0].Data.PublicKey = []byte("junk")

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

	err = web3Service.processDeposit(eth1Data, deposits[0])
	if err == nil {
		t.Fatal("No errors, when an error was expected")
	}

	if !strings.Contains(err.Error(), pubKeyErr) {
		t.Errorf("Did not get expected error. Wanted: '%s' but got '%s'", pubKeyErr, err.Error())
	}

}

func TestProcessDeposit_InvalidSignature(t *testing.T) {
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 1)
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

	err = web3Service.processDeposit(eth1Data, deposits[0])
	if err == nil {
		t.Fatal("No errors, when an error was expected")
	}

	if !strings.Contains(err.Error(), pubKeyErr) {
		t.Errorf("Did not get expected error. Wanted: '%s' but got '%s'", pubKeyErr, err.Error())
	}

}

func TestProcessDeposit_UnableToVerify(t *testing.T) {
	helpers.ClearAllCaches()
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	testutil.ResetCache()

	deposits, keys := testutil.SetupInitialDeposits(t, 1)
	sig := keys[0].Sign([]byte{'F', 'A', 'K', 'E'}, bls.Domain(params.BeaconConfig().DomainDeposit, params.BeaconConfig().GenesisForkVersion))
	deposits[0].Data.Signature = sig.Marshal()[:]
	eth1Data := testutil.GenerateEth1Data(t, deposits)

	err = web3Service.processDeposit(eth1Data, deposits[0])
	if err == nil {
		t.Fatal("No errors, when an error was expected")
	}

	want := "deposit signature did not verify"

	if !strings.Contains(err.Error(), want) {
		t.Errorf("Did not get expected error. Wanted: '%s' but got '%s'", want, err.Error())
	}

}

func TestProcessDeposit_IncompleteDeposit(t *testing.T) {
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}

	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			Amount:                params.BeaconConfig().EffectiveBalanceIncrement, // incomplete deposit
			WithdrawalCredentials: []byte("testing"),
		},
	}

	sk, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	deposit.Data.PublicKey = sk.PublicKey().Marshal()
	signedRoot, err := ssz.SigningRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}

	sig := sk.Sign(signedRoot[:], bls.Domain(params.BeaconConfig().DomainDeposit, params.BeaconConfig().GenesisForkVersion))
	deposit.Data.Signature = sig.Marshal()

	_, root := testutil.GenerateDepositProof(t, []*ethpb.Deposit{deposit})

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}

	factor := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().EffectiveBalanceIncrement
	// deposit till 31e9
	for i := 0; i < int(factor-1); i++ {
		if err := web3Service.processDeposit(eth1Data, deposit); err != nil {
			t.Fatalf("Could not process deposit %v", err)
		}

		if web3Service.activeValidatorCount == 1 {
			t.Errorf("Did not get correct active validator count received %d, but wanted %d", web3Service.activeValidatorCount, 0)
		}
	}
}

func TestProcessDeposit_AllDepositedSuccessfully(t *testing.T) {
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &kv.Store{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}
	testutil.ResetCache()

	deposits, keys := testutil.SetupInitialDeposits(t, 10)
	deposits, root := testutil.GenerateDepositProof(t, deposits)

	eth1Data := &ethpb.Eth1Data{
		DepositCount: uint64(len(deposits)),
		DepositRoot:  root[:],
	}

	for i, k := range keys {
		eth1Data.DepositCount = uint64(i + 1)
		if err := web3Service.processDeposit(eth1Data, deposits[i]); err != nil {
			t.Fatalf("Could not process deposit %v", err)
		}

		if web3Service.activeValidatorCount != uint64(i+1) {
			t.Errorf("Did not get correct active validator count received %d, but wanted %d", web3Service.activeValidatorCount, uint64(i+1))
		}
		pubkey := bytesutil.ToBytes48(k.PublicKey().Marshal())
		if web3Service.depositedPubkeys[pubkey] != params.BeaconConfig().MaxEffectiveBalance {
			t.Errorf("Wanted a full deposit of %d but got %d", params.BeaconConfig().MaxEffectiveBalance, web3Service.depositedPubkeys[pubkey])
		}
	}
}
