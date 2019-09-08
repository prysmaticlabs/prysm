package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/big"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
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
	numValidators      = flag.Int("num-validators", 0, "Number of validators to deterministically include in the generated genesis state")
	useMainnetConfig   = flag.Bool("mainnet-config", false, "Select whether genesis state should be generated with mainnet or minimal (default) params")
	genesisTime        = flag.Uint64("genesis-time", 0, "Unix timestamp used as the genesis time in the generated genesis state")
	sszOutputFile      = flag.String("output-ssz", "", "Output filename of the SSZ marshaling of the generated genesis state")
	yamlOutputFile     = flag.String("output-yaml", "", "Output filename of the YAML marshaling of the generated genesis state")
	jsonOutputFile     = flag.String("output-json", "", "Output filename of the JSON marshaling of the generated genesis state")
	// This is the recommended mock eth1 block hash according to the Eth2 interop guidelines.
	// https://github.com/ethereum/eth2.0-pm/blob/a085c9870f3956d6228ed2a40cd37f0c6580ecd7/interop/mocked_start/README.md
	mockEth1BlockHash = []byte{66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66}
)

func main() {
	flag.Parse()
	// TODO(#3398): Cannot generate more than 190 keys due to BLS errors.
	if *numValidators == 0 {
		log.Fatal("Expected --num-validators to have been provided, received 0")
	}
	if *genesisTime == 0 {
		log.Print("No --genesis-time specified, defaulting to 0 as the unix timestamp")
	}
	if *sszOutputFile == "" && *yamlOutputFile == "" && *jsonOutputFile == "" {
		log.Fatal("Expected --output-ssz, --output-yaml, or --output-json to have been provided, received nil")
	}
	if !*useMainnetConfig {
		params.OverrideBeaconConfig(params.MinimalSpecConfig())
	}
	privKeys, pubKeys, err := deterministicallyGenerateKeys(*numValidators)
	if err != nil {
		log.Fatalf("Could not deterministically generate keys for %d validators: %v", *numValidators, err)
	}
	depositDataItems, depositDataRoots, err := depositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		log.Fatalf("Could not generate deposit data from keys: %v", err)
	}
	trie, err := trieutil.GenerateTrieFromItems(
		depositDataRoots,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		log.Fatalf("Could not generate Merkle trie for deposit proofs: %v", err)
	}
	deposits, err := generateDepositsFromData(depositDataItems, trie)
	if err != nil {
		log.Fatalf("Could not generate deposits from the deposit data provided: %v", err)
	}
	root := trie.Root()
	genesisState, err := state.GenesisBeaconState(deposits, *genesisTime, &ethpb.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
		BlockHash:    mockEth1BlockHash,
	})
	if err != nil {
		log.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	if *sszOutputFile != "" {
		encodedState, err := ssz.Marshal(genesisState)
		if err != nil {
			log.Fatalf("Could not ssz marshal the genesis beacon state: %v", err)
		}
		if err := ioutil.WriteFile(*sszOutputFile, encodedState, 0644); err != nil {
			log.Fatalf("Could not write encoded genesis beacon state to file: %v", err)
		}
		log.Printf("Done writing to %s", *sszOutputFile)
	}
	if *yamlOutputFile != "" {
		encodedState, err := yaml.Marshal(genesisState)
		if err != nil {
			log.Fatalf("Could not yaml marshal the genesis beacon state: %v", err)
		}
		if err := ioutil.WriteFile(*yamlOutputFile, encodedState, 0644); err != nil {
			log.Fatalf("Could not write encoded genesis beacon state to file: %v", err)
		}
		log.Printf("Done writing to %s", *yamlOutputFile)
	}
	if *jsonOutputFile != "" {
		encodedState, err := json.Marshal(genesisState)
		if err != nil {
			log.Fatalf("Could not json marshal the genesis beacon state: %v", err)
		}
		if err := ioutil.WriteFile(*jsonOutputFile, encodedState, 0644); err != nil {
			log.Fatalf("Could not write encoded genesis beacon state to file: %v", err)
		}
		log.Printf("Done writing to %s", *jsonOutputFile)
	}
}

// Deterministically creates BLS private keys using a fixed curve order according to
// the algorithm specified in the Eth2.0-Specs interop mock start section found here:
// https://github.com/ethereum/eth2.0-pm/blob/a085c9870f3956d6228ed2a40cd37f0c6580ecd7/interop/mocked_start/README.md
func deterministicallyGenerateKeys(n int) ([]*bls.SecretKey, []*bls.PublicKey, error) {
	privKeys := make([]*bls.SecretKey, n)
	pubKeys := make([]*bls.PublicKey, n)
	for i := 0; i < n; i++ {
		enc := make([]byte, 32)
		binary.LittleEndian.PutUint32(enc, uint32(i))
		hash := hashutil.Hash(enc)
		// Reverse byte order to big endian for use with big ints.
		b := reverseByteOrder(hash[:])
		num := new(big.Int)
		num = num.SetBytes(b)
		order := new(big.Int)
		var ok bool
		order, ok = order.SetString(blsCurveOrder, 10)
		if !ok {
			return nil, nil, errors.New("could not set bls curve order as big int")
		}
		num = num.Mod(num, order)
		priv, err := bls.SecretKeyFromBytes(num.Bytes())
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create bls secret key from raw bytes")
		}
		privKeys[i] = priv
		pubKeys[i] = priv.PublicKey()
	}
	return privKeys, pubKeys, nil
}

// Generates a list of deposit items by creating proofs for each of them from a sparse Merkle trie.
func generateDepositsFromData(depositDataItems []*ethpb.Deposit_Data, trie *trieutil.MerkleTrie) ([]*ethpb.Deposit, error) {
	deposits := make([]*ethpb.Deposit, len(depositDataItems))
	for i, item := range depositDataItems {
		proof, err := trie.MerkleProof(i)
		if err != nil {
			return nil, errors.Wrapf(err, "could not generate proof for deposit %d", i)
		}
		deposits[i] = &ethpb.Deposit{
			Proof: proof,
			Data:  item,
		}
	}
	return deposits, nil
}

// Generates a list of deposit data items from a set of BLS validator keys.
func depositDataFromKeys(privKeys []*bls.SecretKey, pubKeys []*bls.PublicKey) ([]*ethpb.Deposit_Data, [][]byte, error) {
	dataRoots := make([][]byte, len(privKeys))
	depositDataItems := make([]*ethpb.Deposit_Data, len(privKeys))
	for i := 0; i < len(privKeys); i++ {
		data, err := createDepositData(privKeys[i], pubKeys[i])
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not create deposit data for key: %#x", privKeys[i].Marshal())
		}
		hash, err := ssz.HashTreeRoot(data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not hash tree root deposit data item")
		}
		dataRoots[i] = hash[:]
		depositDataItems[i] = data
	}
	return depositDataItems, dataRoots, nil
}

// Generates a deposit data item from BLS keys and signs the hash tree root of the data.
func createDepositData(privKey *bls.SecretKey, pubKey *bls.PublicKey) (*ethpb.Deposit_Data, error) {
	di := &ethpb.Deposit_Data{
		PublicKey:             pubKey.Marshal(),
		WithdrawalCredentials: withdrawalCredentialsHash(pubKey.Marshal()),
		Amount:                params.BeaconConfig().MaxEffectiveBalance,
	}
	sr, err := ssz.SigningRoot(di)
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

// Switch the endianness of a byte slice by reversing its order.
func reverseByteOrder(input []byte) []byte {
	b := input
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-i-1] = b[len(b)-i-1], b[i]
	}
	return b
}
