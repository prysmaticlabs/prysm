package blockchain

import (
	"bytes"
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
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
	service.head = &head{
		slot: 0,
		root: oldRoot,
		block: wrapper.WrappedPhase0SignedBeaconBlock(
			&ethpb.SignedBeaconBlock{
				Block: &ethpb.BeaconBlock{
					Slot:      0,
					StateRoot: make([]byte, 32),
				},
			},
		),
	}

	newHeadSignedBlock := testutil.NewBeaconBlock()
	newHeadSignedBlock.Block.Slot = 1
	newHeadBlock := newHeadSignedBlock.Block

	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(newHeadSignedBlock)))
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
	assert.DeepEqual(t, newHeadSignedBlock, service.headBlock().Proto(), "Head did not change")
	assert.DeepSSZEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
}

func TestSaveHead_Different_Reorg(t *testing.T) {
	ctx := context.Background()
	hook := logTest.NewGlobal()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	oldRoot := [32]byte{'A'}
	service.head = &head{
		slot: 0,
		root: oldRoot,
		block: wrapper.WrappedPhase0SignedBeaconBlock(
			&ethpb.SignedBeaconBlock{
				Block: &ethpb.BeaconBlock{
					Slot:      0,
					StateRoot: make([]byte, 32),
				},
			},
		),
	}

	reorgChainParent := [32]byte{'B'}
	newHeadSignedBlock := testutil.NewBeaconBlock()
	newHeadSignedBlock.Block.Slot = 1
	newHeadSignedBlock.Block.ParentRoot = reorgChainParent[:]
	newHeadBlock := newHeadSignedBlock.Block

	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(newHeadSignedBlock)))
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
	assert.DeepEqual(t, newHeadSignedBlock, service.headBlock().Proto(), "Head did not change")
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
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(b)))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{}

	require.NoError(t, service.updateHead(context.Background(), []uint64{}))
}

func Test_notifyNewHeadEvent(t *testing.T) {
	t.Run("genesis_state_root", func(t *testing.T) {
		bState, _ := testutil.DeterministicGenesisState(t, 10)
		notifier := &mock.MockStateNotifier{RecordEvents: true}
		srv := &Service{
			cfg: &Config{
				StateNotifier: notifier,
			},
			genesisRoot: [32]byte{1},
		}
		newHeadStateRoot := [32]byte{2}
		newHeadRoot := [32]byte{3}
		err := srv.notifyNewHeadEvent(1, bState, newHeadStateRoot[:], newHeadRoot[:])
		require.NoError(t, err)
		events := notifier.ReceivedEvents()
		require.Equal(t, 1, len(events))

		eventHead, ok := events[0].Data.(*ethpbv1.EventHead)
		require.Equal(t, true, ok)
		wanted := &ethpbv1.EventHead{
			Slot:                      1,
			Block:                     newHeadRoot[:],
			State:                     newHeadStateRoot[:],
			EpochTransition:           false,
			PreviousDutyDependentRoot: srv.genesisRoot[:],
			CurrentDutyDependentRoot:  srv.genesisRoot[:],
		}
		require.DeepSSZEqual(t, wanted, eventHead)
	})
	t.Run("non_genesis_values", func(t *testing.T) {
		bState, _ := testutil.DeterministicGenesisState(t, 10)
		notifier := &mock.MockStateNotifier{RecordEvents: true}
		genesisRoot := [32]byte{1}
		srv := &Service{
			cfg: &Config{
				StateNotifier: notifier,
			},
			genesisRoot: genesisRoot,
		}
		epoch1Start, err := helpers.StartSlot(1)
		require.NoError(t, err)
		epoch2Start, err := helpers.StartSlot(1)
		require.NoError(t, err)
		require.NoError(t, bState.SetSlot(epoch1Start))

		newHeadStateRoot := [32]byte{2}
		newHeadRoot := [32]byte{3}
		err = srv.notifyNewHeadEvent(epoch2Start, bState, newHeadStateRoot[:], newHeadRoot[:])
		require.NoError(t, err)
		events := notifier.ReceivedEvents()
		require.Equal(t, 1, len(events))

		eventHead, ok := events[0].Data.(*ethpbv1.EventHead)
		require.Equal(t, true, ok)
		wanted := &ethpbv1.EventHead{
			Slot:                      epoch2Start,
			Block:                     newHeadRoot[:],
			State:                     newHeadStateRoot[:],
			EpochTransition:           false,
			PreviousDutyDependentRoot: genesisRoot[:],
			CurrentDutyDependentRoot:  make([]byte, 32),
		}
		require.DeepSSZEqual(t, wanted, eventHead)
	})
}
