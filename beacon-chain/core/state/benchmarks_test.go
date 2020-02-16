package state

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/benchutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var runAmount = 25

func TestBenchmarkExecuteStateTransition(t *testing.T) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState1Epoch()
	if err != nil {
		t.Fatal(err)
	}
	block, err := benchutil.PreGenFullBlock()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
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
		if _, err := ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
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
	beaconState.SetSlot(beaconState.Slot() - params.BeaconConfig().SlotsPerEpoch)
	if err := helpers.UpdateCommitteeCache(beaconState, helpers.CurrentEpoch(beaconState)); err != nil {
		b.Fatal(err)
	}
	beaconState.SetSlot(currentSlot)
	// Run the state transition once to populate the cache.
	if _, err := ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		b.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
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
	beaconState.SetSlot(beaconState.Slot() - params.BeaconConfig().SlotsPerEpoch)
	if err := helpers.UpdateCommitteeCache(beaconState, helpers.CurrentEpoch(beaconState)); err != nil {
		b.Fatal(err)
	}
	beaconState.SetSlot(currentSlot)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ProcessEpochPrecompute is the optimized version of process epoch. It's enabled by default
		// at run time.
		if _, err := ProcessEpochPrecompute(context.Background(), beaconState.Copy()); err != nil {
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
		if _, err := ssz.HashTreeRoot(beaconState); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRootState_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}

	// Hydrate the HashTreeRootState cache.
	if _, err := beaconState.HashTreeRoot(); err != nil {
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

func clonedStates(beaconState *beaconstate.BeaconState) []*beaconstate.BeaconState {
	clonedStates := make([]*beaconstate.BeaconState, runAmount)
	for i := 0; i < runAmount; i++ {
		clonedStates[i] = beaconState.Copy()
	}
	return clonedStates
}
