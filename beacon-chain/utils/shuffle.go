// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// Permuted Index returns pseudo random permutation of the active index.
func PermutedIndex(index uint64, listSize uint64, seed common.Hash) (uint64, error) {
	if index >= listSize {
		err := errors.New("index is greater or equal than listSize")
		return 0, err
	}

	if listSize > mathutil.PowerOf2(40) {
		err := errors.New("listSize is greater than 2**40")
		return 0, err
	}

	for round := 0; round < 90; round++ {

		hashedValue := hashutil.Hash(append(seed[:], bytesutil.Bytes1(uint64(round))...))
		pivot := bytesutil.FromBytes8(hashedValue[:8]) % listSize
		flip := (pivot + (listSize - index)) % listSize
		position := index
		if flip > index {
			position = flip
		}
		valuesToBeHashed := append(seed[:], bytesutil.Bytes1(uint64(round))...)
		valuesToBeHashed = append(valuesToBeHashed, bytesutil.Bytes4(position/256)...)
		source := hashutil.Hash(valuesToBeHashed)
		byteV := source[(int(position)%256)/8]
		bitV := (byteV >> (position % 8)) % 2
		if bitV == 1 {
			index = flip
		}
	}
	return index, nil
}

// SplitIndices splits a list into n pieces.
func SplitIndices(l []uint64, n uint64) [][]uint64 {
	var divided [][]uint64
	var lSize = uint64(len(l))
	for i := uint64(0); i < n; i++ {
		start := lSize * i / n
		end := lSize * (i + 1) / n
		divided = append(divided, l[start:end])
	}
	return divided
}
