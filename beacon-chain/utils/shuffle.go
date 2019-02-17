// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"encoding/binary"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	//"github.com/prysmaticlabs/prysm/shared/params"
)

// ShuffleIndices returns a list of pseudorandomly sampled
// indices. This is used to shuffle validators on ETH2.0 beacon chain.
func ShuffleIndices(seed common.Hash, indicesList []uint64) ([]uint64, error) {
	var hashBytes []byte
	bs1 := make([]byte, 8)
	bs4 := make([]byte, 8)
	listSize := len(indicesList)
	num := int(math.Floor(float64((listSize + 255) / 256)))
	powersOfTwo := []uint64{1, 2, 4, 8, 16, 32, 64, 128}

	for round := 0; round < 90; round++ {
		binary.LittleEndian.PutUint64(bs1[:], uint64(round))

		for i := 0; i < num; i++ {
			binary.LittleEndian.PutUint64(bs4[:], uint64(i))
			bs := append(bs1, bs4...)
			hash := hashutil.Hash(append(seed[:], bs...))
			hashBytes = append(hashBytes, hash[:]...)
		}

		hash := hashutil.Hash(append(seed[:], bs1...))
		hashFromBytes := binary.LittleEndian.Uint64(hash[:])
		pivot := hashFromBytes % uint64(listSize)

		for i, index := range indicesList {
			flip := (pivot - index) % uint64(listSize)
			var hashPos uint64
			if index > flip {
				hashPos = index
			} else {
				hashPos = flip
			}
			hashBytesIndex := int(math.Floor((float64(hashPos) / 8)))
			hByte := hashBytes[hashBytesIndex]
			hInt := uint64(hByte)
			p := powersOfTwo[hashPos%8]
			if hInt&p != 0 {
				indicesList[i] = flip
			}
		}
	}
	return indicesList, nil
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
