package benchmarks_test

import (
	"context"
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
)

func BenchmarkGenerateMarshalledFullStateAndBlock(b *testing.B) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	attsPerEpoch := benchmarkConfig(b).MaxAttestations
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (attsPerEpoch / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = attsPerEpoch
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

	block := testutil.GenerateFullBlock(b, beaconState, privs, conf, params.BeaconConfig().SlotsPerEpoch-1)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		b.Error(err)
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: 4,
		Signatures:      true,
	}
	atts := []*ethpb.Attestation{}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch-1; i++ {
		attsForSlot := testutil.GenerateAttestations(b, beaconState, privs, attConfig, i)
		atts = append(atts, attsForSlot...)
	}

	block = testutil.GenerateFullBlock(b, beaconState, privs, attConfig, beaconState.Slot)
	block.Body.Attestations = append(atts, block.Body.Attestations...)
	b.Logf("%d\n", len(block.Body.Attestations))

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
	if err := ioutil.WriteFile("beaconState1Epoch.bytes", beaconBytes, 0644); err != nil {
		b.Fatal(err)
	}

	_, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		b.Fatal(err)
	}

	blockBytes, err := proto.Marshal(block)
	if err != nil {
		b.Fatal(err)
	}
	if err := ioutil.WriteFile("block128Atts.bytes", blockBytes, 0644); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkGenerate2FullEpochState(b *testing.B) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	attsPerEpoch := benchmarkConfig(b).MaxAttestations
	committeeSize := (uint64(validatorCount) / slotsPerEpoch) / (attsPerEpoch / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = attsPerEpoch
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

	block := testutil.GenerateFullBlock(b, beaconState, privs, conf, beaconState.Slot)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		b.Error(err)
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: 4,
		Signatures:      true,
	}
	atts := []*ethpb.Attestation{}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2-1; i++ {
		block := testutil.GenerateFullBlock(b, beaconState, privs, attConfig, beaconState.Slot)
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		if err != nil {
			b.Error(err)
		}
	}

	block = testutil.GenerateFullBlock(b, beaconState, privs, attConfig, beaconState.Slot)
	block.Body.Attestations = append(atts, block.Body.Attestations...)
	b.Logf("%d\n", len(block.Body.Attestations))

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
	if err := ioutil.WriteFile("beaconState1Epoch.bytes", beaconBytes, 0644); err != nil {
		b.Fatal(err)
	}

	_, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		b.Fatal(err)
	}

	blockBytes, err := proto.Marshal(block)
	if err != nil {
		b.Fatal(err)
	}
	if err := ioutil.WriteFile("block128Atts.bytes", blockBytes, 0644); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkSaveGenesisBeaconState(b *testing.B) {
	deposits, _, _ := testutil.SetupInitialDeposits(b, uint64(validatorCount))
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		b.Fatal(err)
	}
	beaconBytes, err := proto.Marshal(genesisState)
	if err != nil {
		b.Fatal(err)
	}
	if err := ioutil.WriteFile("genesisState.bytes", beaconBytes, 0644); err != nil {
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

func beaconState1Epoch(b testing.TB) *pb.BeaconState {
	beaconBytes, err := ioutil.ReadFile("beaconState1Epoch.bytes")
	if err != nil {
		b.Fatal(err)
	}
	beaconState := &pb.BeaconState{}
	if err := proto.Unmarshal(beaconBytes, beaconState); err != nil {
		b.Fatal(err)
	}
	return beaconState
}

func fullBlock(b testing.TB) *ethpb.BeaconBlock {
	blockBytes, err := ioutil.ReadFile("block128Atts.bytes")
	if err != nil {
		b.Fatal(err)
	}
	beaconBlock := &ethpb.BeaconBlock{}
	if err := proto.Unmarshal(blockBytes, beaconBlock); err != nil {
		b.Fatal(err)
	}
	return beaconBlock
}
