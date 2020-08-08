package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)

	r := [32]byte{'A'}

	require.Equal(t, false, db.HasState(context.Background(), r))

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(100))

	require.NoError(t, db.SaveState(context.Background(), st, r))
	assert.Equal(t, true, db.HasState(context.Background(), r))

	savedS, err := db.State(context.Background(), r)
	require.NoError(t, err)

	if !reflect.DeepEqual(st.InnerStateUnsafe(), savedS.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(st.InnerStateUnsafe(), savedS.InnerStateUnsafe())
		t.Errorf("Did not retrieve saved state: %v", diff)
	}

	savedS, err = db.State(context.Background(), [32]byte{'B'})
	require.NoError(t, err)
	assert.Equal(t, (*state.BeaconState)(nil), savedS, "Unsaved state should've been nil")
}

func TestHeadState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)

	headRoot := [32]byte{'A'}

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(100))
	require.NoError(t, db.SaveState(context.Background(), st, headRoot))
	require.NoError(t, db.SaveHeadBlockRoot(context.Background(), headRoot))

	savedHeadS, err := db.HeadState(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, st.InnerStateUnsafe(), savedHeadS.InnerStateUnsafe(), "Did not retrieve saved state")
}

func TestGenesisState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)

	headRoot := [32]byte{'B'}

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), headRoot))
	require.NoError(t, db.SaveState(context.Background(), st, headRoot))

	savedGenesisS, err := db.GenesisState(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, st.InnerStateUnsafe(), savedGenesisS.InnerStateUnsafe(), "Did not retrieve saved state")
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), [32]byte{'C'}))

	savedGenesisS, err = db.HeadState(context.Background())
	require.NoError(t, err)
	assert.Equal(t, (*state.BeaconState)(nil), savedGenesisS, "Unsaved genesis state should've been nil")
}

func TestStore_StatesBatchDelete(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	numBlocks := 100
	totalBlocks := make([]*ethpb.SignedBeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	evenBlockRoots := make([][32]byte, 0)
	for i := 0; i < len(totalBlocks); i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = uint64(i)
		totalBlocks[i] = b
		r, err := stateutil.BlockRoot(totalBlocks[i].Block)
		require.NoError(t, err)
		st := testutil.NewBeaconState()
		require.NoError(t, st.SetSlot(uint64(i)))
		require.NoError(t, db.SaveState(context.Background(), st, r))
		blockRoots = append(blockRoots, r)
		if i%2 == 0 {
			evenBlockRoots = append(evenBlockRoots, r)
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, totalBlocks))
	// We delete all even indexed states.
	require.NoError(t, db.DeleteStates(ctx, evenBlockRoots))
	// When we retrieve the data, only the odd indexed state should remain.
	for _, r := range blockRoots {
		s, err := db.State(context.Background(), r)
		require.NoError(t, err)
		if s == nil {
			continue
		}
		assert.Equal(t, uint64(1), s.Slot()%2, "State with slot %d should have been deleted", s.Slot())
	}
}

func TestStore_DeleteGenesisState(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisBlockRoot := [32]byte{'A'}
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(100))
	require.NoError(t, db.SaveState(ctx, st, genesisBlockRoot))
	wantedErr := "cannot delete genesis, finalized, or head state"
	assert.ErrorContains(t, wantedErr, db.DeleteState(ctx, genesisBlockRoot))
}

func TestStore_DeleteFinalizedState(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: genesis[:],
			Slot:       100,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, blk))

	finalizedBlockRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)

	finalizedState := testutil.NewBeaconState()
	require.NoError(t, finalizedState.SetSlot(100))
	require.NoError(t, db.SaveState(ctx, finalizedState, finalizedBlockRoot))
	finalizedCheckpoint := &ethpb.Checkpoint{Root: finalizedBlockRoot[:]}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, finalizedCheckpoint))
	wantedErr := "cannot delete genesis, finalized, or head state"
	assert.ErrorContains(t, wantedErr, db.DeleteState(ctx, finalizedBlockRoot))
}

func TestStore_DeleteHeadState(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: genesis[:],
			Slot:       100,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, blk))

	headBlockRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(100))
	require.NoError(t, db.SaveState(ctx, st, headBlockRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headBlockRoot))
	wantedErr := "cannot delete genesis, finalized, or head state"
	assert.ErrorContains(t, wantedErr, db.DeleteState(ctx, headBlockRoot))
}

func TestStore_SaveDeleteState_CanGetHighestBelow(t *testing.T) {
	db := setupDB(t)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	r, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), b))
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(1))
	s0 := st.InnerStateUnsafe()
	require.NoError(t, db.SaveState(context.Background(), st, r))

	b = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 100}}
	r1, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), b))
	st = testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(100))
	s1 := st.InnerStateUnsafe()
	require.NoError(t, db.SaveState(context.Background(), st, r1))

	b = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1000}}
	r2, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), b))
	st = testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(1000))
	s2 := st.InnerStateUnsafe()

	require.NoError(t, db.SaveState(context.Background(), st, r2))

	highest, err := db.HighestSlotStatesBelow(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(highest[0].InnerStateUnsafe(), s0), "Did not retrieve saved state: %v != %v", highest, s0)

	highest, err = db.HighestSlotStatesBelow(context.Background(), 101)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(highest[0].InnerStateUnsafe(), s1), "Did not retrieve saved state: %v != %v", highest, s1)

	highest, err = db.HighestSlotStatesBelow(context.Background(), 1001)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(highest[0].InnerStateUnsafe(), s2), "Did not retrieve saved state: %v != %v", highest, s2)
}

func TestStore_GenesisState_CanGetHighestBelow(t *testing.T) {
	db := setupDB(t)

	genesisState := testutil.NewBeaconState()
	genesisRoot := [32]byte{'a'}
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), genesisRoot))
	require.NoError(t, db.SaveState(context.Background(), genesisState, genesisRoot))

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	r, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), b))

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(1))
	require.NoError(t, db.SaveState(context.Background(), st, r))

	highest, err := db.HighestSlotStatesBelow(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(highest[0].InnerStateUnsafe(), st.InnerStateUnsafe()))

	highest, err = db.HighestSlotStatesBelow(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(highest[0].InnerStateUnsafe(), genesisState.InnerStateUnsafe()))
	highest, err = db.HighestSlotStatesBelow(context.Background(), 0)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(highest[0].InnerStateUnsafe(), genesisState.InnerStateUnsafe()))
}
