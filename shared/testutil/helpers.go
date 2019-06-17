package testutil

import (
	"crypto/rand"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// SetupInitialDeposits prepares the entered amount of deposits
// and secret keys.
func SetupInitialDeposits(t testing.TB, numDeposits uint64, generateKeys bool) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		pubkey := []byte{}
		if generateKeys {
			priv, err := bls.RandKey(rand.Reader)
			if err != nil {
				t.Fatalf("could not generate random key: %v", err)
			}
			privKeys[i] = priv
			pubkey = priv.PublicKey().Marshal()
		} else {
			privKeys = []*bls.SecretKey{}
			pubkey = make([]byte, params.BeaconConfig().BLSPubkeyLength)
			copy(pubkey[:], []byte(strconv.FormatUint(uint64(i), 10)))
		}

		depositData := &pb.DepositData{
			Pubkey:                pubkey,
			Amount:                params.BeaconConfig().MaxDepositAmount,
			WithdrawalCredentials: []byte{1},
		}
		deposits[i] = &pb.Deposit{
			Data:  depositData,
			Index: uint64(i),
		}
	}

	return deposits, privKeys
}

// GenerateEth1Data takes an array of deposits and generates the deposit trie for them.
func GenerateEth1Data(t testing.TB, deposits []*pb.Deposit) *pb.Eth1Data {
	encodedDeposits := make([][]byte, len(deposits))
	for i := 0; i < len(encodedDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
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
		proof, err := depositTrie.MerkleProof(int(deposits[i].Index))
		if err != nil {
			t.Fatalf("Could not generate proof: %v", err)
		}
		deposits[i].Proof = proof
	}
	root := depositTrie.Root()
	eth1Data := &pb.Eth1Data{
		BlockRoot:   root[:],
		DepositRoot: root[:],
	}

	return eth1Data
}
