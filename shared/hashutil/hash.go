package hashutil

import (
	"golang.org/x/crypto/blake2b"
)

func Hash(data []byte) [32]byte {
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash
}
