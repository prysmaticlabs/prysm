package stateutil_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
)

func TestState_FieldCount(t *testing.T) {
	count := 20
	typ := reflect.TypeOf(pb.BeaconState{})
	numFields := 0
	for i := 0; i < typ.NumField(); i++ {
		if strings.HasPrefix(typ.Field(i).Name, "XXX_") {
			continue
		}
		numFields++
	}
	if numFields != count {
		t.Errorf("Expected state to have %d fields, received %d", count, numFields)
	}
}

func TestHashTreeRootEquality(t *testing.T) {
	genesisState := setupGenesisState(t, 512)
	r1, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := stateutil.HashTreeRootState(genesisState)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("Wanted %#x, got %#x", r1, r2)
	}
}

func TestHashTreeRootState_ElementChanged(t *testing.T) {
	roots := make([][]byte, 4)
	for i := 0; i < len(roots); i++ {
		rt := [32]byte{1, 2, 3}
		roots[i] = rt[:]
	}

	if _, err := stateutil.ArraysRoot(roots, "BlockRoots"); err != nil {
		t.Fatal(err)
	}

	newRt := [32]byte{4, 5, 6}
	roots[0] = newRt[:]

	if _, err := stateutil.ArraysRoot(roots, "BlockRoots"); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkHashTreeRootState_Custom(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 512)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := stateutil.HashTreeRootState(genesisState); err != nil {
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
