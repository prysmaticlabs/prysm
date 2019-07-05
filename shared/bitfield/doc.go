// Package bitfield exposes helpers for bitfield operations.
//
// A bitfield is also known as a Bitlist or Bitvector in Ethereum 2.0 spec.
// Both variants are static arrays in that they cannot dynamically change in
// size after being constructed. These data types represent a list of bits whose
// value is treated akin to a boolean. The bits are in little endian order.
//
// 	Bitvector - A list of bits that is fixed in size.
// 	Bitlist - A list of bits that is determined at runtime.
package bitfield
