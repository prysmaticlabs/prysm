package testutil

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenerateFullBlock_PassesStateTransition(t *testing.T) {
	beaconState, privs, err := DeterministicGenesisState(128)
	if err != nil {
		t.Fatal(err)
	}
	conf := &BlockGenConfig{
		NumAttestations: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateFullBlock_ThousandValidators(t *testing.T) {
	helpers.ClearAllCaches()
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	defer params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privs, err := DeterministicGenesisState(1024)
	if err != nil {
		t.Fatal(err)
	}
	conf := &BlockGenConfig{
		NumAttestations: 4,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateFullBlock_Passes4Epochs(t *testing.T) {
	helpers.ClearAllCaches()
	// Changing to minimal config as this will process 4 epochs of blocks.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	defer params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privs, err := DeterministicGenesisState(64)
	if err != nil {
		t.Fatal(err)
	}

	conf := &BlockGenConfig{
		NumAttestations: 4,
	}
	finalSlot := params.BeaconConfig().SlotsPerEpoch*4 + 3
	for i := 0; i < int(finalSlot); i++ {
		block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
		if err != nil {
			t.Fatal(err)
		}
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Blocks are one slot ahead of beacon state.
	if finalSlot != beaconState.Slot {
		t.Fatalf("expected output slot to be %d, received %d", finalSlot, beaconState.Slot)
	}
	if beaconState.CurrentJustifiedCheckpoint.Epoch != 3 {
		t.Fatalf("expected justified epoch to change to 3, received %d", beaconState.CurrentJustifiedCheckpoint.Epoch)
	}
	if beaconState.FinalizedCheckpoint.Epoch != 2 {
		t.Fatalf("expected finalized epoch to change to 2, received %d", beaconState.CurrentJustifiedCheckpoint.Epoch)
	}
}

func TestGenerateFullBlock_ValidProposerSlashings(t *testing.T) {
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	defer params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privs, err := DeterministicGenesisState(32)
	if err != nil {
		t.Fatal(err)
	}
	conf := &BlockGenConfig{
		NumProposerSlashings: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot+1)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	slashableIndice := block.Body.ProposerSlashings[0].ProposerIndex
	if !beaconState.Validators[slashableIndice].Slashed {
		t.Fatal("expected validator to be slashed")
	}
}

func TestGenerateFullBlock_ValidAttesterSlashings(t *testing.T) {
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	defer params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privs, err := DeterministicGenesisState(32)
	if err != nil {
		t.Fatal(err)
	}
	conf := &BlockGenConfig{
		NumAttesterSlashings: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	slashableIndices := block.Body.AttesterSlashings[0].Attestation_1.CustodyBit_0Indices
	if !beaconState.Validators[slashableIndices[0]].Slashed {
		t.Fatal("expected validator to be slashed")
	}
}

func TestGenerateFullBlock_ValidAttestations(t *testing.T) {
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	defer params.OverrideBeaconConfig(params.MainnetConfig())
	helpers.ClearAllCaches()

	beaconState, privs, err := DeterministicGenesisState(256)
	if err != nil {
		t.Fatal(err)
	}
	conf := &BlockGenConfig{
		NumAttestations: 4,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}
	if len(beaconState.CurrentEpochAttestations) != 4 {
		t.Fatal("expected 4 attestations to be saved to the beacon state")
	}
}

func TestGenerateFullBlock_ValidDeposits(t *testing.T) {
	beaconState, privs, err := DeterministicGenesisState(256)
	if err != nil {
		t.Fatal(err)
	}
	deposits, _, err := DeterministicDepositsAndKeys(257)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Eth1Data = eth1Data
	conf := &BlockGenConfig{
		NumDeposits: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	depositedPubkey := block.Body.Deposits[0].Data.PublicKey
	valIndexMap := stateutils.ValidatorIndexMap(beaconState)
	index := valIndexMap[bytesutil.ToBytes48(depositedPubkey)]
	if beaconState.Validators[index].EffectiveBalance != params.BeaconConfig().MaxEffectiveBalance {
		t.Fatalf(
			"expected validator balance to be max effective balance, received %d",
			beaconState.Validators[index].EffectiveBalance,
		)
	}
}

func TestGenerateFullBlock_ValidVoluntaryExits(t *testing.T) {
	beaconState, privs, err := DeterministicGenesisState(256)
	if err != nil {
		t.Fatal(err)
	}
	// Moving the state 2048 epochs forward due to PERSISTENT_COMMITTEE_PERIOD.
	beaconState.Slot = 3 + params.BeaconConfig().PersistentCommitteePeriod*params.BeaconConfig().SlotsPerEpoch
	conf := &BlockGenConfig{
		NumVoluntaryExits: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	exitedIndex := block.Body.VoluntaryExits[0].ValidatorIndex
	if beaconState.Validators[exitedIndex].ExitEpoch == params.BeaconConfig().FarFutureEpoch {
		t.Fatal("expected exiting validator index to be marked as exiting")
	}
}
