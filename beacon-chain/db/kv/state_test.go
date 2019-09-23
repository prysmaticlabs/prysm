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

func TestGenerateStateAtSlot(b *testing.T) {
	c := params.BeaconConfig()
	c.MaxAttestations = 2
	c.MaxAttesterSlashings = 0
	c.MaxProposerSlashings = 0
	c.MaxDeposits = 0
	c.MaxVoluntaryExits = 0
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.BeaconConfig())

	db := setupDB(b)
	defer teardownDB(b, db)

	deposits, privs := testutil.SetupInitialDeposits(b, 128)
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}
	genesisState.Slot = 1

	root, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		b.Fatal(err)
	}

	if err := db.SaveState(context.Background(), genesisState, root); err != nil {
		b.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{
		Epoch: 0,
		Root:  root[:],
	}
	if err := db.SaveFinalizedCheckpoint(context.Background(), checkPoint); err != nil {
		b.Fatal(err)
	}

	checkPoint = &ethpb.Checkpoint{
		Epoch: 1,
		Root:  params.BeaconConfig().ZeroHash[:],
	}
	if err := db.SaveJustifiedCheckpoint(context.Background(), checkPoint); err != nil {
		b.Fatal(err)
	}

	blocks := make([]*ethpb.BeaconBlock, 8)
	block := testutil.GenerateFullBlock(b, genesisState, privs)
	newState, err := state.ExecuteStateTransition(context.Background(), genesisState, block)
	if err != nil {
		b.Fatal(err)
	}
	blocks[0] = block
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(b, newState, privs)
		blocks[i] = block
		newState, err = state.ExecuteStateTransition(context.Background(), newState, block)
		if err != nil {
			b.Fatal(err)
		}
	}
	if err := db.SaveBlocks(context.Background(), blocks); err != nil {
		b.Fatal(err)
	}

	// Slot after 8 blocks should be 9
	generatedState, err := db.GenerateStateAtSlot(context.Background(), 9)
	if err != nil {
		b.Fatal(err)
	}

	if generatedState.Slot != newState.Slot {
		b.Fatalf(
			"expected generated state slot %d to equal actual state slot %d",
			generatedState.Slot,
			newState.Slot,
		)
	}

	if !bytes.Equal(generatedState.LatestBlockHeader.StateRoot, newState.LatestBlockHeader.StateRoot) {
		b.Fatalf(
			"expected generated state slot %d to equal actual state slot %d",
			generatedState.Slot,
			newState.Slot,
		)
	}
}

func BenchmarkGenerateStateAtSlot(b *testing.B) {
	c := params.BeaconConfig()
	c.MaxAttestations = 2
	c.MaxAttesterSlashings = 0
	c.MaxProposerSlashings = 0
	c.MaxDeposits = 0
	c.MaxVoluntaryExits = 0
	params.OverrideBeaconConfig(c)
	defer params.OverrideBeaconConfig(params.BeaconConfig())

	db := setupDB(b)
	defer teardownDB(b, db)

	deposits, privs := testutil.SetupInitialDeposits(b, 128)
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}
	genesisState.Slot = 1

	root, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		b.Fatal(err)
	}

	if err := db.SaveState(context.Background(), genesisState, root); err != nil {
		b.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{
		Epoch: 0,
		Root:  root[:],
	}
	if err := db.SaveFinalizedCheckpoint(context.Background(), checkPoint); err != nil {
		b.Fatal(err)
	}

	checkPoint = &ethpb.Checkpoint{
		Epoch: 1,
		Root:  params.BeaconConfig().ZeroHash[:],
	}
	if err := db.SaveJustifiedCheckpoint(context.Background(), checkPoint); err != nil {
		b.Fatal(err)
	}

	blocks := make([]*ethpb.BeaconBlock, 16)
	block := testutil.GenerateFullBlock(b, genesisState, privs)
	newState, err := state.ExecuteStateTransition(context.Background(), genesisState, block)
	if err != nil {
		b.Fatal(err)
	}
	blocks[0] = block
	for i := 1; i < len(blocks); i++ {
		block := testutil.GenerateFullBlock(b, newState, privs)
		blocks[i] = block
		newState, err = state.ExecuteStateTransition(context.Background(), newState, block)
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
		_, err := db.GenerateStateAtSlot(context.Background(), genesisState.Slot+12)
		if err != nil {
			b.Fatal(err)
		}
	}
}
