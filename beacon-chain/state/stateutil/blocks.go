package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockHeaderRoot computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func BlockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)
	if header != nil {
		headerSlotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerSlotBuf, header.Slot)
		headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
		fieldRoots[0] = headerSlotRoot[:]
		proposerIdxBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerIdxBuf, header.ProposerIndex)
		proposerIndexRoot := bytesutil.ToBytes32(proposerIdxBuf)
		fieldRoots[1] = proposerIndexRoot[:]
		parentRoot := bytesutil.ToBytes32(header.ParentRoot)
		fieldRoots[2] = parentRoot[:]
		stateRoot := bytesutil.ToBytes32(header.StateRoot)
		fieldRoots[3] = stateRoot[:]
		bodyRoot := bytesutil.ToBytes32(header.BodyRoot)
		fieldRoots[4] = bodyRoot[:]
	}
	return bitwiseMerkleize(hashutil.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// BlockRoot returns the block hash tree root of the provided block.
func BlockRoot(blk *ethpb.BeaconBlock) ([32]byte, error) {
	if !featureconfig.Get().EnableBlockHTR {
		return ssz.HashTreeRoot(blk)
	}
	fieldRoots := make([][32]byte, 5)
	if blk != nil {
		headerSlotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerSlotBuf, blk.Slot)
		headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
		fieldRoots[0] = headerSlotRoot
		proposerIdxBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerIdxBuf, blk.ProposerIndex)
		proposerIndexRoot := bytesutil.ToBytes32(proposerIdxBuf)
		fieldRoots[1] = proposerIndexRoot
		parentRoot := bytesutil.ToBytes32(blk.ParentRoot)
		fieldRoots[2] = parentRoot
		stateRoot := bytesutil.ToBytes32(blk.StateRoot)
		fieldRoots[3] = stateRoot
		bodyRoot, err := BlockBodyRoot(blk.Body)
		if err != nil {
			return [32]byte{}, err
		}
		fieldRoots[4] = bodyRoot
	}
	return bitwiseMerkleizeArrays(hashutil.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// BlockBodyRoot returns the hash tree root of the block body.
func BlockBodyRoot(body *ethpb.BeaconBlockBody) ([32]byte, error) {
	if !featureconfig.Get().EnableBlockHTR {
		return ssz.HashTreeRoot(body)
	}
	hasher := hashutil.CustomSHA256Hasher()
	fieldRoots := make([][32]byte, 8)
	if body == nil {
		return [32]byte{}, errors.New("nil block body provided")
	}
	rawRandao := bytesutil.ToBytes96(body.RandaoReveal)
	packedRandao, err := pack([][]byte{rawRandao[:]})
	if err != nil {
		return [32]byte{}, err
	}
	randaoRoot, err := bitwiseMerkleize(hasher, packedRandao, uint64(len(packedRandao)), uint64(len(packedRandao)))
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[0] = randaoRoot

	eth1Root, err := Eth1Root(hasher, body.Eth1Data)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[1] = eth1Root

	graffitiRoot := bytesutil.ToBytes32(body.Graffiti)
	fieldRoots[2] = graffitiRoot

	proposerSlashingsRoot, err := ssz.HashTreeRootWithCapacity(body.ProposerSlashings, 16)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[3] = proposerSlashingsRoot
	attesterSlashingsRoot, err := ssz.HashTreeRootWithCapacity(body.AttesterSlashings, 1)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[4] = attesterSlashingsRoot
	attsRoot, err := blockAttestationRoot(body.Attestations)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[5] = attsRoot

	depositRoot, err := ssz.HashTreeRootWithCapacity(body.Deposits, 16)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[6] = depositRoot

	exitRoot, err := ssz.HashTreeRootWithCapacity(body.VoluntaryExits, 16)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[7] = exitRoot
	return bitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// Eth1Root computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func Eth1Root(hasher HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
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
	root, err := bitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	if featureconfig.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(enc), root, 32)
	}
	return root, nil
}

// Eth1DataVotesRoot computes the HashTreeRoot Merkleization of
// a list of Eth1Data structs according to the eth2
// Simple Serialize specification.
func Eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
	eth1VotesRoots := make([][]byte, 0)
	enc := make([]byte, len(eth1DataVotes)*32)
	hasher := hashutil.CustomSHA256Hasher()
	for i := 0; i < len(eth1DataVotes); i++ {
		eth1, err := Eth1Root(hasher, eth1DataVotes[i])
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
	eth1Chunks, err := pack(eth1VotesRoots)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not chunk eth1 votes roots")
	}
	eth1VotesRootsRoot, err := bitwiseMerkleize(
		hasher,
		eth1Chunks,
		uint64(len(eth1Chunks)),
		params.BeaconConfig().EpochsPerEth1VotingPeriod*params.BeaconConfig().SlotsPerEpoch,
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
	root := mixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot)
	if featureconfig.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(hashKey[:]), root, 32)
	}
	return root, nil
}

// AddInMixin describes a method from which a lenth mixin is added to the
// provided root.
func AddInMixin(root [32]byte, length uint64) ([32]byte, error) {
	rootBuf := new(bytes.Buffer)
	if err := binary.Write(rootBuf, binary.LittleEndian, length); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
	}
	// We need to mix in the length of the slice.
	rootBufRoot := make([]byte, 32)
	copy(rootBufRoot, rootBuf.Bytes())
	return mixInLength(root, rootBufRoot), nil
}
