package v1

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// Eth1Data corresponding to the proof-of-work chain information stored in the beacon state.
func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.eth1Data == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.eth1DataInternal()
}

// eth1DataInternal corresponding to the proof-of-work chain information stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) eth1DataInternal() *ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.eth1Data == nil {
		return nil
	}

	return ethpb.CopyETH1Data(b.eth1Data)
}

// Eth1DataVotes corresponds to votes from Ethereum on the canonical proof-of-work chain
// data retrieved from eth1.
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.eth1DataVotes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.eth1DataVotesInternal()
}

// eth1DataVotesInternal corresponds to votes from Ethereum on the canonical proof-of-work chain
// data retrieved from eth1.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) eth1DataVotesInternal() []*ethpb.Eth1Data {
	if !b.hasInnerState() {
		return nil
	}
	if b.eth1DataVotes == nil {
		return nil
	}

	res := make([]*ethpb.Eth1Data, len(b.eth1DataVotes))
	for i := 0; i < len(res); i++ {
		res[i] = ethpb.CopyETH1Data(b.eth1DataVotes[i])
	}
	return res
}

// Eth1DepositIndex corresponds to the index of the deposit made to the
// validator deposit contract at the time of this state's eth1 data.
func (b *BeaconState) Eth1DepositIndex() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.eth1DepositIndexInternal()
}

// eth1DepositIndex corresponds to the index of the deposit made to the
// validator deposit contract at the time of this state's eth1 data.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) eth1DepositIndexInternal() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	return b.eth1DepositIndex
}

// eth1Root computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the Ethereum
// Simple Serialize specification.
func eth1Root(hasher ssz.HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	if eth1Data == nil {
		return [32]byte{}, errors.New("nil eth1 data")
	}

	enc := stateutil.Eth1DataEncKey(eth1Data)
	if features.Get().EnableSSZCache {
		if found, ok := cachedHasher.rootsCache.Get(string(enc)); ok && found != nil {
			return found.([32]byte), nil
		}
	}

	root, err := stateutil.Eth1DataRootWithHasher(hasher, eth1Data)
	if err != nil {
		return [32]byte{}, err
	}

	if features.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(enc), root, 32)
	}
	return root, nil
}

// eth1DataVotesRoot computes the HashTreeRoot Merkleization of
// a list of Eth1Data structs according to the Ethereum
// Simple Serialize specification.
func eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
	hashKey, err := stateutil.Eth1DatasEncKey(eth1DataVotes)
	if err != nil {
		return [32]byte{}, err
	}

	if features.Get().EnableSSZCache {
		if found, ok := cachedHasher.rootsCache.Get(string(hashKey[:])); ok && found != nil {
			return found.([32]byte), nil
		}
	}
	root, err := stateutil.Eth1DatasRoot(eth1DataVotes)
	if err != nil {
		return [32]byte{}, err
	}
	if features.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(hashKey[:]), root, 32)
	}
	return root, nil
}
