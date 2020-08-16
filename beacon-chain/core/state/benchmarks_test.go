package state_test

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/benchutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var runAmount = 25

func TestExecuteStateTransition_FullBlock(t *testing.T) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState1Epoch()
	require.NoError(t, err)
	block, err := benchutil.PreGenFullBlock()
	require.NoError(t, err)

	oldSlot := beaconState.Slot()
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err, "Failed to process block, benchmarks will fail")
	require.NotEqual(t, oldSlot, beaconState.Slot(), "Expected slots to be different")
}

func BenchmarkExecuteStateTransition_FullBlock(b *testing.B) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState1Epoch()
	require.NoError(b, err)
	cleanStates := clonedStates(beaconState)
	block, err := benchutil.PreGenFullBlock()
	require.NoError(b, err)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block)
		require.NoError(b, err)
	}
}

func BenchmarkExecuteStateTransition_WithCache(b *testing.B) {
	benchutil.SetBenchmarkConfig()

	beaconState, err := benchutil.PreGenState1Epoch()
	require.NoError(b, err)
	cleanStates := clonedStates(beaconState)
	block, err := benchutil.PreGenFullBlock()
	require.NoError(b, err)

	// We have to reset slot back to last epoch to hydrate cache. Since
	// some attestations in block are from previous epoch
	currentSlot := beaconState.Slot()
	require.NoError(b, beaconState.SetSlot(beaconState.Slot()-params.BeaconConfig().SlotsPerEpoch))
	require.NoError(b, helpers.UpdateCommitteeCache(beaconState, helpers.CurrentEpoch(beaconState)))
	require.NoError(b, beaconState.SetSlot(currentSlot))
	// Run the state transition once to populate the cache.
	_, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(b, err, "Failed to process block, benchmarks will fail")

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block)
		require.NoError(b, err, "Failed to process block, benchmarks will fail")
	}
}

func BenchmarkProcessEpoch_2FullEpochs(b *testing.B) {
	benchutil.SetBenchmarkConfig()
	beaconState, err := benchutil.PreGenState2FullEpochs()
	require.NoError(b, err)

	// We have to reset slot back to last epoch to hydrate cache. Since
	// some attestations in block are from previous epoch
	currentSlot := beaconState.Slot()
	require.NoError(b, beaconState.SetSlot(beaconState.Slot()-params.BeaconConfig().SlotsPerEpoch))
	require.NoError(b, helpers.UpdateCommitteeCache(beaconState, helpers.CurrentEpoch(beaconState)))
	require.NoError(b, beaconState.SetSlot(currentSlot))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ProcessEpochPrecompute is the optimized version of process epoch. It's enabled by default
		// at run time.
		_, err := state.ProcessEpochPrecompute(context.Background(), beaconState.Copy())
		require.NoError(b, err)
	}
}

func BenchmarkHashTreeRoot_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	require.NoError(b, err)

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
	require.NoError(b, err)

	ctx := context.Background()

	// Hydrate the HashTreeRootState cache.
	_, err = beaconState.HashTreeRoot(ctx)
	require.NoError(b, err)

	b.N = 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := beaconState.HashTreeRoot(ctx)
		require.NoError(b, err)
	}
}

func BenchmarkMarshalState_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	require.NoError(b, err)
	natState := beaconState.InnerStateUnsafe()

	b.Run("Proto_Marshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			_, err := proto.Marshal(natState)
			require.NoError(b, err)
		}
	})

	b.Run("Fast_SSZ_Marshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			_, err := natState.MarshalSSZ()
			require.NoError(b, err)
		}
	})
}

func BenchmarkUnmarshalState_FullState(b *testing.B) {
	beaconState, err := benchutil.PreGenState2FullEpochs()
	require.NoError(b, err)
	natState := beaconState.InnerStateUnsafe()
	protoObject, err := proto.Marshal(natState)
	require.NoError(b, err)
	sszObject, err := natState.MarshalSSZ()
	require.NoError(b, err)

	b.Run("Proto_Unmarshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			require.NoError(b, proto.Unmarshal(protoObject, &pb.BeaconState{}))
		}
	})

	b.Run("Fast_SSZ_Unmarshal", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 1000
		for i := 0; i < b.N; i++ {
			sszState := &pb.BeaconState{}
			require.NoError(b, sszState.UnmarshalSSZ(sszObject))
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
