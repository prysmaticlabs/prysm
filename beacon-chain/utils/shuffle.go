// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"encoding/binary"
	"errors"
	"math"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

	
	for round := 0; round < 90; round++ {
		var hashBytes []byte
		bs1 := make([]byte, 8)
		binary.LittleEndian.PutUint64(bs1, uint64(round))
		fmt.Printf("bs1 is %v\n", bs1)
		num := int(math.Floor(float64((listSize + 255) / 256)))
		fmt.Printf("num is %v\n", num)
		for i := 0; i < num; i++ {
			bs4 := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs4, uint64(i))
			fmt.Printf("bs4 is %v\n", bs4)
			bs := append(bs1, bs4...)
			fmt.Printf("bs is %v\n", bs)
			hash := hashutil.Hash(append(seed[:], bs...))
			fmt.Printf("hash is %v\n", hash)
			hashBytes = append(hashBytes, hash[:]...)
			fmt.Printf("hashBytes is %v\n", hashBytes)

		}

		hash := hashutil.Hash(append(seed[:], bs1...))
		fmt.Printf("hash is %v\n", hash)
		hashFromBytes := binary.LittleEndian.Uint64(hash[:])
		fmt.Printf("hashFromBytes is %v\n", hashFromBytes)
		pivot := hashFromBytes % uint64(listSize)
		fmt.Printf("pivot is %v\n", pivot)

		powersOfTwo := []uint64{1, 2, 4, 8, 16, 32, 64, 128}

		for i, index := range(indicesList) {
			flip := (pivot - index) % uint64(listSize)
			fmt.Printf("flip is %v\n", flip)
			var hashPos uint64
			if index > flip {
				hashPos = index
			} else {
				hashPos = flip
			}
			fmt.Printf("hashPos is %v\n", hashPos)
			hashBytesIndex := int(math.Floor((float64(hashPos) / 8)))
			fmt.Printf("hashBytesIndex is %v\n", hashBytesIndex)
			hByte := hashBytes[hashBytesIndex]
			fmt.Printf("hByte is %v\n", hByte)
			hInt := uint64(hByte)
			fmt.Printf("hByte is %v\n", hByte)
			p := powersOfTwo[hashPos % 8]
			fmt.Printf("p is %v\n", p)
			if  hInt & p != 0 {
				indicesList[i] = flip
			}
			
		}
	}
	return indicesList, nil
}

func ShuffleIndices(seed common.Hash, indicesList []uint64) ([]uint64, error) {
	// Each entropy is consumed from the seed in randBytes chunks.
	randBytes := params.BeaconConfig().RandBytes

	maxValidatorsPerRandBytes := params.BeaconConfig().MaxNumLog2Validators / randBytes
	upperBound := 1<<(randBytes*maxValidatorsPerRandBytes) - 1
	// Since we are consuming randBytes of entropy at a time in the loop,
	// we have a bias at 2**24, this check defines our max list size and is used to remove the bias.
	// more info on modulo bias: https://stackoverflow.com/questions/10984974/why-do-people-say-there-is-modulo-bias-when-using-a-random-number-generator.
	if len(indicesList) >= upperBound {
		return nil, errors.New("input list exceeded upper bound and reached modulo bias")
	}

	// Rehash the seed to obtain a new pattern of bytes.
	hashSeed := hashutil.Hash(seed[:])
	totalCount := len(indicesList)
	index := 0
	for index < totalCount-1 {
		// Iterate through the hashSeed bytes in chunks of size randBytes.
		for i := 0; i < 32-(32%int(randBytes)); i += int(randBytes) {
			// Determine the number of indices remaining and exit if last index reached.
			remaining := totalCount - index
			if remaining == 1 {
				break
			}
			// Read randBytes of hashSeed as a maxValidatorsPerRandBytes x randBytes big-endian integer.
			randChunk := hashSeed[i : i+int(randBytes)]
			var randValue int
			for j := 0; j < int(randBytes); j++ {
				randValue |= int(randChunk[j])
			}

			// Sample values greater than or equal to sampleMax will cause
			// modulo bias when mapped into the remaining range.
			randMax := upperBound - upperBound%remaining

			// Perform swap if the consumed entropy will not cause modulo bias.
			if randValue < randMax {
				// Select replacement index from the current index.
				replacementIndex := (randValue % remaining) + index
				indicesList[index], indicesList[replacementIndex] = indicesList[replacementIndex], indicesList[index]
				index++
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
