package benchmarks_test

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

var validatorCount = 65536
var runAmount = 25

func benchmarkConfig(t testing.TB) *testutil.BlockGenConfig {
	t.Logf("Running block benchmarks for %d validators", validatorCount)

	return &testutil.BlockGenConfig{
		MaxProposerSlashings: 0,
		MaxAttesterSlashings: 0,
		MaxAttestations:      128,
		MaxDeposits:          0,
		MaxVoluntaryExits:    0,
	}
}

func setConfig(t testing.TB) {
	maxAtts := benchmarkConfig(t).MaxAttestations
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)
}

func TestBenchmarkExecuteStateTransition(t *testing.T) {
	setConfig(t)

	beaconState := beaconState1Epoch(t)
	block := fullBlock(t)

	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func TestBenchmarkProcessEpoch(t *testing.T) {
	setConfig(t)

	beaconState := beaconState2FullEpochs(t)

	if _, err := state.ProcessEpoch(context.Background(), beaconState); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func BenchmarkExecuteStateTransition(b *testing.B) {
	setConfig(b)

	beaconState := beaconState1Epoch(b)
	cleanStates := clonedStates(beaconState)
	block := fullBlock(b)

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
		EnableNewCache: true,
	}
	featureconfig.Init(config)
	setConfig(b)

	beaconState := beaconState1Epoch(b)
	cleanStates := clonedStates(beaconState)
	block := fullBlock(b)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessEpoch_2FullEpochs(b *testing.B) {
	config := &featureconfig.Flags{
		EnableActiveIndicesCache: true,
		EnableActiveCountCache:   true,
		EnableNewCache:           true,
	}
	featureconfig.Init(config)
	setConfig(b)

	beaconState := beaconState2FullEpochs(b)
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
	beaconState := beaconState2FullEpochs(b)

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
