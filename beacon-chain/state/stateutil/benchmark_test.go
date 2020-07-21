package stateutil_test

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func BenchmarkBlockHTR(b *testing.B) {
	genState, keys := testutil.DeterministicGenesisState(b, 200)
	conf := testutil.DefaultBlockGenConfig()
	blk, err := testutil.GenerateFullBlock(genState, keys, conf, 10)
	require.NoError(b, err)
	atts := make([]*ethpb.Attestation, 0, 128)
	for i := 0; i < 128; i++ {
		atts = append(atts, blk.Block.Body.Attestations[0])
	}
	deposits, _, err := testutil.DeterministicDepositsAndKeys(16)
	require.NoError(b, err)
	blk.Block.Body.Attestations = atts
	blk.Block.Body.Deposits = deposits

	b.Run("SSZ_HTR", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := stateutil.BlockRoot(blk.Block)
			require.NoError(b, err)
		}
	})

	b.Run("Custom_SSZ_HTR", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := stateutil.BlockRoot(blk.Block)
			require.NoError(b, err)
		}
	})
}

func BenchmarkMerkleize_Buffered(b *testing.B) {
	roots := make([][32]byte, 8192)
	for i := 0; i < 8192; i++ {
		roots[0] = [32]byte{byte(i)}
	}

	newMerkleize := func(chunks [][32]byte, count uint64, limit uint64) ([32]byte, error) {
		leafIndexer := func(i uint64) []byte {
			return chunks[i][:]
		}
		return htrutils.Merkleize(htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher()), count, limit, leafIndexer), nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.N = 1000
	for i := 0; i < b.N; i++ {
		_, err := newMerkleize(roots, 8192, 8192)
		require.NoError(b, err)
	}
}
