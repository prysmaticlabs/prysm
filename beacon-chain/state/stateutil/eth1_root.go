package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Eth1DataEncKey returns the encoded key in bytes of input `eth1Data`,
// the returned key bytes can be used for caching purposes.
func Eth1DataEncKey(eth1Data *ethpb.Eth1Data) []byte {
	enc := make([]byte, 0, 96)
	if eth1Data != nil {
		if len(eth1Data.DepositRoot) > 0 {
			depRoot := bytesutil.ToBytes32(eth1Data.DepositRoot)
			enc = append(enc, depRoot[:]...)
		}
		eth1DataCountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
		eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
		enc = append(enc, eth1CountRoot[:]...)
		if len(eth1Data.BlockHash) > 0 {
			blockHash := bytesutil.ToBytes32(eth1Data.BlockHash)
			enc = append(enc, blockHash[:]...)
		}
	}
	return enc
}

// Eth1DataRootWithHasher returns the hash tree root of input `eth1Data`.
func Eth1DataRootWithHasher(hasher htrutils.HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	if eth1Data == nil {
		return [32]byte{}, errors.New("nil eth1 data")
	}

	fieldRoots := make([][]byte, 3)
	for i := 0; i < len(fieldRoots); i++ {
		fieldRoots[i] = make([]byte, 32)
	}
	if len(eth1Data.DepositRoot) > 0 {
		depRoot := bytesutil.ToBytes32(eth1Data.DepositRoot)
		fieldRoots[0] = depRoot[:]
	}
	eth1DataCountBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
	eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
	fieldRoots[1] = eth1CountRoot[:]
	if len(eth1Data.BlockHash) > 0 {
		blockHash := bytesutil.ToBytes32(eth1Data.BlockHash)
		fieldRoots[2] = blockHash[:]
	}
	root, err := htrutils.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	return root, nil
}

// Eth1DataEncKey returns the encoded key in bytes of input `eth1Data`s,
// the returned key bytes can be used for caching purposes.
func Eth1DatasEncKey(eth1Datas []*ethpb.Eth1Data) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	enc := make([]byte, len(eth1Datas)*32)
	for i := 0; i < len(eth1Datas); i++ {
		eth1, err := Eth1DataRootWithHasher(hasher, eth1Datas[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		copy(enc[(i*32):(i+1)*32], eth1[:])
	}
	hashKey := hashutil.FastSum256(enc)
	return hashKey, nil
}

// Eth1DatasRoot returns the hash tree root of input `eth1Datas`.
func Eth1DatasRoot(eth1Datas []*ethpb.Eth1Data) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	eth1VotesRoots := make([][]byte, 0)
	for i := 0; i < len(eth1Datas); i++ {
		eth1, err := Eth1DataRootWithHasher(hasher, eth1Datas[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
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
	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(eth1Datas))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
	}
	// We need to mix in the length of the slice.
	eth1VotesRootBufRoot := make([]byte, 32)
	copy(eth1VotesRootBufRoot, eth1VotesRootBuf.Bytes())
	root := htrutils.MixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot)

	return root, nil
}
