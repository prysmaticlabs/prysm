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
