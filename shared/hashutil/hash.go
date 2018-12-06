package hashutil

import (
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

// Hash defines a function that returns the
// Keccak-256/SHA3 hash of the data passed in.
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#appendix
func Hash(data []byte) [32]byte {
	var hash [32]byte

	h := sha3.Sum256(data)
	copy(hash[:], h[:32])
	return hash
}
