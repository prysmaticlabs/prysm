package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"

	"gopkg.in/yaml.v2"
)

const (
	blsWithdrawalPrefixByte = byte(0)
)

var (
	inputFile          = flag.String("validator-keys-yaml", "", "Input validator keys YAML file")
	domainDeposit      = [4]byte{3, 0, 0, 0}
	genesisForkVersion = []byte{0, 0, 0, 0}
)

type keyPair struct {
	PrivateKey string `yaml:"privkey"`
	PublicKey  string `yaml:"pubkey"`
}

func main() {
	flag.Parse()
	f, err := os.Open(*inputFile)
	if err != nil {
		panic(err)
	}
	enc, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	var ks []*keyPair
	if err := yaml.Unmarshal(enc, &ks); err != nil {
		panic(err)
	}
	hashes := make([][]byte, len(ks))
	dataList := make([]*ethpb.Deposit_Data, len(ks))
	for i, item := range ks {
		data, err := createDepositData([]byte(item.PrivateKey), []byte(item.PublicKey))
		if err != nil {
			panic(err)
		}
		hash, err := ssz.HashTreeRoot(data)
		if err != nil {
			panic(err)
		}
		hashes[i] = hash[:]
		dataList[i] = data
	}
	trie, err := trieutil.GenerateTrieFromItems(
		hashes,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		panic(err)
	}
	deposits := make([]*ethpb.Deposit, len(dataList))
	for i, item := range dataList {
		proof, err := trie.MerkleProof(i)
		if err != nil {
			panic(err)
		}
		deposits[i] = &ethpb.Deposit{
			Proof: proof,
			Data:  item,
		}
	}
	genesisState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{
		DepositRoot:  make([]byte, 32),
		DepositCount: 0,
		BlockHash:    make([]byte, 32),
	})
	if err != nil {
		panic(err)
	}
	// Now we need to marshal to yaml...
	fmt.Println(genesisState)
}

func createDepositData(privKey []byte, pubKey []byte) (*ethpb.Deposit_Data, error) {
	sk1, err := bls.SecretKeyFromBytes(privKey)
	if err != nil {
		return nil, err
	}
	di := &ethpb.Deposit_Data{
		PublicKey:             pubKey,
		WithdrawalCredentials: withdrawalCredentialsHash(pubKey),
		Amount:                params.BeaconConfig().MaxEffectiveBalance,
	}

	sr, err := ssz.HashTreeRoot(di)
	if err != nil {
		return nil, err
	}

	domain := bls.Domain(domainDeposit[:], genesisForkVersion)
	di.Signature = sk1.Sign(sr[:], domain).Marshal()
	return di, nil
}

// withdrawalCredentialsHash forms a 32 byte hash of the withdrawal public
// address.
//
// The specification is as follows:
//   withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX_BYTE
//   withdrawal_credentials[1:] == hash(withdrawal_pubkey)[1:]
// where withdrawal_credentials is of type bytes32.
func withdrawalCredentialsHash(pubKey []byte) []byte {
	h := hashutil.HashKeccak256(pubKey)
	return append([]byte{blsWithdrawalPrefixByte}, h[0:]...)[:32]
}
