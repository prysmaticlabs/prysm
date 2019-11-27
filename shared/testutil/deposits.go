package testutil

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var lock sync.Mutex
var stateLock sync.Mutex

// Caches
var deposits []*ethpb.Deposit
var privKeys []*bls.SecretKey
var trie *trieutil.MerkleTrie

// DeterministicDepositsAndKeys returns the entered amount of deposits and secret keys.
// The deposits are configured such that for deposit n the validator
// account is key n and the withdrawal account is key n+1.  As such,
// if all secret keys for n validators are required then numDeposits
// should be n+1.
func DeterministicDepositsAndKeys(numDeposits uint64) ([]*ethpb.Deposit, []*bls.SecretKey, error) {
	lock.Lock()
	defer lock.Unlock()
	var err error

	// Populate trie cache, if not initialized yet.
	if trie == nil {
		trie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, nil, err
		}
	}

	if numDeposits > uint64(len(deposits)) {
		// More deposits requested than cached.
		numExisting := uint64(len(deposits))
		numRequired := numDeposits - uint64(len(deposits))
		// Fetch the required number of keys.
		secretKeys, publicKeys, err := interop.DeterministicallyGenerateKeys(numExisting, numRequired+1)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create deterministic keys: ")
		}
		privKeys = append(privKeys, secretKeys[:len(secretKeys)-1]...)

		// Create the new deposits and add them to the trie.
		for i := uint64(0); i < numRequired; i++ {
			withdrawalCreds := hashutil.Hash(publicKeys[i+1].Marshal())
			withdrawalCreds[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

			depositData := &ethpb.Deposit_Data{
				PublicKey:             publicKeys[i].Marshal()[:],
				Amount:                params.BeaconConfig().MaxEffectiveBalance,
				WithdrawalCredentials: withdrawalCreds[:],
			}

			domain := bls.ComputeDomain(params.BeaconConfig().DomainDeposit)
			root, err := ssz.SigningRoot(depositData)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get signing root of deposit data")
			}
			depositData.Signature = secretKeys[i].Sign(root[:], domain).Marshal()

			deposit := &ethpb.Deposit{
				Data: depositData,
			}
			deposits = append(deposits, deposit)

			hashedDeposit, err := ssz.HashTreeRoot(deposit.Data)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not tree hash deposit data")
			}

			if err := trie.InsertIntoTrie(hashedDeposit[:], int(numExisting+i)); err != nil {
				return nil, nil, errors.Wrap(err, "could not tree hash deposit data")
			}
		}

		for i := range deposits {
			proof, err := trie.MerkleProof(int(i))
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not create merkle proof")
			}
			deposits[i].Proof = proof
		}
	}

	return deposits, privKeys[0:numDeposits], nil
}

// DeterministicDepositTrie returns a merkle trie of the requested size from the
// deterministic deposits.
func DeterministicDepositTrie(size int) (*trieutil.MerkleTrie, [][32]byte, error) {
	lock.Lock()
	defer lock.Unlock()
	if size > len(deposits) {
		return nil, [][32]byte{}, errors.New("requested a larger tree than amount of deposits")
	}

	if trie == nil {
		return nil, [][32]byte{}, errors.New("trie cache is empty, generate deposits at an earlier point")
	}

	items := trie.Items()[:size]
	depositTrie, err := trieutil.GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, [][32]byte{}, errors.Wrapf(err, "could not generate trie of %d length", size)
	}

	roots := make([][32]byte, len(items))
	for i, dep := range items {
		roots[i] = bytesutil.ToBytes32(dep)
	}

	return depositTrie, roots, nil
}

// DeterministicEth1Data takes an array of deposits and returns the eth1Data made from the deposit trie.
func DeterministicEth1Data(size int) (*ethpb.Eth1Data, error) {
	depositTrie, _, err := DeterministicDepositTrie(size)
	if err != nil {
		return nil, err
	}
	root := depositTrie.Root()
	eth1Data := &ethpb.Eth1Data{
		BlockHash:    root[:],
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
	}
	return eth1Data, nil
}

// DeterministicGenesisState returns a genesis state made using the deterministic deposits.
func DeterministicGenesisState(numValidators uint64) (*pb.BeaconState, []*bls.SecretKey, error) {
	stateLock.Lock()
	defer stateLock.Unlock()
	deposits, privKeys, err := DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get %d deposits", numValidators)
	}
	fmt.Println(deposits)
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators)
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators)
	}
	return beaconState, privKeys, nil
}

// DepositTrieFromDeposits takes an array of deposits and returns the deposit trie.
func DepositTrieFromDeposits(deposits []*ethpb.Deposit) (*trieutil.MerkleTrie, [][32]byte, error) {
	lock.Lock()
	defer lock.Unlock()
	encodedDeposits := make([][]byte, len(deposits))
	for i := 0; i < len(encodedDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
		if err != nil {
			return nil, [][32]byte{}, errors.Wrap(err, "could not tree hash deposit data")
		}
		encodedDeposits[i] = hashedDeposit[:]
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(encodedDeposits, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		return nil, [][32]byte{}, errors.Wrap(err, "Could not generate deposit trie")
	}

	roots := make([][32]byte, len(deposits))
	for i, dep := range encodedDeposits {
		roots[i] = bytesutil.ToBytes32(dep)
	}

	return depositTrie, roots, nil
}

// ResetCache clears out the old trie, private keys and deposits.
func ResetCache() {
	trie = nil
	privKeys = []*bls.SecretKey{}
	deposits = []*ethpb.Deposit{}
}
