// Package bytes defines helper methods for converting integers to byte slices.
package bytes

import (
	"encoding/binary"
)

func Bytes1(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, x)
	return bytes[7:]
}

func Bytes2(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, x)
	return bytes[6:]
}

func Bytes3(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, x)
	return bytes[5:]
}

func Bytes4(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, x)
	return bytes[4:]
}

func Bytes8(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, x)
	return bytes
}
