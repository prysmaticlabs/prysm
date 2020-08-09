package powchain

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err, "Unable to setup web3 ETH1.0 chain service")

	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)

	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)

	err = web3Service.processDeposit(eth1Data, deposits[0])
	require.NoError(t, err, "could not process deposit")

	valcount, err := helpers.ActiveValidatorCount(web3Service.preGenesisState, 0)
	require.NoError(t, err)
	require.Equal(t, int(1), int(valcount), "Did not get correct active validator count")
}

func TestProcessDeposit_InvalidMerkleBranch(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)

	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)

	deposits[0].Proof = [][]byte{{'f', 'a', 'k', 'e'}}

	err = web3Service.processDeposit(eth1Data, deposits[0])
	require.NotNil(t, err, "No errors, when an error was expected")

	want := "deposit merkle branch of deposit root did not verify for root"

	assert.ErrorContains(t, want, err)
}

func TestProcessDeposit_InvalidPublicKey(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	deposits[0].Data.PublicKey = bytesutil.PadTo([]byte("junk"), 48)

	leaf, err := deposits[0].Data.HashTreeRoot()
	require.NoError(t, err, "Could not hash deposit")

	trie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)

	deposits[0].Proof, err = trie.MerkleProof(0)
	require.NoError(t, err)

	root := trie.Root()

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}

	err = web3Service.processDeposit(eth1Data, deposits[0])
	require.NoError(t, err)

	testutil.AssertLogsContain(t, hook, pubKeyErr)
}

func TestProcessDeposit_InvalidSignature(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)

	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	var fakeSig [96]byte
	copy(fakeSig[:], []byte{'F', 'A', 'K', 'E'})
	deposits[0].Data.Signature = fakeSig[:]

	leaf, err := deposits[0].Data.HashTreeRoot()
	require.NoError(t, err, "Could not hash deposit")

	trie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)

	root := trie.Root()

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}

	err = web3Service.processDeposit(eth1Data, deposits[0])
	require.NoError(t, err)

	testutil.AssertLogsContain(t, hook, pubKeyErr)
}

func TestProcessDeposit_UnableToVerify(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	testutil.ResetCache()

	deposits, keys, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	sig := keys[0].Sign([]byte{'F', 'A', 'K', 'E'})
	deposits[0].Data.Signature = sig.Marshal()[:]

	trie, _, err := testutil.DepositTrieFromDeposits(deposits)
	require.NoError(t, err)
	root := trie.Root()
	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}
	proof, err := trie.MerkleProof(0)
	require.NoError(t, err)
	deposits[0].Proof = proof
	err = web3Service.processDeposit(eth1Data, deposits[0])
	require.NoError(t, err)
	want := "signature did not verify"

	testutil.AssertLogsContain(t, hook, want)

}

func TestProcessDeposit_IncompleteDeposit(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)

	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			Amount:                params.BeaconConfig().EffectiveBalanceIncrement, // incomplete deposit
			WithdrawalCredentials: bytesutil.PadTo([]byte("testing"), 32),
			Signature:             bytesutil.PadTo([]byte("test"), 96),
		},
	}

	sk := bls.RandKey()
	deposit.Data.PublicKey = sk.PublicKey().Marshal()
	d, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	signedRoot, err := helpers.ComputeSigningRoot(deposit.Data, d)
	require.NoError(t, err)

	sig := sk.Sign(signedRoot[:])
	deposit.Data.Signature = sig.Marshal()

	trie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	root := trie.Root()
	eth1Data := &ethpb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}
	proof, err := trie.MerkleProof(0)
	require.NoError(t, err)
	dataRoot, err := deposit.Data.HashTreeRoot()
	require.NoError(t, err)
	deposit.Proof = proof

	factor := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().EffectiveBalanceIncrement
	// deposit till 31e9
	for i := 0; i < int(factor-1); i++ {
		trie.Insert(dataRoot[:], i)

		trieRoot := trie.HashTreeRoot()
		eth1Data.DepositRoot = trieRoot[:]
		eth1Data.DepositCount = uint64(i + 1)

		deposit.Proof, err = trie.MerkleProof(i)
		require.NoError(t, err)
		err = web3Service.processDeposit(eth1Data, deposit)
		require.NoError(t, err, fmt.Sprintf("Could not process deposit at %d", i))

		valcount, err := helpers.ActiveValidatorCount(web3Service.preGenesisState, 0)
		require.NoError(t, err)
		require.Equal(t, int(0), int(valcount), "Did not get correct active validator count")
	}
}

func TestProcessDeposit_AllDepositedSuccessfully(t *testing.T) {
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	testutil.ResetCache()

	deposits, keys, err := testutil.DeterministicDepositsAndKeys(10)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)

	for i := range keys {
		eth1Data.DepositCount = uint64(i + 1)
		err = web3Service.processDeposit(eth1Data, deposits[i])
		require.NoError(t, err, fmt.Sprintf("Could not process deposit at %d", i))

		valCount, err := helpers.ActiveValidatorCount(web3Service.preGenesisState, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(i+1), valCount, "Did not get correct active validator count")

		val, err := web3Service.preGenesisState.ValidatorAtIndex(uint64(i))
		require.NoError(t, err)
		assert.Equal(t, params.BeaconConfig().MaxEffectiveBalance, val.EffectiveBalance)
	}
}
