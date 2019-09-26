package kv

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 100}
	r := [32]byte{'A'}

	if err := db.SaveState(context.Background(), s, r); err != nil {
		t.Fatal(err)
	}

	savedS, err := db.State(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedS) {
		t.Error("did not retrieve saved state")
	}

	savedS, err = db.State(context.Background(), [32]byte{'B'})
	if err != nil {
		t.Fatal(err)
	}

	if savedS != nil {
		t.Error("unsaved state should've been nil")
	}
}

func TestHeadState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 100}
	headRoot := [32]byte{'A'}

	if err := db.SaveHeadBlockRoot(context.Background(), headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), s, headRoot); err != nil {
		t.Fatal(err)
	}

	savedHeadS, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedHeadS) {
		t.Error("did not retrieve saved state")
	}

	if err := db.SaveHeadBlockRoot(context.Background(), [32]byte{'B'}); err != nil {
		t.Fatal(err)
	}

	savedHeadS, err = db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if savedHeadS != nil {
		t.Error("unsaved head state should've been nil")
	}
}

func TestGenesisState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 1}
	headRoot := [32]byte{'B'}

	if err := db.SaveGenesisBlockRoot(context.Background(), headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), s, headRoot); err != nil {
		t.Fatal(err)
	}

	savedGenesisS, err := db.GenesisState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedGenesisS) {
		t.Error("did not retrieve saved state")
	}

	if err := db.SaveGenesisBlockRoot(context.Background(), [32]byte{'C'}); err != nil {
		t.Fatal(err)
	}

	savedGenesisS, err = db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if savedGenesisS != nil {
		t.Error("unsaved genesis state should've been nil")
	}
}

func TestGenerateStateAtSlot_GeneratesCorrectState(t *testing.T) {
	c := params.BeaconConfig()
	c.MaxAttestations = 2
	c.MaxAttesterSlashings = 0
	c.MaxProposerSlashings = 0
	c.MaxDeposits = 0
	c.MaxVoluntaryExits = 0
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.BeaconConfig())

	db := setupDB(t)
	defer teardownDB(t, db)

	deposits, privs := testutil.SetupInitialDeposits(t, 128)
	eth1Data := testutil.GenerateEth1Data(t, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(err)
	}
	genesisState.Slot = 7

	blocks := make([]*ethpb.BeaconBlock, 6)
	firstBlock := testutil.GenerateFullBlock(t, genesisState, privs)
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), genesisState, root); err != nil {
		t.Fatal(err)
	}

	blocks[0] = firstBlock
	newState, err := state.ExecuteStateTransitionForStateRoot(context.Background(), genesisState, firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(t, newState, privs)
		blocks[i] = block
		newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		t.Fatal(err)
	}

	// Slot after 6 blocks should be 7
	generatedState, err := db.GenerateStateAtSlot(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}

	if generatedState.Slot != newState.Slot {
		t.Fatalf(
			"expected generated state slot %d to equal actual state slot %d",
			generatedState.Slot,
			newState.Slot,
		)
	}

	if !bytes.Equal(generatedState.LatestBlockHeader.StateRoot, newState.LatestBlockHeader.StateRoot) {
		t.Fatalf(
			"expected generated state slot %d to equal actual state slot %d",
			generatedState.Slot,
			newState.Slot,
		)
	}

	if !ssz.DeepEqual(generatedState, newState) {
		t.Fatal("expected generated state to deep equal actual state")
	}
}

func TestGenerateStateAtSlot_SkippedSavingSlot(t *testing.T) {
	c := params.BeaconConfig()
	c.MaxAttestations = 2
	c.MaxAttesterSlashings = 0
	c.MaxProposerSlashings = 0
	c.MaxDeposits = 0
	c.MaxVoluntaryExits = 0
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.BeaconConfig())

	db := setupDB(t)
	defer teardownDB(t, db)

	deposits, privs := testutil.SetupInitialDeposits(t, 128)
	eth1Data := testutil.GenerateEth1Data(t, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(err)
	}
	genesisState.Slot = 1

	blocks := make([]*ethpb.BeaconBlock, 6)
	firstBlock := testutil.GenerateFullBlock(t, genesisState, privs)
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), genesisState, root); err != nil {
		t.Fatal(err)
	}

	blocks[0] = firstBlock
	newState, err := state.ExecuteStateTransitionForStateRoot(context.Background(), genesisState, firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(t, newState, privs)
		blocks[i] = block
		newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		t.Fatal(err)
	}

	// Slot at this point is 7, so we generate 2 blocks but only save the second one to simulate that the saving slot was skipped.
	skipBlock := testutil.GenerateFullBlock(t, newState, privs)
	newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, skipBlock)
	if err != nil {
		t.Fatal(err)
	}
	block := testutil.GenerateFullBlock(t, newState, privs)
	newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d\n", newState.Slot)
	root, err = ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SaveState(context.Background(), newState, root); err != nil {
		t.Fatal(err)
	}

	blocks = make([]*ethpb.BeaconBlock, 6)
	blocks[0] = block
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(t, newState, privs)
		blocks[i] = block
		newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		t.Fatal(err)
	}

	// Slot starting from 2, after 6 blocks, a slot skipped, and
	// 6 more blocks, state slot should be 15.
	t.Logf("%d\n", newState.Slot)
	generatedState, err := db.GenerateStateAtSlot(context.Background(), 14)
	if err != nil {
		t.Fatal(err)
	}

	if generatedState.Slot != newState.Slot {
		t.Fatalf(
			"expected generated state slot %d to equal actual state slot %d",
			generatedState.Slot,
			newState.Slot,
		)
	}

	if !bytes.Equal(generatedState.LatestBlockHeader.StateRoot, newState.LatestBlockHeader.StateRoot) {
		t.Fatalf(
			"expected generated state slot %d to equal actual state slot %d",
			generatedState.Slot,
			newState.Slot,
		)
	}

	if !ssz.DeepEqual(generatedState, newState) {
		t.Fatal("expected generated state to deep equal actual state")
	}
}

func BenchmarkGenerateStateAtSlot(b *testing.B) {
	defer params.OverrideBeaconConfig(params.BeaconConfig())
	c := params.BeaconConfig()
	c.MaxAttestations = 4
	c.MaxAttesterSlashings = 0
	c.MaxProposerSlashings = 0
	c.MaxDeposits = 0
	c.MaxVoluntaryExits = 0
	params.OverrideBeaconConfig(c)

	db := setupDB(b)
	defer teardownDB(b, db)

	deposits, privs := testutil.SetupInitialDeposits(b, 256)
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}
	genesisState.Slot = 1

	blocks := make([]*ethpb.BeaconBlock, 6)
	firstBlock := testutil.GenerateFullBlock(b, genesisState, privs)
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		b.Fatal(err)
	}
	if err := db.SaveState(context.Background(), genesisState, root); err != nil {
		b.Fatal(err)
	}

	newState, err := state.ExecuteStateTransitionForStateRoot(context.Background(), genesisState, firstBlock)
	if err != nil {
		b.Fatal(err)
	}
	blocks[0] = firstBlock
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(b, newState, privs)
		blocks[i] = block
		newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
		if err != nil {
			b.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		b.Fatal(err)
	}

	// Slot after 8 blocks should be 9
	b.N = 10
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GenerateStateAtSlot(context.Background(), 7)
		if err != nil {
			b.Fatal(err)
		}
	}
}
