package shared

import (
	"math"

	"github.com/steakknife/hamming"
)

// CheckBit checks if a bit in a bit field is one.
func CheckBit(bitfield []byte, index int) bool {
	chunkLocation := (index + 1) / 8
	indexLocation := (index + 1) % 8
	if indexLocation == 0 {
		indexLocation = 8
	} else {
		chunkLocation++
	}

	field := bitfield[chunkLocation-1] >> (8 - uint(indexLocation))
	return field%2 != 0
}

// BitSetCount counts the number of 1s in a byte using the following algo:
// https://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
func BitSetCount(bytes []byte) int {
	var total int
	for _, b := range bytes {
		total += hamming.CountBitsByte(b)
	}
	return total
}

// BitLength returns the length of the bitfield in bytes.
func BitLength(b int) int {
	return (b + 7) / 8
}

// SetBitfield takes an index and returns bitfield with the index flipped.
func SetBitfield(index int) []byte {
	chunkLocation := index / 8
	indexLocation := math.Pow(2, 8-float64(index%8))
	var bitfield []byte

	for i := 0; i < chunkLocation; i++ {
		bitfield = append(bitfield, byte(0))
	}
	bitfield = append(bitfield, byte(indexLocation))

	return bitfield
}
