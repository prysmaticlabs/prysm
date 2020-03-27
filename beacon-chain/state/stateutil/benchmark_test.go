package stateutil_benchmark

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
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
			if _, err := ssz.HashTreeRoot(blk.Block); err != nil {
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
