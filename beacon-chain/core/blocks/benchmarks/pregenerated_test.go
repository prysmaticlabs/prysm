package blocks_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
)

func BenchmarkGenerateMarshalledFullStateAndBlock(b *testing.B) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	attsPerEpoch := uint64(benchmarkConfig().MaxAttestations)
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (attsPerEpoch / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = benchmarkConfig().MaxAttestations
	params.OverrideBeaconConfig(c)

	beaconState := genesisBeaconState(b)

	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		b.Fatal(err)
	}

	conf := &testutil.BlockGenConfig{
		MaxAttestations: 0,
		Signatures:      true,
	}

	// Process beacon state to mid-epoch to prevent epoch calculations from manipulating benchmarks.
	for i := uint64(0); i < 2; i++ {
		block := testutil.GenerateFullBlock(b, beaconState, privs, conf, i)
		beaconState, err = state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
		if err != nil {
			b.Error(err)
		}
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: 2,
		Signatures:      true,
	}
	atts := []*ethpb.Attestation{}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch-1; i++ {
		b.Logf("state at slot %d\n", beaconState.Slot)
		attsForSlot := testutil.GenerateAttestations(b, beaconState, privs, attConfig)
		atts = append(atts, attsForSlot...)
		block := testutil.GenerateFullBlock(b, beaconState, privs, conf, beaconState.Slot)
		beaconState, err = state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
		if err != nil {
			b.Error(err)
		}
	}

	block := testutil.GenerateFullBlock(b, beaconState, privs, attConfig, beaconState.Slot)
	block.Body.Attestations = append(atts, block.Body.Attestations[0])
	b.Logf(len(block.Body.Attestations))

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

	beaconBytes, err := proto.Marshal(beaconState)
	if err != nil {
		b.Fatal(err)
	}
	if err := ioutil.WriteFile("beaconState2Epochs.bytes", beaconBytes); err != nil {
		b.Fatal(err)
	}

	blockBytes, err := proto.Marshal(block)
	if err != nil {
		b.Fatal(err)
	}
	if err := ioutil.WriteFile("block127Atts.bytes", blockBytes); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkSaveGenesisBeaconState(b *testing.B) {
	testutil.SetupInitialDeposits(b, validatorCount)
	genesisState := state.GenesisBeaconState()
	beaconBytes, err := proto.Marshal(genesisState)
	if err != nil {
		b.Fatal(err)
	}
	if err := ioutil.WriteFile("genesisState.bytes", beaconBytes); err != nil {
		b.Fatal(err)
	}
}

func genesisBeaconState(b testing.TB) *pb.BeaconState {
	beaconBytes, err := ioutil.ReadFile("genesisState.bytes")
	if err != nil {
		b.Fatal(err)
	}
	genesisState := &pb.BeaconState{}
	if err := proto.Unmarshal(beaconBytes, genesisState); err != nil {
		b.Fatal(err)
	}
	return genesisState
}

func beaconState2Epochs(b testing.TB) *pb.BeaconState {
	beaconBytes, err := ioutil.ReadFile("beaconState2Epochs.bytes")
	if err != nil {
		b.Fatal(err)
	}
	beaconState := &pb.BeaconState{}
	if err := proto.Unmarshal(beaconBytes, beaconState); err != nil {
		b.Fatal(err)
	}
	return beaconState
}

func fullBlock(b testing.TB) *pb.BeaconState {
	beaconBytes, err := ioutil.ReadFile("block127Atts.bytes")
	if err != nil {
		b.Fatal(err)
	}
	beaconState := &pb.BeaconState{}
	if err := proto.Unmarshal(beaconBytes, beaconState); err != nil {
		b.Fatal(err)
	}
	return beaconState
}
