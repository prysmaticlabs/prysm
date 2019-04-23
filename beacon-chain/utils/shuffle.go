// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const hSeedSize = int8(32)
const hRoundSize = int8(1)
const hPositionWindowSize = int8(4)
const hPivotViewSize = hSeedSize + hRoundSize
const hTotalSize = hSeedSize + hRoundSize + hPositionWindowSize

// ShuffleIndices returns a list of pseudorandomly sampled
// indices. This is used to shuffle validators on ETH2.0 beacon chain.
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

// PermutedIndex Return `p(index)` in a pseudorandom permutation `p` of `0...list_size - 1` with ``seed`` as entropy.
//     Utilizes 'swap or not' shuffling
//
// Spec pseudocode definition:
// def get_permuted_index(index: int, list_size: int, seed: Bytes32) -> int:
//     """
//     Return `p(index)` in a pseudorandom permutation `p` of `0...list_size - 1` with ``seed`` as entropy.
//     Utilizes 'swap or not' shuffling found in
//     https://link.springer.com/content/pdf/10.1007%2F978-3-642-32009-5_1.pdf
//     See the 'generalized domain' algorithm on page 3.
//     """
//     assert index < list_size
//     assert list_size <= 2**40
//     for round in range(SHUFFLE_ROUND_COUNT):
//         pivot = bytes_to_int(hash(seed + int_to_bytes1(round))[0:8]) % list_size
//         flip = (pivot - index) % list_size
//         position = max(index, flip)
//         source = hash(seed + int_to_bytes1(round) + int_to_bytes4(position // 256))
//         byte = source[(position % 256) // 8]
//         bit = (byte >> (position % 8)) % 2
//         index = flip if bit else index
//     return index
func PermutedIndex(index uint64, listSize uint64, seed [32]byte) (uint64, error) {
	if params.BeaconConfig().ShuffleRoundCount == 0 {
		return index, nil
	}
	// spec: assert index < list_size
	if index >= listSize {
		return 0, fmt.Errorf("input index %d out of bounds: %d",
			index, listSize)
	}
	// spec: assert list_size <= 2**40
	if listSize > uint64(math.Pow(2, 40)) {
		return 0, fmt.Errorf("list size %d out of bounds",
			listSize)
	}
	buf := make([]byte, hTotalSize, hTotalSize)
	// Seed is always the first 32 bytes of the hash input, we never have to change this part of the buffer.
	copy(buf[:32], seed[:])
	for round := uint8(0); round < uint8(params.BeaconConfig().ShuffleRoundCount); round++ {
		buf[hSeedSize] = round
		hash := hashutil.Hash(buf[:hPivotViewSize])
		hash8 := hash[:8]
		pivot := BytesToInt(hash8) % listSize
		flip := (pivot - index) % listSize
		// spec: position = max(index, flip)
		// Why? Don't do double work: we consider every pair only once.
		// (Otherwise we would swap it back in place)
		// Pick the highest index of the pair as position to retrieve randomness with.
		position := index
		if flip > position {
			position = flip
		}
		// spec: source = hash(seed + int_to_bytes1(round) + int_to_bytes4(position // 256))
		// - seed is still in 0:32 (excl., 32 bytes)
		// - round number is still in 32
		// - mix in the position for randomness, except the last byte of it,
		//     which will be used later to select a bit from the resulting hash.
		p4b := IntToBytes4(position >> 8)
		copy(buf[hPivotViewSize:], p4b[:])
		source := hashutil.Hash(buf)
		// spec: byte = source[(position % 256) // 8]
		// Effectively keep the first 5 bits of the byte value of the position,
		//  and use it to retrieve one of the 32 (= 2^5) bytes of the hash.
		byteV := source[(position&0xff)>>3]
		// Using the last 3 bits of the position-byte, determine which bit to get from the hash-byte (8 bits, = 2^3)
		// spec: bit = (byte >> (position % 8)) % 2
		bitV := (byteV >> (position & 0x7)) & 0x1
		//index = flip if bit else index
		if bitV == 1 {
			index = flip
		}

	}
	return index, nil
}

// BytesToInt returns the uint64 representation of a byte slice.
//
// Spec pseudocode definition:
// def bytes_to_int(data: bytes) -> int:
//     return int.from_bytes(data, 'little')
func BytesToInt(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data)
}

// IntToBytes1 returns the byte little endian representation of an uint64.
// int_to_bytes1(x): return x.to_bytes(1, 'little')
func IntToBytes1(x uint64) byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, x)
	return b[0]
}

// IntToBytes2 returns the 2 bytes little endian representation of an uint64.
func IntToBytes2(x uint64) [2]byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, x)
	var arr [2]byte
	copy(arr[:], b[:2])
	return arr
}

// IntToBytes3 returns the 3 bytes little endian representation of an uint64.
func IntToBytes3(x uint64) [3]byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, x)
	var arr [3]byte
	copy(arr[:], b[:3])
	return arr
}

// IntToBytes4 returns the 4 bytes little endian representation of an uint64.
func IntToBytes4(x uint64) [4]byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, x)
	var arr [4]byte
	copy(arr[:], b[:4])
	return arr
}

// IntToBytes8 returns the 8 bytes little endian representation of an uint64.
func IntToBytes8(x uint64) [8]byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, x)
	var arr [8]byte
	copy(arr[:], b[:8])
	return arr
}

// IntToBytes32 returns the 4 bytes little endian representation of an uint64.
func IntToBytes32(x uint64) [32]byte {
	b := make([]byte, 32)
	binary.LittleEndian.PutUint64(b, x)
	var arr [32]byte
	copy(arr[:], b[:32])
	return arr
}

// IntToBytes48 returns the 4 bytes little endian representation of an uint64.
func IntToBytes48(x uint64) [48]byte {
	b := make([]byte, 48)
	binary.LittleEndian.PutUint64(b, x)
	var arr [48]byte
	copy(arr[:], b[:48])
	return arr
}

// IntToBytes96 returns the 4 bytes little endian representation of an uint64.
func IntToBytes96(x uint64) [96]byte {
	b := make([]byte, 96)
	binary.LittleEndian.PutUint64(b, x)
	var arr [96]byte
	copy(arr[:], b[:96])
	return arr
}
