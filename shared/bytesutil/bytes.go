// Package bytesutil defines helper methods for converting integers to byte slices.
package bytesutil

import (
	"encoding/binary"
)

// ToBytes returns integer x to bytes in little-endian format at the specified length.
// Spec pseudocode definition:
//   def int_to_bytes(integer: int, length: int) -> bytes:
//     return integer.to_bytes(length, 'little')
func ToBytes(x uint64, length int) []byte {
	makeLength := length
	if length < 8 {
		makeLength = 8
	}
	bytes := make([]byte, makeLength)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:length]
}

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

// Bytes4 returns integer x to bytes in little-endian format, x.to_bytes(4, 'little').
func Bytes4(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:4]
}

// Bytes8 returns integer x to bytes in little-endian format, x.to_bytes(8, 'little').
func Bytes8(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes
}

// Bytes32 returns integer x to bytes in little-endian format, x.to_bytes(8, 'little').
func Bytes32(x uint64) []byte {
	bytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes
}

// FromBytes4 returns an integer which is stored in the little-endian format(4, 'little')
// from a byte array.
func FromBytes4(x []byte) uint64 {
	empty4bytes := make([]byte, 4)
	return binary.LittleEndian.Uint64(append(x[:4], empty4bytes...))
}

// FromBytes8 returns an integer which is stored in the little-endian format(8, 'little')
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

// ToBytes8 is a convenience method for converting a byte slice to a fix
// sized 8 byte array. This method will truncate the input if it is larger
// than 8 bytes.
func ToBytes8(x []byte) [8]byte {
	var y [8]byte
	copy(y[:], x)
	return y
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

// ToLowInt64 returns the lowest 8 bytes interpreted as little endian.
func ToLowInt64(x []byte) int64 {
	if len(x) > 8 {
		x = x[:8]
	}
	return int64(binary.LittleEndian.Uint64(x))
}
