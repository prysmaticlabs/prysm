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
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

var validatorCount = 65536
var runAmount = 15

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
		EnableCommitteeCache: true,
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
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	attsPerEpoch := uint64(1024)
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (attsPerEpoch / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.MainnetConfig())

	beaconState := genesisBeaconState(b)

	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		b.Fatal(err)
	}

	conf := &testutil.BlockGenConfig{
		MaxAttestations: 16,
		Signatures:      false,
	}

	for i := uint64(0); i < (slotsPerEpoch*2)-1; i++ {
		block := testutil.GenerateFullBlock(b, beaconState, privs, conf, beaconState.Slot)
		beaconState, err = state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
		if err != nil {
			b.Error(err)
		}
	}

	cleanStates := clonedStates(beaconState)

	fmt.Printf("Atts in current epoch %d\n", len(beaconState.CurrentEpochAttestations))
	fmt.Printf("Atts in prev epoch %d\n", len(beaconState.PreviousEpochAttestations))

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Printf("%d ", i)
		if _, err := state.ProcessEpoch(context.Background(), cleanStates[i]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashTreeRoot_65536Validators(b *testing.B) {
	maxAtts := benchmarkConfig(b).MaxAttestations
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.MainnetConfig())

	beaconState := beaconState1Epoch(b)

	b.N = 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Println(i)
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
