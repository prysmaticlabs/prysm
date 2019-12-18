package stateutil_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
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

func BenchmarkHashTreeRootState_Custom_512(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 512)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := stateutil.HashTreeRootState(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Custom_16384(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 16384)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := stateutil.HashTreeRootState(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Custom_300000(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 300000)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := stateutil.HashTreeRootState(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Generic_512(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 512)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ssz.HashTreeRoot(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Generic_16384(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 16384)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ssz.HashTreeRoot(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_Generic_300000(b *testing.B) {
	b.StopTimer()
	genesisState := setupGenesisState(b, 300000)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ssz.HashTreeRoot(genesisState); err != nil {
			b.Fatal(err)
		}
	}
}

func setupGenesisState(tb testing.TB, count uint64) *pb.BeaconState {
	genesisState, _, err := interop.GenerateGenesisState(0, 1)
	if err != nil {
		tb.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	for i := uint64(1); i < count; i++ {
		someRoot := [32]byte{}
		someKey := [48]byte{}
		copy(someRoot[:], strconv.Itoa(int(i)))
		copy(someKey[:], strconv.Itoa(int(i)))
		genesisState.Validators = append(genesisState.Validators, &ethpb.Validator{
			PublicKey:                  someKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 1,
			ActivationEpoch:            1,
			ExitEpoch:                  1,
			WithdrawableEpoch:          1,
		})
		genesisState.Balances = append(genesisState.Balances, params.BeaconConfig().MaxEffectiveBalance)
	}
	return genesisState
}
