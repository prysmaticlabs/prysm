package testutil

import (
	"sync"
	"testing"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var lock sync.Mutex

// Caches
var cachedDeposits []*ethpb.Deposit
var privKeys []bls.SecretKey
var trie *trieutil.SparseMerkleTrie

// DeterministicDepositsAndKeys returns the entered amount of deposits and secret keys.
// The deposits are configured such that for deposit n the validator
// account is key n and the withdrawal account is key n+1.  As such,
// if all secret keys for n validators are required then numDeposits
// should be n+1.
func DeterministicDepositsAndKeys(numDeposits uint64) ([]*ethpb.Deposit, []bls.SecretKey, error) {
	lock.Lock()
	defer lock.Unlock()
	var err error

	// Populate trie cache, if not initialized yet.
	if trie == nil {
		trie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create new trie")
		}
	}

	// If more deposits requested than cached, generate more.
	if numDeposits > uint64(len(cachedDeposits)) {
		numExisting := uint64(len(cachedDeposits))
		numRequired := numDeposits - uint64(len(cachedDeposits))
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

			domain, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not compute domain")
			}
			root, err := ssz.SigningRoot(depositData)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get signing root of deposit data")
			}
			sigRoot, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: root[:], Domain: domain})
			if err != nil {
				return nil, nil, err
			}
			depositData.Signature = secretKeys[i].Sign(sigRoot[:]).Marshal()

			deposit := &ethpb.Deposit{
				Data: depositData,
			}
			cachedDeposits = append(cachedDeposits, deposit)

			hashedDeposit, err := ssz.HashTreeRoot(deposit.Data)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not tree hash deposit data")
			}

			trie.Insert(hashedDeposit[:], int(numExisting+i))
		}
	}

	depositTrie, _, err := DeterministicDepositTrie(int(numDeposits))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create deposit trie")
	}
	requestedDeposits := cachedDeposits[:numDeposits]
	for i := range requestedDeposits {
		proof, err := depositTrie.MerkleProof(i)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create merkle proof")
		}
		requestedDeposits[i].Proof = proof
	}

	return requestedDeposits, privKeys[0:numDeposits], nil
}

// DeterministicDepositTrie returns a merkle trie of the requested size from the
// deterministic deposits.
func DeterministicDepositTrie(size int) (*trieutil.SparseMerkleTrie, [][32]byte, error) {
	items := trie.Items()
	if size > len(items) {
		return nil, [][32]byte{}, errors.New("requested a larger tree than amount of deposits")
	}

	if trie == nil {
		return nil, [][32]byte{}, errors.New("trie cache is empty, generate deposits at an earlier point")
	}

	items = items[:size]
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
		return nil, errors.Wrap(err, "failed to create trie")
	}
	root := depositTrie.Root()
	eth1Data := &ethpb.Eth1Data{
		BlockHash:    root[:],
		DepositRoot:  root[:],
		DepositCount: uint64(size),
	}
	return eth1Data, nil
}

// DeterministicGenesisState returns a genesis state made using the deterministic deposits.
func DeterministicGenesisState(t testing.TB, numValidators uint64) (*stateTrie.BeaconState, []bls.SecretKey) {
	deposits, privKeys, err := DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	ResetCache()
	return beaconState, privKeys
}

// DepositTrieFromDeposits takes an array of deposits and returns the deposit trie.
func DepositTrieFromDeposits(deposits []*ethpb.Deposit) (*trieutil.SparseMerkleTrie, [][32]byte, error) {
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
	privKeys = []bls.SecretKey{}
	cachedDeposits = []*ethpb.Deposit{}
}

// DeterministicDepositsAndKeysSameValidator returns the entered amount of deposits and secret keys
// of the same validator. This is for negative test cases such as same deposits from same validators in a block don't
// result in duplicated validator indices.
func DeterministicDepositsAndKeysSameValidator(numDeposits uint64) ([]*ethpb.Deposit, []bls.SecretKey, error) {
	lock.Lock()
	defer lock.Unlock()
	var err error

	// Populate trie cache, if not initialized yet.
	if trie == nil {
		trie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create new trie")
		}
	}

	// If more deposits requested than cached, generate more.
	if numDeposits > uint64(len(cachedDeposits)) {
		numExisting := uint64(len(cachedDeposits))
		numRequired := numDeposits - uint64(len(cachedDeposits))
		// Fetch the required number of keys.
		secretKeys, publicKeys, err := interop.DeterministicallyGenerateKeys(numExisting, numRequired+1)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create deterministic keys: ")
		}
		privKeys = append(privKeys, secretKeys[:len(secretKeys)-1]...)

		// Create the new deposits and add them to the trie. Always use the first validator to create deposit
		for i := uint64(0); i < numRequired; i++ {
			withdrawalCreds := hashutil.Hash(publicKeys[1].Marshal())
			withdrawalCreds[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

			depositData := &ethpb.Deposit_Data{
				PublicKey:             publicKeys[1].Marshal(),
				Amount:                params.BeaconConfig().MaxEffectiveBalance,
				WithdrawalCredentials: withdrawalCreds[:],
			}

			domain, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not compute domain")
			}
			root, err := ssz.SigningRoot(depositData)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get signing root of deposit data")
			}
			sigRoot, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: root[:], Domain: domain})
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get signing root of deposit data and domain")
			}
			// Always use the same validator to sign
			depositData.Signature = secretKeys[1].Sign(sigRoot[:]).Marshal()

			deposit := &ethpb.Deposit{
				Data: depositData,
			}
			cachedDeposits = append(cachedDeposits, deposit)

			hashedDeposit, err := ssz.HashTreeRoot(deposit.Data)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not tree hash deposit data")
			}

			trie.Insert(hashedDeposit[:], int(numExisting+i))
		}
	}

	depositTrie, _, err := DeterministicDepositTrie(int(numDeposits))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create deposit trie")
	}
	requestedDeposits := cachedDeposits[:numDeposits]
	for i := range requestedDeposits {
		proof, err := depositTrie.MerkleProof(i)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create merkle proof")
		}
		requestedDeposits[i].Proof = proof
	}

	return requestedDeposits, privKeys[0:numDeposits], nil
}
