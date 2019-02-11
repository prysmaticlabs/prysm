// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	//"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// ShuffleIndices returns a list of pseudorandomly sampled
// indices. This is used to shuffle validators on ETH2.0 beacon chain.
func SwapOrNotShuffle(seed common.Hash, indicesList []uint64) ([]uint64, error) {
	listSize := len(indicesList)
	// Each entropy is consumed from the seed in randBytes chunks.
	randBytes := params.BeaconConfig().RandBytes

	maxValidatorsPerRandBytes := params.BeaconConfig().MaxNumLog2Validators / randBytes
	upperBound := 1<<(randBytes*maxValidatorsPerRandBytes) - 1
	// Since we are consuming randBytes of entropy at a time in the loop,
	// we have a bias at 2**24, this check defines our max list size and is used to remove the bias.
	// more info on modulo bias: https://stackoverflow.com/questions/10984974/why-do-people-say-there-is-modulo-bias-when-using-a-random-number-generator.
	if listSize >= upperBound {
		return nil, errors.New("input list exceeded upper bound and reached modulo bias")
	}

	var round uint64
	for round = 0; round < 90; round++ {
		hashBytes := make([]byte, 0)
        bs1 := make([]byte, 1)
		binary.LittleEndian.PutUint64(bs1, round)
		num := uint64(math.Floor(float64((listSize + 255) / 256)))
		var i uint64
		for i = 0; i < num; i++ {
			bs4 := make([]byte, 4)
			binary.LittleEndian.PutUint64(bs4, i)
			bs := append(bs1, bs4...)
			hash := hashutil.Hash(append(seed[:], bs...))
			hashBytes = append(hashBytes, hash[:]...)
		}

		hash := hashutil.Hash(append(seed[:], bs1...))
		hashFromBytes := binary.LittleEndian.Uint64(hash[:])
		pivot := hashFromBytes % uint64(listSize)

		powersOfTwo := []uint64{1, 2, 4, 8, 16, 32, 64, 128}

		for i, index := range(indicesList) {
			flip := (pivot - index) % uint64(listSize)
			var hashPos uint64
			if index > flip {
				hashPos = index
			} else {
				hashPos = flip
			}
			hashBytesIndex := uint64(math.Floor(float64((hashPos / 8))))
			h := uint64(hashBytes[hashBytesIndex])
			p := powersOfTwo[hashPos % 8]
			if  h & p != 0 {
				indicesList[i] = flip
			}
			
		}
	}
	return indicesList, nil
}

func ShuffleIndices(seed common.Hash, indicesList []uint64) ([]uint64, error) {
	// Each entropy is consumed from the seed in randBytes chunks.
	randBytes := params.BeaconConfig().RandBytes

	if listSize > mathutil.PowerOf2(40) {
		err := errors.New("listSize is greater than 2**40")
		return 0, err
	}

	bs4 := make([]byte, 4)
	buf := new(bytes.Buffer)

	for round := 0; round < 90; round++ {
		if err := binary.Write(buf, binary.LittleEndian, uint8(round)); err != nil {
			return 0, err
		}
		bs1 := buf.Bytes()
		hashedValue := hashutil.Hash(append(seed[:], bs1...))
		hashedValue8 := hashedValue[:8]
		pivot := binary.LittleEndian.Uint64(hashedValue8[:]) % listSize
		flip := (pivot - index) % listSize
		position := index
		if flip > position {
			position = flip
		}
		positionVal := uint32(math.Floor(float64(position / 256)))
		binary.LittleEndian.PutUint32(bs4[:], positionVal)
		bs := append(bs1, bs4...)
		source := hashutil.Hash(append(seed[:], bs...))
		positionIndex := uint64(mathutil.CeilDiv8(int(position) % 256))
		byteV := source[positionIndex]
		bitV := (byteV >> (position % 8)) % 2
		if bitV == 1 {
			index = flip
		}
	}
	return index, nil
}

func ShuffleIndices(seed common.Hash, indicesList []uint64) ([]uint64, error) {
	var permutedIndicesList []uint64
	for _, index := range indicesList {
		permutedIndex, err := GetPermutedIndex(index, uint64(len(indicesList)), seed)
		if err != nil {
			return nil, errors.New("GetPermutedIndex error")
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
