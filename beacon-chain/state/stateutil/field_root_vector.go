package stateutil

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
)

func ArraysRoot(input [][]byte, length uint64) ([32]byte, error) {
	hashFunc := hash.CustomSHA256Hasher()

	leaves := make([][32]byte, length)
	for i, chunk := range input {
		copy(leaves[i][:], chunk)
	}
	res, err := merkleize(leaves, length, hashFunc)
	if err != nil {
		return [32]byte{}, err
	}

	return res, nil
}

func merkleize(leaves [][32]byte, length uint64,
	hasher func([]byte) [32]byte) ([32]byte, error) {
	if len(leaves) == 0 {
		return [32]byte{}, errors.New("zero leaves provided")
	}
	if len(leaves) == 1 {
		return leaves[0], nil
	}
	hashLayer := leaves
	layers := make([][][32]byte, ssz.Depth(length)+1)
	layers[0] = hashLayer
	var err error
	_, hashLayer, err = MerkleizeTrieLeaves(layers, hashLayer, hasher)
	if err != nil {
		return [32]byte{}, err
	}
	root := hashLayer[0]
	return root, nil
}
