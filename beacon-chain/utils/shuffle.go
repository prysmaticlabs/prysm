// Package utils defines utility functions for the beacon-chain.
package utils

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const seedSize = int8(32)
const roundSize = int8(1)
const positionWindowSize = int8(4)
const pivotViewSize = seedSize + roundSize
const totalSize = seedSize + roundSize + positionWindowSize
const maxShuffleListSize = 1 << 40

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
		start := SplitOffset(lSize, n, i)
		end := SplitOffset(lSize, n, i+1)
		divided = append(divided, l[start:end])
	}
	return divided
}

// ShuffledIndex returns `p(index)` in a pseudorandom permutation `p` of `0...list_size - 1` with ``seed`` as entropy.
// We utilize 'swap or not' shuffling in this implementation; we are allocating the memory with the seed that stays
// constant between iterations instead of reallocating it each iteration as in the spec. This implementation is based
// on the original implementation from protolambda, https://github.com/protolambda/eth2-shuffle
//
// Spec pseudocode definition:
//   def get_shuffled_index(index: ValidatorIndex, index_count: int, seed: Bytes32) -> ValidatorIndex:
//     """
//     Return the shuffled validator index corresponding to ``seed`` (and ``index_count``).
//     """
//     assert index < index_count
//     assert index_count <= 2**40
//     # Swap or not (https://link.springer.com/content/pdf/10.1007%2F978-3-642-32009-5_1.pdf)
//     # See the 'generalized domain' algorithm on page 3
//     for round in range(SHUFFLE_ROUND_COUNT):
//         pivot = bytes_to_int(hash(seed + int_to_bytes(round, length=1))[0:8]) % index_count
//         flip = (pivot + index_count - index) % index_count
//         position = max(index, flip)
//         source = hash(seed + int_to_bytes(round, length=1) + int_to_bytes(position // 256, length=4))
//         byte = source[(position % 256) // 8]
//         bit = (byte >> (position % 8)) % 2
//         index = flip if bit else index
//     return index
func ShuffledIndex(index uint64, indexCount uint64, seed [32]byte) (uint64, error) {
	if params.BeaconConfig().ShuffleRoundCount == 0 {
		return index, nil
	}
	if index >= indexCount {
		return 0, fmt.Errorf("input index %d out of bounds: %d",
			index, indexCount)
	}
	if indexCount > maxShuffleListSize {
		return 0, fmt.Errorf("list size %d out of bounds",
			indexCount)
	}
	buf := make([]byte, totalSize, totalSize)
	// Seed is always the first 32 bytes of the hash input, we never have to change this part of the buffer.
	copy(buf[:32], seed[:])
	for round := uint8(0); round < uint8(params.BeaconConfig().ShuffleRoundCount); round++ {
		buf[seedSize] = round
		hash := hashutil.Hash(buf[:pivotViewSize])
		hash8 := hash[:8]
		pivot := bytesutil.FromBytes8(hash8) % indexCount
		flip := (pivot + indexCount - index) % indexCount
		// spec: position = max(index, flip)
		// Consider every pair only once by picking the highest pair index to retrieve randomness.
		position := index
		if flip > position {
			position = flip
		}
		// Add position except its last byte to []buf for randomness,
		// it will be used later to select a bit from the resulting hash.
		position4bytes := bytesutil.Bytes4(position >> 8)
		copy(buf[pivotViewSize:], position4bytes[:])
		source := hashutil.Hash(buf)
		// Effectively keep the first 5 bits of the byte value of the position,
		// and use it to retrieve one of the 32 (= 2^5) bytes of the hash.
		byteV := source[(position&0xff)>>3]
		// Using the last 3 bits of the position-byte, determine which bit to get from the hash-byte (note: 8 bits = 2^3)
		bitV := (byteV >> (position & 0x7)) & 0x1
		// index = flip if bit else index
		if bitV == 1 {
			index = flip
		}
	}
	return index, nil
}

// SplitOffset returns (listsize * index) / chunks
//
// Spec pseudocode definition:
// def get_split_offset(list_size: int, chunks: int, index: int) -> int:
//     """
//     Returns a value such that for a list L, chunk count k and index i,
//     split(L, k)[i] == L[get_split_offset(len(L), k, i): get_split_offset(len(L), k, i+1)]
//     """
//     return (list_size * index) // chunks
func SplitOffset(listSize uint64, chunks uint64, index uint64) uint64 {
	return (listSize * index) / chunks
}
