package stateV0

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// eth1Root computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func eth1Root(hasher htrutils.HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	enc := make([]byte, 0, 96)
	fieldRoots := make([][]byte, 3)
	for i := 0; i < len(fieldRoots); i++ {
		fieldRoots[i] = make([]byte, 32)
	}
	if eth1Data != nil {
		if len(eth1Data.DepositRoot) > 0 {
			depRoot := bytesutil.ToBytes32(eth1Data.DepositRoot)
			fieldRoots[0] = depRoot[:]
			enc = append(enc, depRoot[:]...)
		}
		eth1DataCountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
		eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
		fieldRoots[1] = eth1CountRoot[:]
		enc = append(enc, eth1CountRoot[:]...)
		if len(eth1Data.BlockHash) > 0 {
			blockHash := bytesutil.ToBytes32(eth1Data.BlockHash)
			fieldRoots[2] = blockHash[:]
			enc = append(enc, blockHash[:]...)
		}
		if featureconfig.Get().EnableSSZCache {
			if found, ok := cachedHasher.rootsCache.Get(string(enc)); ok && found != nil {
				return found.([32]byte), nil
			}
		}
	}
	root, err := htrutils.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	if featureconfig.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(enc), root, 32)
	}
	return root, nil
}

// eth1DataVotesRoot computes the HashTreeRoot Merkleization of
// a list of Eth1Data structs according to the eth2
// Simple Serialize specification.
func eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
	eth1VotesRoots := make([][]byte, 0)
	enc := make([]byte, len(eth1DataVotes)*32)
	hasher := hashutil.CustomSHA256Hasher()
	for i := 0; i < len(eth1DataVotes); i++ {
		eth1, err := eth1Root(hasher, eth1DataVotes[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		copy(enc[(i*32):(i+1)*32], eth1[:])
		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
	}
	hashKey := hashutil.FastSum256(enc)
	if featureconfig.Get().EnableSSZCache {
		if found, ok := cachedHasher.rootsCache.Get(string(hashKey[:])); ok && found != nil {
			return found.([32]byte), nil
		}
	}
	eth1Chunks, err := htrutils.Pack(eth1VotesRoots)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not chunk eth1 votes roots")
	}
	eth1VotesRootsRoot, err := htrutils.BitwiseMerkleize(
		hasher,
		eth1Chunks,
		uint64(len(eth1Chunks)),
		uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))),
	)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	eth1VotesRootBuf := new(bytes.Buffer)
	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(eth1DataVotes))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
	}
	// We need to mix in the length of the slice.
	eth1VotesRootBufRoot := make([]byte, 32)
	copy(eth1VotesRootBufRoot, eth1VotesRootBuf.Bytes())
	root := htrutils.MixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot)
	if featureconfig.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(hashKey[:]), root, 32)
	}
	return root, nil
}
