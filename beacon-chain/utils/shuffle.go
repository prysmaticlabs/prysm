// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

func GetPermutedIndex(index uint64, listSize uint64, seed common.Hash) (uint64, error) {
	if index >= listSize {
		err := errors.New("index is greater or equal than listSize")
		return 0, err
	}

	if listSize > mathutil.PowerOf2(40) {
		err := errors.New("listSize is greater than 2**40")
		return 0, err
	}

	bs4 := make([]byte, 4)
	bs2 := make([]byte, 2)

	for round := 0; round < 90; round++ {
		binary.LittleEndian.PutUint16(bs2[:], uint16(round))
		bs1 := bs2[:1]
		hashedValue := hashutil.Hash(append(seed[:], bs1...))
		hashedValue8 := hashedValue[:8]
		pivot := binary.LittleEndian.Uint64(hashedValue8[:]) % listSize
		flip := (pivot - index) % listSize
		position := index
		if flip > position {
			position = flip
		}
		positionVal := uint32(position/256)
		binary.LittleEndian.PutUint32(bs4[:], positionVal)
		bs := append(bs1, bs4...)
		source := hashutil.Hash(append(seed[:], bs...))
		positionIndex := mathutil.CeilDiv8(int(position) % 256)
		byteV := source[positionIndex]
		bitV := (byteV >> (position % 8)) % 2
		if bitV == 1 {
			index = flip
		}
	}
	return index, nil
}

// ShuffleIndices returns a list of pseudorandomly sampled
// indices. This is used to shuffle validators on ETH2.0 beacon chain.
func ShuffleIndices(seed common.Hash, indicesList []uint64) ([]uint64, error) {
	var permutedIndicesList []uint64
	for _, index := range indicesList {
		permutedIndex, err := GetPermutedIndex(index, uint64(len(indicesList)), seed)
		if err != nil {
			return nil,err
		}
		permutedIndicesList = append(permutedIndicesList, permutedIndex)
	}
	return permutedIndicesList, nil
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
