package testutil

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGenerateFullBlock_PassesStateTransition(t *testing.T) {
	beaconState, privs := DeterministicGenesisState(t, 128)
	conf := &BlockGenConfig{
		NumAttestations: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)
}

func TestGenerateFullBlock_ThousandValidators(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	beaconState, privs := DeterministicGenesisState(t, 1024)
	conf := &BlockGenConfig{
		NumAttestations: 4,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)
}

func TestGenerateFullBlock_Passes4Epochs(t *testing.T) {
	// Changing to minimal config as this will process 4 epochs of blocks.
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	beaconState, privs := DeterministicGenesisState(t, 64)

	conf := &BlockGenConfig{
		NumAttestations: 2,
	}
	finalSlot := params.BeaconConfig().SlotsPerEpoch*4 + 3
	for i := 0; i < int(finalSlot); i++ {
		block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
		require.NoError(t, err)
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		require.NoError(t, err)
	}

	// Blocks are one slot ahead of beacon state.
	if finalSlot != beaconState.Slot() {
		t.Fatalf("expected output slot to be %d, received %d", finalSlot, beaconState.Slot())
	}
	if beaconState.CurrentJustifiedCheckpoint().Epoch != 3 {
		t.Fatalf("expected justified epoch to change to 3, received %d", beaconState.CurrentJustifiedCheckpoint().Epoch)
	}
	if beaconState.FinalizedCheckpointEpoch() != 2 {
		t.Fatalf("expected finalized epoch to change to 2, received %d", beaconState.CurrentJustifiedCheckpoint().Epoch)
	}
}

func TestGenerateFullBlock_ValidProposerSlashings(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	beaconState, privs := DeterministicGenesisState(t, 32)
	conf := &BlockGenConfig{
		NumProposerSlashings: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot()+1)
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)

	slashableIndice := block.Block.Body.ProposerSlashings[0].Header_1.Header.ProposerIndex
	if val, err := beaconState.ValidatorAtIndexReadOnly(slashableIndice); err != nil || !val.Slashed() {
		require.NoError(t, err)
		t.Fatal("expected validator to be slashed")
	}
}

func TestGenerateFullBlock_ValidAttesterSlashings(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	beaconState, privs := DeterministicGenesisState(t, 32)
	conf := &BlockGenConfig{
		NumAttesterSlashings: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)

	slashableIndices := block.Block.Body.AttesterSlashings[0].Attestation_1.AttestingIndices
	if val, err := beaconState.ValidatorAtIndexReadOnly(slashableIndices[0]); err != nil || !val.Slashed() {
		require.NoError(t, err)
		t.Fatal("expected validator to be slashed")
	}
}

func TestGenerateFullBlock_ValidAttestations(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	beaconState, privs := DeterministicGenesisState(t, 256)
	conf := &BlockGenConfig{
		NumAttestations: 4,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)
	if len(beaconState.CurrentEpochAttestations()) != 4 {
		t.Fatal("expected 4 attestations to be saved to the beacon state")
	}
}

func TestGenerateFullBlock_ValidDeposits(t *testing.T) {
	beaconState, privs := DeterministicGenesisState(t, 256)
	deposits, _, err := DeterministicDepositsAndKeys(257)
	require.NoError(t, err)
	eth1Data, err := DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	require.NoError(t, beaconState.SetEth1Data(eth1Data))
	conf := &BlockGenConfig{
		NumDeposits: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)

	depositedPubkey := block.Block.Body.Deposits[0].Data.PublicKey
	valIndexMap := stateutils.ValidatorIndexMap(beaconState.Validators())
	index := valIndexMap[bytesutil.ToBytes48(depositedPubkey)]
	val, err := beaconState.ValidatorAtIndexReadOnly(index)
	require.NoError(t, err)
	if val.EffectiveBalance() != params.BeaconConfig().MaxEffectiveBalance {
		t.Fatalf(
			"expected validator balance to be max effective balance, received %d",
			val.EffectiveBalance(),
		)
	}
}

func TestGenerateFullBlock_ValidVoluntaryExits(t *testing.T) {
	beaconState, privs := DeterministicGenesisState(t, 256)
	// Moving the state 2048 epochs forward due to PERSISTENT_COMMITTEE_PERIOD.
	err := beaconState.SetSlot(3 + params.BeaconConfig().ShardCommitteePeriod*params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, err)
	conf := &BlockGenConfig{
		NumVoluntaryExits: 1,
	}
	block, err := GenerateFullBlock(beaconState, privs, conf, beaconState.Slot())
	require.NoError(t, err)
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	require.NoError(t, err)

	exitedIndex := block.Block.Body.VoluntaryExits[0].Exit.ValidatorIndex

	val, err := beaconState.ValidatorAtIndexReadOnly(exitedIndex)
	require.NoError(t, err)
	if val.ExitEpoch() == params.BeaconConfig().FarFutureEpoch {
		t.Fatal("expected exiting validator index to be marked as exiting")
	}
}
