package hashutil

import (
	"golang.org/x/crypto/sha3"
)

// Hash defines a function that returns the
// Keccak-256/SHA3 hash of the data passed in.
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#appendix
func Hash(data []byte) [32]byte {
	var hash [32]byte

	h := sha3.NewLegacyKeccak256()

	// #nosec G104
	h.Write(data)
	h.Sum(hash[:0])

	return hash
}
