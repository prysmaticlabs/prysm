package hashutil

import (
	"golang.org/x/crypto/blake2b"
)

// Hash defines a function that returns the
// blake2b hash of the data passed in.
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#appendix
func Hash(data []byte) [32]byte {
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash
}
