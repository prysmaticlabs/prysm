package stateutil

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

func TestHashTreeRootEquality(t *testing.T) {
	genesisState := setupGenesisState(b, 512)
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
	genesisState := setupGenesisState(b, 512)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := HashTreeRootState(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Generic(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 512)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ssz.HashTreeRoot(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func setupGenesisState(tb testing.TB, count uint64) *pb.BeaconState {
	genesisState, _, err := interop.GenerateGenesisState(0, count)
	if err != nil {
		tb.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	return genesisState
}
