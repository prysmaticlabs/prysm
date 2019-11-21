package stateutil

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

func BenchmarkHashTreeRootState_Custom(b *testing.B) {
	b.StopTimer()
	count := 512
	genesisState, _, err := interop.GenerateGenesisState(0, uint64(count))
	if err != nil {
		b.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := HashTreeRootState(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Generic(b *testing.B) {
	b.StopTimer()
	count := 512
	genesisState, _, err := interop.GenerateGenesisState(0, uint64(count))
	if err != nil {
		b.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ssz.HashTreeRoot(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}
