package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	bench "github.com/prysmaticlabs/prysm/beacon-chain/core/state/benchmarks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func main() {
	log.Println("Generating genesis state")
	// Generating this for the 2 following states.
	if err := generateGenesisBeaconState(); err != nil {
		log.Fatal(err)
	}
	log.Println("Generating full block and state after 1 skipped epoch")
	if err := generateMarshalledFullStateAndBlock(); err != nil {
		log.Fatal(err)
	}
	log.Println("Generating state after 2 fully attested epochs")
	if err := generate2FullEpochState(); err != nil {
		log.Fatal(err)
	}
	// Removing this since its 10MB large and no longer needed.
	if err := os.Remove(bench.FilePath(bench.GenesisFileName)); err != nil {
		log.Fatal(err)
	}
}

func generateGenesisBeaconState() error {
	genesisState, _, err := interop.GenerateGenesisState(0, bench.ValidatorCount)
	if err != nil {
		return err
	}
	beaconBytes, err := ssz.Marshal(genesisState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(bench.FilePath(bench.GenesisFileName), beaconBytes, 0644); err != nil {
		return err
	}
	return nil
}

func generateMarshalledFullStateAndBlock() error {
	t := &testing.T{}
	bench.SetConfig()
	beaconState, err := genesisBeaconState()
	if err != nil {
		return err
	}

	privs, _, err := interop.DeterministicallyGenerateKeys(0, bench.ValidatorCount)
	if err != nil {
		return err
	}

	conf := &testutil.BlockGenConfig{
		MaxAttestations: 0,
		Signatures:      true,
	}

	// Small offset for the beacon state so we dont process a block on an epoch.
	slotOffset := uint64(2)
	block := testutil.GenerateFullBlock(t, beaconState, privs, conf, params.BeaconConfig().SlotsPerEpoch+slotOffset)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		return err
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: bench.AttestationsPerEpoch / params.BeaconConfig().SlotsPerEpoch,
		Signatures:      true,
	}

	atts := []*ethpb.Attestation{}
	for i := slotOffset + 1; i < params.BeaconConfig().SlotsPerEpoch+slotOffset; i++ {
		attsForSlot := testutil.GenerateAttestations(t, beaconState, privs, attConfig, i)
		atts = append(atts, attsForSlot...)
	}

	block = testutil.GenerateFullBlock(t, beaconState, privs, attConfig, beaconState.Slot)
	block.Body.Attestations = append(atts, block.Body.Attestations...)

	s, err := state.CalculateStateRoot(context.Background(), beaconState, block)
	if err != nil {
		return err
	}
	block.StateRoot = s[:]
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return err
	}
	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	beaconState.Slot++
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return err
	}
	domain := helpers.Domain(beaconState.Fork, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainBeaconProposer)
	block.Signature = privs[proposerIdx].Sign(blockRoot[:], domain).Marshal()
	beaconState.Slot--

	beaconBytes, err := ssz.Marshal(beaconState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(bench.FilePath(bench.BState1EpochFileName), beaconBytes, 0644); err != nil {
		return err
	}

	// Running a single state transition to make sure the generated files aren't broken.
	_, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		return err
	}

	blockBytes, err := ssz.Marshal(block)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(bench.FilePath(bench.FullBlockFileName), blockBytes, 0644); err != nil {
		return err
	}
	return nil
}

func generate2FullEpochState() error {
	t := &testing.T{}
	bench.SetConfig()
	beaconState, err := genesisBeaconState()
	if err != nil {
		return err
	}

	privs, _, err := interop.DeterministicallyGenerateKeys(0, bench.ValidatorCount)
	if err != nil {
		return err
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: bench.AttestationsPerEpoch / params.BeaconConfig().SlotsPerEpoch,
		Signatures:      true,
	}

	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2-1; i++ {
		block := testutil.GenerateFullBlock(t, beaconState, privs, attConfig, beaconState.Slot)
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		if err != nil {
			return err
		}
	}

	beaconBytes, err := ssz.Marshal(beaconState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(bench.FilePath(bench.BState2EpochFileName), beaconBytes, 0644); err != nil {
		return err
	}
	return nil
}

func genesisBeaconState() (*pb.BeaconState, error) {
	beaconBytes, err := ioutil.ReadFile(bench.FilePath(bench.GenesisFileName))
	if err != nil {
		return nil, err
	}
	genesisState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconBytes, genesisState); err != nil {
		return nil, err
	}
	return genesisState, nil
}
