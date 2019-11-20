package benchmarks_test

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

func TestBenchmarkExecuteStateTransition(t *testing.T) {
	maxAtts := benchmarkConfig(t).MaxAttestations
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)

	beaconState := beaconState1Epoch(t)
	block := fullBlock(t)

	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func BenchmarkExecuteStateTransition(b *testing.B) {
	maxAtts := benchmarkConfig(b).MaxAttestations
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)

	beaconState := beaconState1Epoch(b)
	cleanStates := clonedStates(beaconState)
	block := fullBlock(b)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Println(i)
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteStateTransition_WithCache(b *testing.B) {
	config := &featureconfig.Flag{
		EnableActiveIndicesCache: true,
		EnableActiveCountCache:   true,
		EnableNewCache:           true,
	}
	featureconfig.Init(config)
	maxAtts := benchmarkConfig(b).MaxAttestations
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)

	beaconState := beaconState1Epoch(b)
	cleanStates := clonedStates(beaconState)
	block := fullBlock(b)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Println(i)
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteStateTransition_ProcessEpoch(b *testing.B) {
	config := &featureconfig.Flag{
		EnableActiveIndicesCache: true,
		EnableActiveCountCache:   true,
		EnableNewCache:           true,
	}
	featureconfig.Init(config)
	maxAtts := benchmarkConfig(b).MaxAttestations
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)

	beaconState := beaconState2FullEpochs(b)
	cleanStates := clonedStates(beaconState)

	fmt.Println(beaconState.Slot)

	b.N = 5
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Println(i)
		if _, err := state.ProcessEpoch(context.Background(), cleanStates[i]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRoot_FullState65536Validators(b *testing.B) {
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
