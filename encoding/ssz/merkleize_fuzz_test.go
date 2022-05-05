package ssz_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/ssz"
)

func convertRawChunks(raw []byte) [][32]byte {
	var chunks [][32]byte
	for i := 0; i < len(raw) && len(raw) > 0; i += 32 {
		var c [32]byte
		end := i + 32
		if end >= len(raw) {
			end = len(raw) - 1
		}
		copy(c[:], raw[i:end])
		chunks = append(chunks, c)
	}
	return chunks
}

func FuzzMerkleizeVector(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte, length uint64) {
		ssz.MerkleizeVector(convertRawChunks(b), length)
	})
}
