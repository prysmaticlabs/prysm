// Package bytesutil defines helper methods for converting integers to byte slices.
package bytesutil

import (
	"encoding/binary"
)

// Bytes1 returns integer x to bytes in little-endian format, x.to_bytes(1, 'big').
func Bytes1(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:1]
}

// Bytes2 returns integer x to bytes in little-endian format, x.to_bytes(2, 'big').
func Bytes2(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:2]
}

// Bytes3 returns integer x to bytes in little-endian format, x.to_bytes(3, 'big').
func Bytes3(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:3]
}

// Bytes4 returns integer x to bytes in little-endian format, x.to_bytes(4, 'big').
func Bytes4(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:4]
}

// Bytes8 returns integer x to bytes in little-endian format, x.to_bytes(8, 'big').
func Bytes8(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes
}

// FromBytes8 returns an integer which is stored in the little-endian format(8, 'big')
// from a byte array.
func FromBytes8(x []byte) uint64 {
	return binary.LittleEndian.Uint64(x)
}

// LowerThan returns true if byte slice x is lower than byte slice y. (little-endian format)
// This is used in spec to compare winning block root hash.
// Mentioned in spec as "ties broken by favoring lower `shard_block_root` values".
func LowerThan(x []byte, y []byte) bool {
	for i, b := range x {
		if b > y[i] {
			return false
		}
	}
	return true
}

// ToBytes32 is a convenience method for converting a byte slice to a fix
// sized 32 byte array. This method will truncate the input if it is larger
// than 32 bytes.
func ToBytes32(x []byte) [32]byte {
	var y [32]byte
	copy(y[:], x)
	return y
}

// ToBytes96 is a convenience method for converting a byte slice to a fix
// sized 96 byte array. This method will truncate the input if it is larger
// than 96 bytes.
func ToBytes96(x []byte) [96]byte {
	var y [96]byte
	copy(y[:], x)
	return y
}

// ToBytes48 is a convenience method for converting a byte slice to a fix
// sized 48 byte array. This method will truncate the input if it is larger
// than 48 bytes.
func ToBytes48(x []byte) [48]byte {
	var y [48]byte
	copy(y[:], x)
	return y
}

// Xor xors the bytes in x and y and returns the result.
func Xor(x []byte, y []byte) []byte {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	var result []byte
	for i := 0; i < n; i++ {
		result = append(result, x[i]^y[i])
	}
	return result
}

// Trunc truncates the byte slices to 12 bytes.
func Trunc(x []byte) []byte {
	if len(x) > 12 {
		return x[:12]
	}
	return x
}
