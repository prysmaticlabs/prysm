package bitfield

import (
	"math/bits"
)

// Bitlist is a bitfield implementation backed by an array of bytes. The most
// significant bit in the array of bytes indicates the start position of the
// bitfield.
//
// Examples of the underlying byte array as bitlist:
//  byte{0b00001000} is a bitlist with 3 bits which are all zero. bits=[0,0,0]
//  byte{0b00011111} is a bitlist with 4 bits which are all one.  bits=[1,1,1,1]
//  byte{0b00011000, 0b00000001} is a bitlist with 8 bits.        bits=[0,0,0,1,1,0,0,0]
//  byte{0b00011000, 0b00000010} is a bitlist with 9 bits.        bits=[0,0,0,0,1,1,0,0,0]
type Bitlist []byte

func (b Bitlist) BitAt(idx uint64) bool {
	// Out of bounds, must be false.
	upperBounds := b.Len()
	if idx > upperBounds || upperBounds == 0 {
		return false
	}

	i := uint8(1 << (idx % 8))
	return b[idx/8]&i == i
}

func (b Bitlist) SetBitAt(idx uint64, val bool) {
	// Out of bounds, do nothing.
	upperBounds := b.Len()
	if idx > upperBounds || upperBounds == 0 {
		return
	}

	i := uint8(1 << (idx % 8))
	if val {
		b[idx/8] |= i
	} else {
		b[idx/8] &^= i
	}

}

func (b Bitlist) Len() uint64 {
	// The most significant bit is present in the last byte in the array.
	last := b[len(b)-1]

	// Determine the position of the most significant bit.
	msb := bits.Len8(last)

	// The absolute position of the most significant bit will be the number of
	// bits in the preceding bytes plus the position of the most significant
	// bit. Subtract this value by 1 to determine the length of the bitlist.
	return uint64(8*(len(b)-1) + msb - 1)
}

// Bytes returns the underlying byte array without the length bit.
func (b Bitlist) Bytes() []byte {
	// TODO
	return b
}
