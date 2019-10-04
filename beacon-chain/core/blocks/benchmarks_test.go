package blocks_test

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
			MaxProposerSlashings: 0,
			MaxAttesterSlashings: 0,
			MaxAttestations:      256,
			MaxDeposits:          0,
			MaxVoluntaryExits:    0,
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

	beaconState := createBeaconState(t)

	conf := benchmarkConfig()
	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		t.Fatal(err)
	}
	block := testutil.GenerateFullBlock(t, beaconState, privs, conf)

	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block); err != nil {
		t.Fatalf("failed to process block, benchmarks will fail: %v", err)
	}
}

func BenchmarkExecuteStateTransition(b *testing.B) {
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = 256
	c.MaxAttestations = benchmarkConfig().MaxAttestations
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.MainnetConfig())

	beaconState := createBeaconState(b)

	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		b.Fatal(err)
	}
	atts := []*ethpb.Attestation{}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		attsForSlot := testutil.GenerateAttestations(b, beaconState, privs, 2)
		atts = append(atts, attsForSlot...)
		conf := &testutil.BlockGenConfig{
			MaxAttestations: 0,
		}
		block := testutil.GenerateFullBlock(b, beaconState, privs, conf)
		beaconState, err = state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
		if err != nil {
			b.Error(err)
		}
	}

	conf := &testutil.BlockGenConfig{
		MaxAttestations: 2,
	}
	block := testutil.GenerateFullBlock(b, beaconState, privs, conf)
	block.Body.Attestations = append(atts, block.Body.Attestations...)
	b.Log(len(atts))

	s, err := state.CalculateStateRoot(context.Background(), beaconState, block)
	if err != nil {
		b.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(s)
	if err != nil {
		b.Fatal(err)
	}
	block.StateRoot = root[:]
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		b.Fatal(err)
	}
	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	beaconState.Slot++
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		b.Fatal(err)
	}
	beaconState.Slot--
	domain := helpers.Domain(beaconState.Fork, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainBeaconProposer)
	block.Signature = privs[proposerIdx].Sign(blockRoot[:], domain).Marshal()

	cleanStates := createCleanStates(beaconState)

	b.N = runAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := state.ExecuteStateTransition(context.Background(), cleanStates[i], block); err != nil {
			b.Fatal(err)
		}
	}
}

func createBeaconState(b testing.TB) *pb.BeaconState {
	beaconSSZ, err := ioutil.ReadFile("genesisState.ssz")
	if err != nil {
		b.Fatal(err)
	}
	genesisState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconSSZ, genesisState); err != nil {
		b.Fatal(err)
	}

	return genesisState
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
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}

	stateSSZ, err := ssz.Marshal(beaconState)
	if err != nil {
		b.Fatal(err)
	}
	if err = ioutil.WriteFile("genesisState.ssz", stateSSZ, 0644); err != nil {
		b.Fatal(err)
	}
}
