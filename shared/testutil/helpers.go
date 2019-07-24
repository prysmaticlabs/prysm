package testutil

import (
	"crypto/rand"
	"sync"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var lock sync.Mutex

// Caches
var deposits []*ethpb.Deposit
var privKeys []*bls.SecretKey
var trie *trieutil.MerkleTrie

// SetupInitialDeposits prepares the entered amount of deposits
// and secret keys.
func SetupInitialDeposits(t testing.TB, numDeposits uint64) ([]*ethpb.Deposit, []*bls.SecretKey) {
	lock.Lock()
	defer lock.Unlock()

	var err error

	// Populate trie cache, if not initialized yet.
	if trie == nil {
		trie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Extend caches as needed.
	for i := len(deposits); uint64(len(deposits)) < numDeposits; i++ {
		var withdrawalCreds [32]byte
		copy(withdrawalCreds[:], []byte("testing"))
		depositData := &ethpb.Deposit_Data{
			Amount:                params.BeaconConfig().MaxEffectiveBalance,
			WithdrawalCredentials: withdrawalCreds[:],
		}
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatalf("could not generate random key: %v", err)
		}
		privKeys = append(privKeys, priv)
		depositData.PublicKey = priv.PublicKey().Marshal()[:]
		domain := bls.Domain(params.BeaconConfig().DomainDeposit, params.BeaconConfig().GenesisForkVersion)
		root, err := ssz.SigningRoot(depositData)
		if err != nil {
			t.Fatalf("could not get signing root of deposit data %v", err)
		}
		depositData.Signature = priv.Sign(root[:], domain).Marshal()
		deposit := &ethpb.Deposit{
			Data: depositData,
		}

		deposits = append(deposits, deposit)
	}

	d, _ := GenerateDepositProof(t, deposits[0:numDeposits])
	return d, privKeys[0:numDeposits]
}

// GenerateDepositProof takes an array of deposits and generates the deposit trie for them and proofs.
func GenerateDepositProof(t testing.TB, deposits []*ethpb.Deposit) ([]*ethpb.Deposit, [32]byte) {
	encodedDeposits := make([][]byte, len(deposits))
	for i := 0; i < len(encodedDeposits); i++ {
		hashedDeposit, err := hashutil.DepositHash(deposits[i].Data)
		if err != nil {
			t.Fatalf("could not tree hash deposit data: %v", err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(encodedDeposits, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate deposit trie: %v", err)
	}

	for i := range deposits {
		proof, err := depositTrie.MerkleProof(int(i))
		if err != nil {
			t.Fatalf("Could not generate proof: %v", err)
		}
		deposits[i].Proof = proof
	}
	root := depositTrie.Root()
	return deposits, root
}

// GenerateEth1Data takes an array of deposits and generates the deposit trie for them.
func GenerateEth1Data(t testing.TB, deposits []*ethpb.Deposit) *ethpb.Eth1Data {
	_, root := GenerateDepositProof(t, deposits)
	eth1Data := &ethpb.Eth1Data{
		BlockHash:    root[:],
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
	}

	return eth1Data
}

// ResetCache clears out the old trie, private keys and deposits.
func ResetCache() {
	trie = nil
	privKeys = []*bls.SecretKey{}
	deposits = []*ethpb.Deposit{}
}
