package altair_test

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
)

func TestFuzzProcessEpoch_1000(t *testing.T) {
	ctx := context.Background()
	state := &stateAltair.BeaconState{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		s, err := altair.ProcessEpoch(ctx, state)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestFuzzProcessSlots_1000(t *testing.T) {
	altair.SkipSlotCache.Disable()
	defer altair.SkipSlotCache.Enable()
	ctx := context.Background()
	state := &stateAltair.BeaconState{}
	slot := types.Slot(0)
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&slot)
		s, err := altair.ProcessSlots(ctx, state, slot)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestFuzzCalculateStateRoot_1000(t *testing.T) {
	ctx := context.Background()
	state := &stateAltair.BeaconState{}
	sb := &ethpb.SignedBeaconBlockAltair{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(sb)
		stateRoot, err := altair.CalculateStateRoot(ctx, state, sb)
		if err != nil && stateRoot != [32]byte{} {
			t.Fatalf("state root should be empty on err. found: %v on error: %v for signed block: %v", stateRoot, err, sb)
		}
	}
}

func TestFuzzprocessOperationsNoVerify_1000(t *testing.T) {
	ctx := context.Background()
	state := &stateAltair.BeaconState{}
	bb := &ethpb.SignedBeaconBlockAltair{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(bb)
		s, err := altair.ProcessOperationsNoVerifyAttsSigs(ctx, state, bb)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for block body: %v", s, err, bb)
		}
	}
}
