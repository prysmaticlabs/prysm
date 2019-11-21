package stateutil

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

func TestHashTreeRootEquality(t *testing.T) {
	count := 512
	genesisState, _, err := interop.GenerateGenesisState(0, uint64(count))
	if err != nil {
		t.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	r1, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := HashTreeRootState(genesisState)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("Wanted %#x, got %#x", r1, r2)
	}
}

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
