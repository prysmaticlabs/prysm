package stateutil

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// Eth1Root computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func Eth1Root(hasher ssz.HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	if eth1Data == nil {
		return [32]byte{}, errors.New("nil eth1 data")
	}

	enc := eth1DataEncKey(eth1Data)
	if features.Get().EnableSSZCache {
		if found, ok := CachedHasher.rootsCache.Get(string(enc)); ok && found != nil {
			return found.([32]byte), nil
		}
	}

	root, err := Eth1DataRootWithHasher(hasher, eth1Data)
	if err != nil {
		return [32]byte{}, err
	}

	if features.Get().EnableSSZCache {
		CachedHasher.rootsCache.Set(string(enc), root, 32)
	}
	return root, nil
}

// eth1DataVotesRoot computes the HashTreeRoot Merkleization of
// a list of Eth1Data structs according to the eth2
// Simple Serialize specification.
func eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
	hashKey, err := Eth1DatasEncKey(eth1DataVotes)
	if err != nil {
		return [32]byte{}, err
	}

	if features.Get().EnableSSZCache {
		if found, ok := CachedHasher.rootsCache.Get(string(hashKey[:])); ok && found != nil {
			return found.([32]byte), nil
		}
	}
	root, err := Eth1DatasRoot(eth1DataVotes)
	if err != nil {
		return [32]byte{}, err
	}
	if features.Get().EnableSSZCache {
		CachedHasher.rootsCache.Set(string(hashKey[:]), root, 32)
	}
	return root, nil
}
