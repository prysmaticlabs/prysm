package blockchain

import (
	"bytes"
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveHead_Same(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	service := setupBeaconChain(t, db, sc)

	r := [32]byte{'A'}
	service.head = &head{slot: 0, root: r}

	require.NoError(t, service.saveHead(context.Background(), r))
	assert.Equal(t, uint64(0), service.headSlot(), "Head did not stay the same")
	assert.Equal(t, r, service.headRoot(), "Head did not stay the same")
}

func TestSaveHead_Different(t *testing.T) {
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)
	service := setupBeaconChain(t, db, sc)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	newHeadSignedBlock := testutil.NewBeaconBlock()
	newHeadSignedBlock.Block.Slot = 1
	newHeadBlock := newHeadSignedBlock.Block

	require.NoError(t, service.beaconDB.SaveBlock(context.Background(), newHeadSignedBlock))
	newRoot, err := stateutil.BlockRoot(newHeadBlock)
	require.NoError(t, err)
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(1))
	require.NoError(t, service.beaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Slot: 1, Root: newRoot[:]}))
	require.NoError(t, service.beaconDB.SaveState(context.Background(), headState, newRoot))
	require.NoError(t, service.saveHead(context.Background(), newRoot))

	assert.Equal(t, uint64(1), service.HeadSlot(), "Head did not change")

	cachedRoot, err := service.HeadRoot(context.Background())
	require.NoError(t, err)
	if !bytes.Equal(cachedRoot, newRoot[:]) {
		t.Error("Head did not change")
	}
	assert.DeepEqual(t, newHeadSignedBlock, service.headBlock(), "Head did not change")
	assert.DeepEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
}

func TestSaveHead_Different_Reorg(t *testing.T) {
	ctx := context.Background()
	hook := logTest.NewGlobal()
	db, sc := testDB.SetupDB(t)
	service := setupBeaconChain(t, db, sc)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	reorgChainParent := [32]byte{'B'}
	newHeadSignedBlock := testutil.NewBeaconBlock()
	newHeadSignedBlock.Block.Slot = 1
	newHeadSignedBlock.Block.ParentRoot = reorgChainParent[:]
	newHeadBlock := newHeadSignedBlock.Block

	require.NoError(t, service.beaconDB.SaveBlock(context.Background(), newHeadSignedBlock))
	newRoot, err := stateutil.BlockRoot(newHeadBlock)
	require.NoError(t, err)
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(1))
	require.NoError(t, service.beaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Slot: 1, Root: newRoot[:]}))
	require.NoError(t, service.beaconDB.SaveState(context.Background(), headState, newRoot))
	require.NoError(t, service.saveHead(context.Background(), newRoot))

	assert.Equal(t, uint64(1), service.HeadSlot(), "Head did not change")

	cachedRoot, err := service.HeadRoot(context.Background())
	require.NoError(t, err)
	if !bytes.Equal(cachedRoot, newRoot[:]) {
		t.Error("Head did not change")
	}
	assert.DeepEqual(t, newHeadSignedBlock, service.headBlock(), "Head did not change")
	assert.DeepEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
	testutil.AssertLogsContain(t, hook, "Chain reorg occurred")
}

func TestUpdateRecentCanonicalBlocks_CanUpdateWithoutParent(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	service := setupBeaconChain(t, db, sc)

	r := [32]byte{'a'}
	require.NoError(t, service.updateRecentCanonicalBlocks(context.Background(), r))
	canonical, err := service.IsCanonical(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, true, canonical, "Block should be canonical")
}

func TestUpdateRecentCanonicalBlocks_CanUpdateWithParent(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	service := setupBeaconChain(t, db, sc)
	oldHead := [32]byte{'a'}
	require.NoError(t, service.forkChoiceStore.ProcessBlock(context.Background(), 1, oldHead, [32]byte{'g'}, [32]byte{}, 0, 0))
	currentHead := [32]byte{'b'}
	require.NoError(t, service.forkChoiceStore.ProcessBlock(context.Background(), 3, currentHead, oldHead, [32]byte{}, 0, 0))
	forkedRoot := [32]byte{'c'}
	require.NoError(t, service.forkChoiceStore.ProcessBlock(context.Background(), 2, forkedRoot, oldHead, [32]byte{}, 0, 0))

	require.NoError(t, service.updateRecentCanonicalBlocks(context.Background(), currentHead))
	canonical, err := service.IsCanonical(context.Background(), currentHead)
	require.NoError(t, err)
	assert.Equal(t, true, canonical, "Block should be canonical")
	canonical, err = service.IsCanonical(context.Background(), oldHead)
	require.NoError(t, err)
	assert.Equal(t, true, canonical, "Block should be canonical")
	canonical, err = service.IsCanonical(context.Background(), forkedRoot)
	require.NoError(t, err)
	assert.Equal(t, false, canonical, "Block should not be canonical")
}

func TestCacheJustifiedStateBalances_CanCache(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	service := setupBeaconChain(t, db, sc)

	state, _ := testutil.DeterministicGenesisState(t, 100)
	r := [32]byte{'a'}
	require.NoError(t, service.beaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Root: r[:]}))
	require.NoError(t, service.beaconDB.SaveState(context.Background(), state, r))
	require.NoError(t, service.cacheJustifiedStateBalances(context.Background(), r))
	require.DeepEqual(t, service.getJustifiedBalances(), state.Balances(), "Incorrect justified balances")
}
