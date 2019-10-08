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

func TestSavedStateKeys_GetsCorrectKeys(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	savingInterval := params.BeaconConfig().SavingInterval
	block := &ethpb.BeaconBlock{
		Slot: 0,
	}
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}

	block.Slot = 4
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}

	block.Slot = 7
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}

	stateKeys, err := db.savedStateKeys(context.Background(), 0, savingInterval)
	if err != nil {
		t.Fatal(err)
	}

	if len(stateKeys) > 1 {
		t.Fatalf("only should have returned the earliest block root, returned %d", len(stateKeys))
	}

	if !bytes.Equal(blockRoot[:], stateKeys[0]) {
		t.Fatalf(
			"expected saved state key to match block root, received: %#x, expected: %#x",
			blockRoot,
			stateKeys[0],
		)
	}
}

func TestSavedStateKeys_GetsCorrectKeys_SkippedInterval(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	savingInterval := params.BeaconConfig().SavingInterval
	// Skip over the first interval
	block := &ethpb.BeaconBlock{
		Slot: savingInterval + 3,
	}
	blockRoot1, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}

	block.Slot = savingInterval + 5
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}

	// Skip over 2 more intervals
	block.Slot = savingInterval*4 + 2
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}
	blockRoot2, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}

	block.Slot = savingInterval*4 + 4
	if err := db.SaveBlock(context.Background(), block); err != nil {
		t.Fatal(err)
	}

	stateKeys, err := db.savedStateKeys(context.Background(), 0, savingInterval*5)
	if err != nil {
		t.Fatal(err)
	}

	if len(stateKeys) != 2 {
		t.Fatalf("only should have returned the 2 block roots, returned %d", len(stateKeys))
	}

	if !bytes.Equal(blockRoot1[:], stateKeys[0]) {
		t.Fatalf(
			"expected saved state key to match block root 1, received: %#x, expected: %#x",
			blockRoot1,
			stateKeys[0],
		)
	}

	if !bytes.Equal(blockRoot2[:], stateKeys[1]) {
		t.Fatalf(
			"expected saved state key to match block root 2, received: %#x, expected: %#x",
			blockRoot2,
			stateKeys[0],
		)
	}
}

func TestGenerateStateAtSlot_GeneratesCorrectState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	savingInterval := params.BeaconConfig().SavingInterval
	deposits, _, privs := testutil.SetupInitialDeposits(t, 128)
	eth1Data := testutil.GenerateEth1Data(t, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(err)
	}

	conf := &testutil.BlockGenConfig{}
	firstBlock := testutil.GenerateFullBlock(t, genesisState, privs, conf)
	newState, err := state.ExecuteStateTransitionNoVerify(context.Background(), genesisState, firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), newState, root); err != nil {
		t.Fatal(err)
	}

	blocks := make([]*ethpb.BeaconBlock, savingInterval-1)
	blocks[0] = firstBlock
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(t, newState, privs, conf)
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
	generatedState, err := db.GenerateStateAtSlot(context.Background(), savingInterval-1)
	if err != nil {
		t.Fatal(err)
	}

	if !ssz.DeepEqual(generatedState, newState) {
		t.Fatal("expected generated state to deep equal actual state")
	}
}

func TestGenerateStateAtSlot_SkippedSavingSlot(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	savingInterval := params.BeaconConfig().SavingInterval
	deposits, _, privs := testutil.SetupInitialDeposits(t, 128)
	eth1Data := testutil.GenerateEth1Data(t, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(err)
	}

	conf := &testutil.BlockGenConfig{}
	firstBlock := testutil.GenerateFullBlock(t, genesisState, privs, conf)
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := state.ExecuteStateTransitionNoVerify(context.Background(), genesisState, firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), newState, root); err != nil {
		t.Fatal(err)
	}

	blocks := make([]*ethpb.BeaconBlock, savingInterval-1)
	blocks[0] = firstBlock
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(t, newState, privs, conf)
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
	skipBlock := testutil.GenerateFullBlock(t, newState, privs, conf)
	newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, skipBlock)
	if err != nil {
		t.Fatal(err)
	}
	block := testutil.GenerateFullBlock(t, newState, privs, conf)
	newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
	if err != nil {
		t.Fatal(err)
	}
	root, err = ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SaveState(context.Background(), newState, root); err != nil {
		t.Fatal(err)
	}

	blocks = make([]*ethpb.BeaconBlock, savingInterval-2)
	blocks[0] = block
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(t, newState, privs, conf)
		blocks[i] = block
		newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		t.Fatal(err)
	}

	// Slot starting from 1, after 7 blocks, a slot skipped, and
	// 6 more blocks, state slot should be 14.
	generatedState, err := db.GenerateStateAtSlot(context.Background(), (savingInterval*2)-2)
	if err != nil {
		t.Fatal(err)
	}

	if !ssz.DeepEqual(generatedState, newState) {
		t.Fatal("expected generated state to deep equal actual state")
	}
}

func TestGenerateStateAtSlot_SkippedSavingIntervalSlots(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	savingInterval := params.BeaconConfig().SavingInterval
	deposits, _, privs := testutil.SetupInitialDeposits(t, 128)
	eth1Data := testutil.GenerateEth1Data(t, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(err)
	}

	conf := &testutil.BlockGenConfig{}
	firstBlock := testutil.GenerateFullBlock(t, genesisState, privs, conf)
	newState, err := state.ExecuteStateTransitionNoVerify(context.Background(), genesisState, firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), newState, root); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), firstBlock); err != nil {
		t.Fatal(err)
	}

	// Process 14 slots to skip over the normal interval
	postSkipBlock := testutil.GenerateFullBlock(t, newState, privs, conf)
	postSkipBlock.Slot = savingInterval + (savingInterval - 2)
	newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, postSkipBlock)
	if err != nil {
		t.Fatal(err)
	}
	root, err = ssz.SigningRoot(postSkipBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), newState, root); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), postSkipBlock); err != nil {
		t.Fatal(err)
	}

	// Save one more block so we can generate a state that isn't latest.
	extraBlock := testutil.GenerateFullBlock(t, newState, privs, conf)
	root, err = ssz.SigningRoot(extraBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), extraBlock); err != nil {
		t.Fatal(err)
	}

	// Slot starting from 2, after 6 blocks, a slot skipped, and
	// 6 more blocks, state slot should be 14.
	generatedState, err := db.GenerateStateAtSlot(context.Background(), savingInterval+(savingInterval-2))
	if err != nil {
		t.Fatal(err)
	}

	if !ssz.DeepEqual(generatedState, newState) {
		t.Fatal("expected generated state to deep equal actual state")
	}
}

func BenchmarkGenerateStateAtSlot_WorstCase(b *testing.B) {
	db := setupDB(b)
	defer teardownDB(b, db)

	savingInterval := params.BeaconConfig().SavingInterval
	deposits, _, privs := testutil.SetupInitialDeposits(b, 2048)
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}

	blocks := make([]*ethpb.BeaconBlock, savingInterval-1)
	conf := &testutil.BlockGenConfig{
		MaxAttestations: 32,
	}
	firstBlock := testutil.GenerateFullBlock(b, genesisState, privs, conf)
	root, err := ssz.SigningRoot(firstBlock)
	if err != nil {
		b.Fatal(err)
	}
	newState, err := state.ExecuteStateTransitionNoVerify(context.Background(), genesisState, firstBlock)
	if err != nil {
		b.Fatal(err)
	}
	if err := db.SaveState(context.Background(), newState, root); err != nil {
		b.Fatal(err)
	}

	blocks[0] = firstBlock
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(b, newState, privs, conf)
		blocks[i] = block
		newState, err = state.ExecuteStateTransitionNoVerify(context.Background(), newState, block)
		if err != nil {
			b.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		b.Fatal(err)
	}

	// Slot after 5 blocks should be 7
	b.N = 10
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GenerateStateAtSlot(context.Background(), savingInterval-1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
