package benchmarks

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var runAmount = 25

func TestBenchmarkExecuteStateTransition(t *testing.T) {
	SetConfig()
	beaconState, err := beaconState1Epoch()
	if err != nil {
		t.Fatal(err)
	}
	block, err := fullBlock()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func TestBenchmarkProcessEpoch(t *testing.T) {
	SetConfig()
	beaconState, err := beaconState2FullEpochs()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := state.ProcessEpoch(context.Background(), beaconState); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func BenchmarkExecuteStateTransition(b *testing.B) {
	SetConfig()
	beaconState, err := beaconState1Epoch()
	if err != nil {
		b.Fatal(err)
	}
	cleanStates := clonedStates(beaconState)
	block, err := fullBlock()
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
	config := &featureconfig.Flags{
		EnableNewCache:           true,
		EnableShuffledIndexCache: true,
		EnableBLSPubkeyCache:     true,
	}
	featureconfig.Init(config)
	SetConfig()

	beaconState, err := beaconState1Epoch()
	if err != nil {
		b.Fatal(err)
	}
	cleanStates := clonedStates(beaconState)
	block, err := fullBlock()
	if err != nil {
		b.Fatal(err)
	}

	// We have to reset slot back to last epoch to hydrate cache. Since
	// some attestations in block are from previous epoch
	currentSlot := beaconState.Slot
	beaconState.Slot -= params.BeaconConfig().SlotsPerEpoch
	if err := helpers.UpdateCommitteeCache(beaconState); err != nil {
		b.Fatal(err)
	}

	beaconState.Slot = currentSlot
	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatalf("failed to process block, benchmarks will fail: %v", err)
		}
	}
}

func BenchmarkProcessEpoch_2FullEpochs(b *testing.B) {
	SetConfig()
	beaconState, err := beaconState2FullEpochs()
	if err != nil {
		b.Fatal(err)
	}
	cleanStates := clonedStates(beaconState)

	b.N = 5
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ProcessEpoch(context.Background(), cleanStates[i]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRoot_FullState(b *testing.B) {
	beaconState, err := beaconState2FullEpochs()
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

func clonedStates(beaconState *pb.BeaconState) []*pb.BeaconState {
	clonedStates := make([]*pb.BeaconState, runAmount)
	for i := 0; i < runAmount; i++ {
		clonedStates[i] = proto.Clone(beaconState).(*pb.BeaconState)
	}
	return clonedStates
}

func beaconState1Epoch() (*pb.BeaconState, error) {
	path, err := bazel.Runfile(BState1EpochFileName)
	if err != nil {
		return nil, err
	}
	beaconBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	beaconState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconBytes, beaconState); err != nil {
		return nil, err
	}
	return beaconState, nil
}

func beaconState2FullEpochs() (*pb.BeaconState, error) {
	path, err := bazel.Runfile(BState2EpochFileName)
	if err != nil {
		return nil, err
	}
	beaconBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	beaconState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconBytes, beaconState); err != nil {
		return nil, err
	}
	return beaconState, nil
}

func fullBlock() (*ethpb.BeaconBlock, error) {
	path, err := bazel.Runfile(FullBlockFileName)
	if err != nil {
		return nil, err
	}
	blockBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	beaconBlock := &ethpb.BeaconBlock{}
	if err := ssz.Unmarshal(blockBytes, beaconBlock); err != nil {
		return nil, err
	}
	return beaconBlock, nil
}
