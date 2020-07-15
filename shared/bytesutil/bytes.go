// Package bytesutil defines helper methods for converting integers to byte slices.
package bytesutil

import (
	"encoding/binary"
	"errors"
	"math/bits"
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

// Bytes1 returns integer x to bytes in little-endian format, x.to_bytes(1, 'little').
func Bytes1(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:1]
}

// Bytes2 returns integer x to bytes in little-endian format, x.to_bytes(2, 'little').
func Bytes2(x uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes[:2]
}

// Bytes3 returns integer x to bytes in little-endian format, x.to_bytes(3, 'little').
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

// Bytes32 returns integer x to bytes in little-endian format, x.to_bytes(32, 'little').
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

// ToBytes4 is a convenience method for converting a byte slice to a fix
// sized 4 byte array. This method will truncate the input if it is larger
// than 4 bytes.
func ToBytes4(x []byte) [4]byte {
	var y [4]byte
	copy(y[:], x)
	return y
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

// ToBytes64 is a convenience method for converting a byte slice to a fix
// sized 64 byte array. This method will truncate the input if it is larger
// than 64 bytes.
func ToBytes64(x []byte) [64]byte {
	var y [64]byte
	copy(y[:], x)
	return y
}

// ToBool is a convenience method for converting a byte to a bool.
// This method will use the first bit of the 0 byte to generate the returned value.
func ToBool(x byte) bool {
	return x&1 == 1
}

// FromBytes2 returns an integer which is stored in the little-endian format(2, 'little')
// from a byte array.
func FromBytes2(x []byte) uint16 {
	return binary.LittleEndian.Uint16(x[:2])
}

// FromBool is a convenience method for converting a bool to a byte.
// This method will use the first bit to generate the returned value.
func FromBool(x bool) byte {
	if x {
		return 1
	}
	return 0
}

// FromBytes32 is a convenience method for converting a fixed-size byte array
// to a byte slice.
func FromBytes32(x [32]byte) []byte {
	return x[:]
}

// FromBytes48 is a convenience method for converting a fixed-size byte array
// to a byte slice.
func FromBytes48(x [48]byte) []byte {
	return x[:]
}

// FromBytes48Array is a convenience method for converting an array of
// fixed-size byte arrays to an array of byte slices.
func FromBytes48Array(x [][48]byte) [][]byte {
	y := make([][]byte, len(x))
	for i := range x {
		y[i] = x[i][:]
	}
	return y
}

// Trunc truncates the byte slices to 6 bytes.
func Trunc(x []byte) []byte {
	if len(x) > 6 {
		return x[:6]
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

// SafeCopyBytes will copy and return a non-nil byte array, otherwise it returns nil.
func SafeCopyBytes(cp []byte) []byte {
	if cp != nil {
		copied := make([]byte, len(cp))
		copy(copied, cp)
		return copied
	}
	return nil
}

// Copy2dBytes will copy and return a non-nil 2d byte array, otherwise it returns nil.
func Copy2dBytes(ary [][]byte) [][]byte {
	if ary != nil {
		copied := make([][]byte, len(ary))
		for i, a := range ary {
			copied[i] = SafeCopyBytes(a)
		}
		return copied
	}
	return nil
}

// ReverseBytes32Slice will reverse the provided slice's order.
func ReverseBytes32Slice(arr [][32]byte) [][32]byte {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

// PadTo pads a byte slice to the given size. If the byte slice is larger than the given size, the
// original slice is returned.
func PadTo(b []byte, size int) []byte {
	if len(b) > size {
		return b
	}
	return append(b, make([]byte, size-len(b))...)
}

// SetBit sets the index `i` of bitlist `b` to 1.
// It grows and returns a longer bitlist with 1 set
// if index `i` is out of range.
func SetBit(b []byte, i int) []byte {
	if i >= len(b)*8 {
		h := (i + (8 - i%8)) / 8
		b = append(b, make([]byte, h-len(b))...)
	}

	bit := uint8(1 << (i % 8))
	b[i/8] |= bit
	return b
}

// ClearBit clears the index `i` of bitlist `b`.
// Returns the original bitlist if the index `i`
// is out of range.
func ClearBit(b []byte, i int) []byte {
	if i >= len(b)*8 {
		return b
	}

	bit := uint8(1 << (i % 8))
	b[i/8] &^= bit
	return b
}

// MakeEmptyBitlists returns an empty bitlist with
// input size `i`.
func MakeEmptyBitlists(i int) []byte {
	return make([]byte, (i+(8-i%8))/8)
}

// HighestBitIndex returns the index of the highest
// bit set from bitlist `b`.
func HighestBitIndex(b []byte) (int, error) {
	if b == nil || len(b) == 0 {
		return 0, errors.New("input list can't be empty or nil")
	}

	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == 0 {
			continue
		}
		return bits.Len8(b[i]) + (i * 8), nil
	}

	return 0, nil
}

// HighestBitIndexAt returns the index of the highest
// bit set from bitlist `b` that is at `index` (inclusive).
func HighestBitIndexAt(b []byte, index int) (int, error) {
	bLength := len(b)
	if b == nil || bLength == 0 {
		return 0, errors.New("input list can't be empty or nil")
	}

	start := index / 8
	if start >= bLength {
		start = bLength - 1
	}

	mask := byte(1<<(index%8) - 1)
	for i := start; i >= 0; i-- {
		if index/8 > i {
			mask = 0xff
		}
		masked := b[i] & mask
		minBitsMasked := bits.Len8(masked)
		if b[i] == 0 || (minBitsMasked == 0 && index/8 <= i) {
			continue
		}

		return minBitsMasked + (i * 8), nil
	}

	return 0, nil
}

// Uint64ToBytes little endian conversion.
func Uint64ToBytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}

// BytesToUint64 little endian conversion. Returns 0 if empty bytes or byte slice with length less
// than 8.
func BytesToUint64(b []byte) uint64 {
	if len(b) < 8 { // This will panic otherwise.
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}

// Uint64ToBytes big endian conversion.
func Uint64ToBytesBigEndian(i uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

// BytesToUint64 big endian conversion. Returns 0 if empty bytes or byte slice with length less
// than 8.
func BytesToUint64BigEndian(b []byte) uint64 {
	if len(b) < 8 { // This will panic otherwise.
		return 0
	}
	return binary.BigEndian.Uint64(b)
}
