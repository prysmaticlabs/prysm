package testutil

import (
	"crypto/rand"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// SetupInitialDeposits prepares the entered amount of deposits
// and secret keys.
func SetupInitialDeposits(t testing.TB, numDeposits uint64, generateKeys bool) ([]*ethpb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*ethpb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		pubkey := []byte{}
		var sig [96]byte
		var withdrawalCreds [32]byte
		copy(withdrawalCreds[:], []byte("testing"))
		depositData := &ethpb.Deposit_Data{
			Amount:                params.BeaconConfig().MaxEffectiveBalance,
			WithdrawalCredentials: withdrawalCreds[:],
		}
		if generateKeys {
			priv, err := bls.RandKey(rand.Reader)
			if err != nil {
				t.Fatalf("could not generate random key: %v", err)
			}
			privKeys[i] = priv
			pubkey = priv.PublicKey().Marshal()
			depositData.PublicKey = pubkey
			domain := bls.Domain(params.BeaconConfig().DomainDeposit, params.BeaconConfig().GenesisForkVersion)
			root, err := ssz.SigningRoot(depositData)
			if err != nil {
				t.Fatalf("could not get signing root of deposit data %v", err)
			}
			marshalledSig := priv.Sign(root[:], domain).Marshal()
			copy(sig[:], marshalledSig)
			depositData.Signature = sig[:]
		} else {
			privKeys = []*bls.SecretKey{}
			pubkey = make([]byte, params.BeaconConfig().BLSPubkeyLength)
			copy(pubkey[:], []byte(strconv.FormatUint(uint64(i), 10)))
			copy(sig[:], []byte("testing"))
			depositData.PublicKey = pubkey
			depositData.Signature = sig[:]
		}

		deposits[i] = &ethpb.Deposit{
			Data: depositData,
		}
	}
	deposits, _ = GenerateDepositProof(t, deposits)
	return deposits, privKeys
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
		BlockHash:   root[:],
		DepositRoot: root[:],
	}

	return eth1Data
}
