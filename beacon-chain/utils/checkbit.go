package utils

import (
	"errors"
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

	if len(bitfield) < chunkLocation {
		return false, errors.New("attester index does not exist")
	}

	field := bitfield[chunkLocation-1] >> (8 - uint(indexLocation))
	if field%2 != 0 {
		return true, nil
	}
	return false, nil
}

// BitSetCount counts the number of 1s in a byte using the following algo:
// https://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
func BitSetCount(v byte) byte {
	v = (v & 0x55) + ((v >> 1) & 0x55)
	v = (v & 0x33) + ((v >> 2) & 0x33)
	return (v + (v >> 4)) & 0xF
}
