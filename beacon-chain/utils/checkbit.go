package utils

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
func BitSetCount(v byte) byte {
	v = (v & 0x55) + ((v >> 1) & 0x55)
	v = (v & 0x33) + ((v >> 2) & 0x33)
	return (v + (v >> 4)) & 0xF
}

// BitLength returns the length of the bitfield for a given number of attesters in bytes.
func BitLength(b int) int {
	return (b + 7) / 8
}
