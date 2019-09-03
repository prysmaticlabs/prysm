package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"log"
	"math/big"

	"github.com/ghodss/yaml"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

const (
	blsWithdrawalPrefixByte = byte(0)
	blsCurveOrder           = "52435875175126190479447740508185965837690552500527637822603658699938581184513"
)

var (
	domainDeposit      = [4]byte{3, 0, 0, 0}
	genesisForkVersion = []byte{0, 0, 0, 0}
	sszOutputFile      = flag.String("output-ssz", "", "Output filename of the SSZ marshaling of the generated genesis state")
	yamlOutputFile     = flag.String("output-yaml", "", "Output filename of the YAML marshaling of the generated genesis state")
	numValidators      = flag.Int("num-validators", 0, "Number of validators to deterministically include in the generated genesis state")
	genesisTime        = flag.Uint64("genesis-time", 0, "Unix timestamp used as the genesis time in the generated genesis state")
)

func main() {
	flag.Parse()
	if *numValidators == 0 {
		log.Fatal("Expected --num-validators to have been provided, received 0")
	}
	if *genesisTime == 0 {
		log.Print("No --genesis-time specified, defaulting to 0 as the unix timestamp")
	}
	if *sszOutputFile == "" && *yamlOutputFile == "" {
		log.Fatal("Expected --output-ssz or --output-yaml to have been provided, received nil")
	}
	params.UseDemoBeaconConfig()
	privKeys, pubKeys := deterministicallyGenerateKeys(*numValidators)
	hashes := make([][]byte, len(privKeys))
	dataList := make([]*ethpb.Deposit_Data, len(privKeys))
	for i := 0; i < len(dataList); i++ {
		data, err := createDepositData(privKeys[i], pubKeys[i])
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
	root := trie.Root()
	// Mock eth1 data for the beacon state.
	blockHash, err := hex.DecodeString("4242424242424242424242424242424242424242424242424242424242424242")
	if err != nil {
		panic(err)
	}
	genesisState, err := state.GenesisBeaconState(deposits, *genesisTime, &ethpb.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
		BlockHash:    blockHash,
	})
	if err != nil {
		panic(err)
	}
	if *sszOutputFile != "" {
		encodedState, err := ssz.Marshal(genesisState)
		if err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(*sszOutputFile, encodedState, 0644); err != nil {
			panic(err)
		}
		log.Printf("Done writing to %s", *sszOutputFile)
	}
	if *yamlOutputFile != "" {
		encodedState, err := yaml.Marshal(genesisState)
		if err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(*yamlOutputFile, encodedState, 0644); err != nil {
			panic(err)
		}
		log.Printf("Done writing to %s", *yamlOutputFile)
	}
}

func deterministicallyGenerateKeys(n int) ([]*bls.SecretKey, []*bls.PublicKey) {
	privKeys := make([]*bls.SecretKey, n)
	pubKeys := make([]*bls.PublicKey, n)
	for i := 0; i < n; i++ {
		enc := make([]byte, 32)
		binary.LittleEndian.PutUint32(enc, uint32(i))
		hash := hashutil.Hash(enc)
		b := hash[:]
		// Reverse byte order to big endian for use with big ints.
		for i := 0; i < len(b)/2; i++ {
			b[i], b[len(b)-i-1] = b[len(b)-i-1], b[i]
		}
		num := new(big.Int)
		num = num.SetBytes(b)
		order := new(big.Int)
		var ok bool
		order, ok = order.SetString(blsCurveOrder, 10)
		if !ok {
			panic("Not ok")
		}
		num = num.Mod(num, order)
		priv, err := bls.SecretKeyFromBytes(num.Bytes())
		if err != nil {
			panic(err)
		}
		privKeys[i] = priv
		pubKeys[i] = priv.PublicKey()
	}
	return privKeys, pubKeys
}

func createDepositData(privKey *bls.SecretKey, pubKey *bls.PublicKey) (*ethpb.Deposit_Data, error) {
	di := &ethpb.Deposit_Data{
		PublicKey:             pubKey.Marshal(),
		WithdrawalCredentials: withdrawalCredentialsHash(pubKey.Marshal()),
		Amount:                params.BeaconConfig().MaxEffectiveBalance,
	}
	sr, err := ssz.HashTreeRoot(di)
	if err != nil {
		return nil, err
	}
	domain := bls.Domain(domainDeposit[:], genesisForkVersion)
	di.Signature = privKey.Sign(sr[:], domain).Marshal()
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
