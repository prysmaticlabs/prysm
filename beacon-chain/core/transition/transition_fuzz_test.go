package transition

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestFuzzExecuteStateTransition_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	sb := &ethpb.SignedBeaconBlock{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(sb)
		wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
		require.NoError(t, err)
		s, err := ExecuteStateTransition(ctx, state, wsb)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v and signed block: %v", s, err, state, sb)
		}
	}
}

func TestFuzzCalculateStateRoot_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	sb := &ethpb.SignedBeaconBlock{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(sb)
		wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
		require.NoError(t, err)
		stateRoot, err := CalculateStateRoot(ctx, state, wsb)
		if err != nil && stateRoot != [32]byte{} {
			t.Fatalf("state root should be empty on err. found: %v on error: %v for signed block: %v", stateRoot, err, sb)
		}
	}
}

func TestFuzzProcessSlot_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		s, err := ProcessSlot(ctx, state)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestFuzzProcessSlots_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	slot := types.Slot(0)
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&slot)
		s, err := ProcessSlots(ctx, state, slot)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestFuzzprocessOperationsNoVerify_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	bb := &ethpb.SignedBeaconBlock{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(bb)
		wsb, err := wrapper.WrappedSignedBeaconBlock(bb)
		require.NoError(t, err)
		s, err := ProcessOperationsNoVerifyAttsSigs(ctx, state, wsb)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for block body: %v", s, err, bb)
		}
	}
}

func TestFuzzverifyOperationLengths_10000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	bb := &ethpb.SignedBeaconBlock{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(bb)
		wsb, err := wrapper.WrappedSignedBeaconBlock(bb)
		require.NoError(t, err)
		_, err = VerifyOperationLengths(context.Background(), state, wsb)
		_ = err
	}
}

func TestFuzzCanProcessEpoch_10000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		time.CanProcessEpoch(state)
	}
}

func TestFuzzProcessEpochPrecompute_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		s, err := ProcessEpochPrecompute(ctx, state)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestFuzzProcessBlockForStateRoot_1000(t *testing.T) {
	SkipSlotCache.Disable()
	defer SkipSlotCache.Enable()
	ctx := context.Background()
	state, err := v1.InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	sb := &ethpb.SignedBeaconBlock{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(sb)
		wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
		require.NoError(t, err)
		s, err := ProcessBlockForStateRoot(ctx, state, wsb)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for signed block: %v", s, err, sb)
		}
	}
}
