package stateutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/crypto/hash"
	htrutils2 "github.com/prysmaticlabs/prysm/encoding/htrutils"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func BenchmarkMerkleize_Buffered(b *testing.B) {
	roots := make([][32]byte, 8192)
	for i := 0; i < 8192; i++ {
		roots[0] = [32]byte{byte(i)}
	}

	newMerkleize := func(chunks [][32]byte, count uint64, limit uint64) ([32]byte, error) {
		leafIndexer := func(i uint64) []byte {
			return chunks[i][:]
		}
		return htrutils2.Merkleize(htrutils2.NewHasherFunc(hash.CustomSHA256Hasher()), count, limit, leafIndexer), nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := newMerkleize(roots, 8192, 8192)
		require.NoError(b, err)
	}
}
