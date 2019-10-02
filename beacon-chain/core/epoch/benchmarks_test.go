package epoch

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
)

var validatorCount = 32768
var runAmount = 40
var conditions = "SML"

func benchmarkConfig() *testutil.BlockGenConfig {
	logrus.Printf("Running block benchmarks for %d validators", validatorCount)
	logrus.SetLevel(logrus.PanicLevel)
	logrus.SetOutput(ioutil.Discard)
	if conditions == "BIG" {
		return &testutil.BlockGenConfig{
			MaxProposerSlashings: 16,
			MaxAttesterSlashings: 1,
			MaxAttestations:      128,
			MaxDeposits:          16,
			MaxVoluntaryExits:    16,
		}
	} else if conditions == "SML" {
		return &testutil.BlockGenConfig{
			MaxProposerSlashings: 0,
			MaxAttesterSlashings: 0,
			MaxAttestations:      128,
			MaxDeposits:          0,
			MaxVoluntaryExits:    0,
		}
	}
	return nil
}

func TestBenchmarkExecuteStateTransition_PerformsSuccessfully(t *testing.T) {
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.MainnetConfig())

	beaconState, block := createBeaconStateAndBlock(t)
	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func BenchmarkExecuteStateTransition(b *testing.B) {
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.MainnetConfig())

	beaconState, block := createBeaconStateAndBlock(b)
	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
}

func createBeaconStateAndBlock(b testing.TB) (*pb.BeaconState, *ethpb.BeaconBlock) {
	beaconSSZ, err := ioutil.ReadFile("genesisState.ssz")
	if err != nil {
		b.Fatal(err)
	}
	genesisState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconSSZ, genesisState); err != nil {
		b.Fatal(err)
	}

	conf := benchmarkConfig()
	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		b.Fatal(err)
	}
	fullBlock := testutil.GenerateFullBlock(b, genesisState, privs, conf)

	return genesisState, fullBlock
}

func createCleanStates(beaconState *pb.BeaconState) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, runAmount)
	for i := 0; i < runAmount; i++ {
		cleanStates[i] = proto.Clone(beaconState).(*pb.BeaconState)
	}
	return cleanStates
}

// Using benchmark here so I can run the function from VS code and not have it timeout
// like a normal test.
func BenchmarkSaveStateToDisk(b *testing.B) {
	deposits, _, _ := testutil.SetupInitialDeposits(b, uint64(validatorCount))
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}

	stateSSZ, err := ssz.Marshal(genesisState)
	if err != nil {
		b.Fatal(err)
	}
	if err = ioutil.WriteFile("genesisState.ssz", stateSSZ, 0644); err != nil {
		b.Fatal(err)
	}
}
