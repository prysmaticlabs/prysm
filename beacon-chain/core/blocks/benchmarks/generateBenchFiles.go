package benchmarks

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func main() {
	log.Println("generating genesis state")
	if err := generateGenesisBeaconState(); err != nil {
		log.Fatal(err)
	}
	log.Println("generating full block and state after 1 skipped epoch")
	if err := generateMarshalledFullStateAndBlock(); err != nil {
		log.Fatal(err)
	}
	log.Println("generating state after 2 fully attested epochs")
	if err := generate2FullEpochState(); err != nil {
		log.Fatal(err)
	}
}

func generateGenesisBeaconState() error {
	t := &testing.T{}
	deposits, _, _ := testutil.SetupInitialDeposits(t, uint64(validatorCount))
	eth1Data := testutil.GenerateEth1Data(t, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		return err
	}
	beaconBytes, err := ssz.Marshal(genesisState)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile("genesisState.ssz", beaconBytes, 0644); err != nil {
		return err
	}
	return nil
}

func generateMarshalledFullStateAndBlock() error {
	t := &testing.T{}
	setConfig()
	beaconState, err := genesisBeaconState()
	if err != nil {
		return err
	}

	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		return err
	}

	conf := &testutil.BlockGenConfig{
		MaxAttestations: 0,
		Signatures:      true,
	}

	block := testutil.GenerateFullBlock(t, beaconState, privs, conf, params.BeaconConfig().SlotsPerEpoch+2)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		return err
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: 4,
		Signatures:      true,
	}
	atts := []*ethpb.Attestation{}
	for i := uint64(3); i < params.BeaconConfig().SlotsPerEpoch+2; i++ {
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
	if err := ioutil.WriteFile("beaconState1Epoch.ssz", beaconBytes, 0644); err != nil {
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
	if err := ioutil.WriteFile("block128Atts.ssz", blockBytes, 0644); err != nil {
		return err
	}
	return nil
}

func generate2FullEpochState() error {
	t := &testing.T{}
	setConfig()
	beaconState, err := genesisBeaconState()
	if err != nil {
		return err
	}

	privs, _, err := interop.DeterministicallyGenerateKeys(0, uint64(validatorCount))
	if err != nil {
		return err
	}

	attConfig := &testutil.BlockGenConfig{
		MaxAttestations: 4,
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
	if err := ioutil.WriteFile("beaconState2FullEpochs.ssz", beaconBytes, 0644); err != nil {
		return err
	}
	return nil
}

func genesisBeaconState() (*pb.BeaconState, error) {
	beaconBytes, err := ioutil.ReadFile("genesisState.ssz")
	if err != nil {
		return nil, err
	}
	genesisState := &pb.BeaconState{}
	if err := ssz.Unmarshal(beaconBytes, genesisState); err != nil {
		return nil, err
	}
	return genesisState, nil
}
