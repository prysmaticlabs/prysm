package stateutil_benchmark

import (
	"testing"

	"github.com/protolambda/zssz/merkle"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func BenchmarkBlockHTR(b *testing.B) {
	genState, keys := testutil.DeterministicGenesisState(b, 200)
	conf := testutil.DefaultBlockGenConfig()
	blk, err := testutil.GenerateFullBlock(genState, keys, conf, 10)
	if err != nil {
		b.Fatal(err)
	}
	atts := make([]*ethpb.Attestation, 0, 128)
	for i := 0; i < 128; i++ {
		atts = append(atts, blk.Block.Body.Attestations[0])
	}
	deposits, _, err := testutil.DeterministicDepositsAndKeys(16)
	if err != nil {
		b.Fatal(err)
	}
	blk.Block.Body.Attestations = atts
	blk.Block.Body.Deposits = deposits

	b.Run("SSZ_HTR", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := stateutil.BlockRoot(blk.Block); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Custom_SSZ_HTR", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := stateutil.BlockRoot(blk.Block); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkMerkleize(b *testing.B) {
	roots := make([][32]byte, 8192)
	for i := 0; i < 8192; i++ {
		roots[0] = [32]byte{byte(i)}
	}
	oldMerkleize := func(chunks [][32]byte, count uint64, limit uint64) ([32]byte, error) {
		leafIndexer := func(i uint64) []byte {
			return chunks[i][:]
		}
		return merkle.Merkleize(hashutil.CustomSHA256Hasher(), count, limit, leafIndexer), nil
	}

	newMerkleize := func(chunks [][32]byte, count uint64, limit uint64) ([32]byte, error) {
		leafIndexer := func(i uint64) []byte {
			return chunks[i][:]
		}
		return stateutil.Merkleize(stateutil.NewHasherFunc(hashutil.CustomSHA256Hasher()), count, limit, leafIndexer), nil
	}

	b.Run("Non Buffered Merkleizer", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			_, _ = oldMerkleize(roots, 8192, 8192)
		}
	})

	b.Run("Buffered Merkleizer", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			_, _ = newMerkleize(roots, 8192, 8192)
		}
	})

}
