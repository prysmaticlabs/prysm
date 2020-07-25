package state_test

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/benchutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var runAmount = 25

func TestExecuteStateTransition_FullBlock(t *testing.T) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState1Epoch()
	if err != nil {
		t.Fatal(err)
	}
	block, err := benchutil.PreGenFullBlock()
	if err != nil {
		t.Fatal(err)
	}

	oldSlot := beaconState.Slot()
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
	if oldSlot == beaconState.Slot() {
		t.Fatal("Expected slots to be different")
	}
}

func BenchmarkExecuteStateTransition_FullBlock(b *testing.B) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState1Epoch()
	if err != nil {
		b.Fatal(err)
	}
	cleanStates := clonedStates(beaconState)
	block, err := benchutil.PreGenFullBlock()
	if err != nil {
		b.Fatal(err)
	}

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteStateTransition_WithCache(b *testing.B) {
	benchutil.SetBenchmarkConfig()

	beaconState, err := benchutil.PreGenState1Epoch()
	if err != nil {
		b.Fatal(err)
	}
	cleanStates := clonedStates(beaconState)
	block, err := benchutil.PreGenFullBlock()
	if err != nil {
		b.Fatal(err)
	}

	// We have to reset slot back to last epoch to hydrate cache. Since
	// some attestations in block are from previous epoch
	currentSlot := beaconState.Slot()
	if err := beaconState.SetSlot(beaconState.Slot() - params.BeaconConfig().SlotsPerEpoch); err != nil {
		b.Fatal(err)
	}
	if err := helpers.UpdateCommitteeCache(beaconState, helpers.CurrentEpoch(beaconState)); err != nil {
		b.Fatal(err)
	}
	if err := beaconState.SetSlot(currentSlot); err != nil {
		b.Fatal(err)
	}
	// Run the state transition once to populate the cache.
	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		b.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatalf("failed to process block, benchmarks will fail: %v", err)
		}
	}
}

func BenchmarkProcessEpoch_2FullEpochs(b *testing.B) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}

	// We have to reset slot back to last epoch to hydrate cache. Since
	// some attestations in block are from previous epoch
	currentSlot := beaconState.Slot()
	if err := beaconState.SetSlot(beaconState.Slot() - params.BeaconConfig().SlotsPerEpoch); err != nil {
		b.Fatal(err)
	}
	if err := helpers.UpdateCommitteeCache(beaconState, helpers.CurrentEpoch(beaconState)); err != nil {
		b.Fatal(err)
	}
	if err := beaconState.SetSlot(currentSlot); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ProcessEpochPrecompute is the optimized version of process epoch. It's enabled by default
		// at run time.
		if _, err := state.ProcessEpochPrecompute(context.Background(), beaconState.Copy()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRoot_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}

	b.N = 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := beaconState.HashTreeRoot(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	// Hydrate the HashTreeRootState cache.
	if _, err := beaconState.HashTreeRoot(ctx); err != nil {
		b.Fatal(err)
	}

	b.N = 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := beaconState.HashTreeRoot(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalState_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}
	natState := beaconState.InnerStateUnsafe()

	b.Run("Proto_Marshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			if _, err := proto.Marshal(natState); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Fast_SSZ_Marshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			if _, err := natState.MarshalSSZ(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUnmarshalState_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}
	natState := beaconState.InnerStateUnsafe()
	protoObject, err := proto.Marshal(natState)
	if err != nil {
		b.Fatal(err)
	}
	sszObject, err := natState.MarshalSSZ()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Proto_Unmarshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			if err := proto.Unmarshal(protoObject, &pb.BeaconState{}); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Fast_SSZ_Unmarshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			sszState := &pb.BeaconState{}
			if err := sszState.UnmarshalSSZ(sszObject); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func clonedStates(beaconState *beaconstate.BeaconState) []*beaconstate.BeaconState {
	clonedStates := make([]*beaconstate.BeaconState, runAmount)
	for i := 0; i < runAmount; i++ {
		clonedStates[i] = beaconState.Copy()
	}
	return clonedStates
}
