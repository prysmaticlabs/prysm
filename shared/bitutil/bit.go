package bitutil

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/steakknife/hamming"
)

// SetBitfield takes an index and returns bitfield with the index flipped.
func SetBitfield(index int, committeeLength int) []byte {
	chunkLocation := index / 8
	indexLocation := mathutil.PowerOf2(uint64(7 - (index % 8)))
	var bitfield []byte

	for i := 0; i < chunkLocation; i++ {
		bitfield = append(bitfield, byte(0))
	}
	bitfield = append(bitfield, byte(indexLocation))

	for len(bitfield) < committeeLength {
		bitfield = append(bitfield, byte(0))
	}

	return bitfield
}

// CheckBit checks if a bit in a bit field (small endian) is one.
func CheckBit(bitfield []byte, index int) (bool, error) {
	if len(bitfield) < mathutil.CeilDiv8(index) {
		return false, fmt.Errorf(
			"wanted index is out of bound of bitfield length %d, got: %d",
			mathutil.CeilDiv8(index),
			len(bitfield))
	}
	bitExists := BitfieldBit(bitfield, index) == 1
	return bitExists, nil
}

// BitSetCount counts the number of 1s in a byte using Hamming weight.
// See: https://en.wikipedia.org/wiki/Hamming_weight
func BitSetCount(b []byte) int {
	return hamming.CountBitsBytes(b)
}

// BitLength returns the length of the bitfield in bytes.
func BitLength(b int) int {
	return (b + 7) / 8
}

// FillBitfield returns a bitfield of length `count`, all set to true.
func FillBitfield(count int) []byte {
	numChunks := count/8 + 1
	bitfield := make([]byte, numChunks)
	for i := 0; i < numChunks; i++ {
		if i+1 == numChunks {
			bitfield[i] = fillNBits(uint64(count % 8))
		} else {
			bitfield[i] = byte(8)
		}
	}

	return bitfield
}

func fillNBits(numBits uint64) byte {
	result := byte(0)
	for i := uint64(0); i < numBits; i++ {
		result = fillBit(result, i)
	}

	return result
}

func fillBit(target byte, index uint64) byte {
	bitShift := 7 - index
	return target ^ (1 << bitShift)
}

// BitfieldBit extracts the bit in bitfield at position i.
// Spec pseudocode definition:
//   def get_bitfield_bit(bitfield: bytes, i: int) -> int:
//     """
//     Extract the bit in ``bitfield`` at position ``i``.
//     """
//     return (bitfield[i // 8] >> (i % 8)) % 2
func BitfieldBit(bitfield []byte, i int) byte {
	//take the relevant byte in bitfield in order to get i value
	rb := bitfield[i>>3]
	//shift the byte till i is its first bit
	fb := rb >> (uint(i) & 0x7)
	//zero the byte value except its first bit
	b := fb & 0x1
	return b
}
