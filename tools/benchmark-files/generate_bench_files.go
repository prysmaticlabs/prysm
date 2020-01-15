package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/benchutil"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Output dir is: %s", wd)
	log.Println("Generating genesis state")
	// Generating this for the 2 following states.
	if err := generateGenesisBeaconState(); err != nil {
		log.Fatalf("Could not generate genesis state: %v", err)
	}
	log.Println("Generating full block and state after 1 skipped epoch")
	if err := generateMarshalledFullStateAndBlock(); err != nil {
		log.Fatalf("Could not generate full state and block: %v", err)
	}
	log.Println("Generating state after 2 fully attested epochs")
	if err := generate2FullEpochState(); err != nil {
		log.Fatalf("Could not generate 2 full epoch state: %v", err)
	}
	// Removing this since its 10MB large and no longer needed.
	if err := os.Remove(benchutil.GenesisFileName); err != nil {
		log.Fatal(err)
	}
}

func generateGenesisBeaconState() error {
	genesisState, _, err := interop.GenerateGenesisState(0, benchutil.ValidatorCount)
	if err != nil {
		return err
	}
	beaconBytes, err := proto.Marshal(genesisState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(benchutil.GenesisFileName, beaconBytes, 0644); err != nil {
		return err
	}
	return nil
}

func generateMarshalledFullStateAndBlock() error {
	benchutil.SetBenchmarkConfig()
	beaconState, err := genesisBeaconState()
	if err != nil {
		return err
	}

	privs, _, err := interop.DeterministicallyGenerateKeys(0, benchutil.ValidatorCount)
	if err != nil {
		return err
	}

	conf := &testutil.BlockGenConfig{}
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	// Small offset for the beacon state so we dont process a block on an epoch.
	slotOffset := uint64(2)
	block, err := testutil.GenerateFullBlock(beaconState, privs, conf, slotsPerEpoch+slotOffset)
	if err != nil {
		return err
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		return err
	}

	attConfig := &testutil.BlockGenConfig{
		NumAttestations: benchutil.AttestationsPerEpoch / slotsPerEpoch,
	}

	atts := []*ethpb.Attestation{}
	for i := slotOffset + 1; i < slotsPerEpoch+slotOffset; i++ {
		attsForSlot, err := testutil.GenerateAttestations(beaconState, privs, attConfig.NumAttestations, i)
		if err != nil {
			return err
		}
		atts = append(atts, attsForSlot...)
	}

	block, err = testutil.GenerateFullBlock(beaconState, privs, attConfig, beaconState.Slot)
	if err != nil {
		return errors.Wrap(err, "could not generate full block")
	}
	block.Block.Body.Attestations = append(atts, block.Block.Body.Attestations...)

	s, err := state.CalculateStateRoot(context.Background(), beaconState, block)
	if err != nil {
		return errors.Wrap(err, "could not calculate state root")
	}
	block.Block.StateRoot = s[:]
	blockRoot, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root of block")
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

	beaconBytes, err := proto.Marshal(beaconState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(benchutil.BState1EpochFileName, beaconBytes, 0644); err != nil {
		return err
	}

	// Running a single state transition to make sure the generated files aren't broken.
	_, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		return err
	}

	blockBytes, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(benchutil.FullBlockFileName, blockBytes, 0644); err != nil {
		return err
	}
	return nil
}

func generate2FullEpochState() error {
	benchutil.SetBenchmarkConfig()
	beaconState, err := genesisBeaconState()
	if err != nil {
		return err
	}

	privs, _, err := interop.DeterministicallyGenerateKeys(0, benchutil.ValidatorCount)
	if err != nil {
		return err
	}

	attConfig := &testutil.BlockGenConfig{
		NumAttestations: benchutil.AttestationsPerEpoch / params.BeaconConfig().SlotsPerEpoch,
	}

	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2-1; i++ {
		block, err := testutil.GenerateFullBlock(beaconState, privs, attConfig, beaconState.Slot)
		if err != nil {
			return err
		}
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		if err != nil {
			return err
		}
	}

	beaconBytes, err := proto.Marshal(beaconState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(benchutil.BState2EpochFileName, beaconBytes, 0644); err != nil {
		return err
	}
	return nil
}

func genesisBeaconState() (*pb.BeaconState, error) {
	beaconBytes, err := ioutil.ReadFile(benchutil.GenesisFileName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read genesis state file")
	}
	genesisState := &pb.BeaconState{}
	if err := proto.Unmarshal(beaconBytes, genesisState); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal genesis state file")
	}
	return genesisState, nil
}