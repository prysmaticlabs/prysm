package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func blockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)
	headerSlotBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(headerSlotBuf, header.Slot)
	headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
	fieldRoots[0] = headerSlotRoot[:]
	fieldRoots[1] = header.ParentRoot
	fieldRoots[2] = header.StateRoot
	fieldRoots[3] = header.BodyRoot
	signatureChunks, err := pack([][]byte{header.Signature})
	if err != nil {
		return [32]byte{}, nil
	}
	sigRoot, err := bitwiseMerkleize(signatureChunks, uint64(len(signatureChunks)), uint64(len(signatureChunks)))
	if err != nil {
		return [32]byte{}, nil
	}
	fieldRoots[4] = sigRoot[:]
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}

func eth1Root(eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	fieldRoots := make([][]byte, 3)
	for i := 0; i < len(fieldRoots); i++ {
		fieldRoots[i] = make([]byte, 32)
	}
	if eth1Data != nil {
		if len(eth1Data.DepositRoot) > 0 {
			fieldRoots[0] = eth1Data.DepositRoot
		}
		eth1DataCountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
		eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
		fieldRoots[1] = eth1CountRoot[:]
		if len(eth1Data.BlockHash) > 0 {
			fieldRoots[2] = eth1Data.BlockHash
		}
	}
	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}

func eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
	eth1VotesRoots := make([][]byte, 0)
	for i := 0; i < len(eth1DataVotes); i++ {
		eth1, err := eth1Root(eth1DataVotes[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
	}
	eth1Chunks, err := pack(eth1VotesRoots)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not chunk eth1 votes roots")
	}
	eth1VotesRootsRoot, err := bitwiseMerkleize(eth1Chunks, uint64(len(eth1Chunks)), uint64(1024))
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
	return mixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot), nil
}
