package testutil

import (
	"fmt"
)

// CheckBit checks if a bit in a bit field is one.
func CheckBit(bitfield []byte, index int) (bool, error) {
	chunkLocation := (index + 1) / 8
	indexLocation := (index + 1) % 8
	if indexLocation == 0 {
		indexLocation = 8
	} else {
		chunkLocation++
	}

	if chunkLocation > len(bitfield) {
		return false, fmt.Errorf("index out of range for bitfield: length: %d, position: %d ",
			len(bitfield), chunkLocation-1)
	}

	field := bitfield[chunkLocation-1] >> (8 - uint(indexLocation))
	return field%2 != 0, nil
}

// BitSetCount counts the number of 1s in a byte using the following algo:
// https://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
func BitSetCount(bytes []byte) int {
	var total int
	for _, b := range bytes {
		b = (b & 0x55) + ((b >> 1) & 0x55)
		b = (b & 0x33) + ((b >> 2) & 0x33)
		total += int((b + (b >> 4)) & 0xF)
	}
	return total
}

// BitLength returns the length of the bitfield for a giben number of attesters in bytes.
func BitLength(b int) int {
	return (b + 7) / 8
}
