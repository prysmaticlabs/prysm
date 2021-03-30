package blockchain

import (
	"bytes"
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveHead_Same(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	r := [32]byte{'A'}
	service.head = &head{slot: 0, root: r}

	require.NoError(t, service.saveHead(context.Background(), r))
	assert.Equal(t, types.Slot(0), service.headSlot(), "Head did not stay the same")
	assert.Equal(t, r, service.headRoot(), "Head did not stay the same")
}

func TestSaveHead_Different(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	newHeadSignedBlock := testutil.NewBeaconBlock()
	newHeadSignedBlock.Block.Slot = 1
	newHeadBlock := newHeadSignedBlock.Block

	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), newHeadSignedBlock))
	newRoot, err := newHeadBlock.HashTreeRoot()
	require.NoError(t, err)
	headState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(1))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Slot: 1, Root: newRoot[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(context.Background(), headState, newRoot))
	require.NoError(t, service.saveHead(context.Background(), newRoot))

	assert.Equal(t, types.Slot(1), service.HeadSlot(), "Head did not change")

	cachedRoot, err := service.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, cachedRoot, newRoot[:], "Head did not change")
	assert.DeepEqual(t, newHeadSignedBlock, service.headBlock(), "Head did not change")
	assert.DeepSSZEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
}

func TestSaveHead_Different_Reorg(t *testing.T) {
	ctx := context.Background()
	hook := logTest.NewGlobal()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	oldRoot := [32]byte{'A'}
	service.head = &head{slot: 0, root: oldRoot}

	reorgChainParent := [32]byte{'B'}
	newHeadSignedBlock := testutil.NewBeaconBlock()
	newHeadSignedBlock.Block.Slot = 1
	newHeadSignedBlock.Block.ParentRoot = reorgChainParent[:]
	newHeadBlock := newHeadSignedBlock.Block

	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), newHeadSignedBlock))
	newRoot, err := newHeadBlock.HashTreeRoot()
	require.NoError(t, err)
	headState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(1))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Slot: 1, Root: newRoot[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(context.Background(), headState, newRoot))
	require.NoError(t, service.saveHead(context.Background(), newRoot))

	assert.Equal(t, types.Slot(1), service.HeadSlot(), "Head did not change")

	cachedRoot, err := service.HeadRoot(context.Background())
	require.NoError(t, err)
	if !bytes.Equal(cachedRoot, newRoot[:]) {
		t.Error("Head did not change")
	}
	assert.DeepEqual(t, newHeadSignedBlock, service.headBlock(), "Head did not change")
	assert.DeepSSZEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
	require.LogsContain(t, hook, "Chain reorg occurred")
}

func TestCacheJustifiedStateBalances_CanCache(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	state, _ := testutil.DeterministicGenesisState(t, 100)
	r := [32]byte{'a'}
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(context.Background(), &pb.StateSummary{Root: r[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(context.Background(), state, r))
	require.NoError(t, service.cacheJustifiedStateBalances(context.Background(), r))
	require.DeepEqual(t, service.getJustifiedBalances(), state.Balances(), "Incorrect justified balances")
}

func TestUpdateHead_MissingJustifiedRoot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	b := testutil.NewBeaconBlock()
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), b))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{}

	require.NoError(t, service.updateHead(context.Background(), []uint64{}))
}
