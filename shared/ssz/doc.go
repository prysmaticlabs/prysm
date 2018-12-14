/*
Package ssz implements the Simple Serialize algorithm specified at
https://github.com/ethereum/eth2.0-specs/blob/master/specs/simple-serialize.md

Currently directly supported types:

bool
uint8
uint16
uint32
uint64
bytes
slice
struct

Types that can be implicitly supported:

address:
	use byte slice of length 20 instead
hash:
	use byte slice of length 32 instead if the hash is 32 bytes long, for example
*/
package ssz
