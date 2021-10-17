package blockchain

import (
	"context"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
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

	// Chain setup
	// Old: 0 <-- 1
	// New: 0 <-- 1 <-- 2
	baseBlk := util.NewBeaconBlock()
	baseBlk.Block.Slot = 0
	baseBlkRoot, err := baseBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(baseBlk)))

	blk1 := util.NewBeaconBlock()
	blk1.Block.Slot = 1
	blk1.Block.ParentRoot = baseBlkRoot[:]
	blk1Root, err := blk1.Block.HashTreeRoot()
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk1)))

	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = 2
	blk2.Block.ParentRoot = blk1Root[:]
	blk2Root, err := blk2.Block.HashTreeRoot()
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk2)))

	// Old head is "1"
	service.head = &head{
		slot:  1,
		root:  blk1Root,
		block: wrapper.WrappedPhase0SignedBeaconBlock(blk1),
	}

	// Update beacon-chain state to use "2"
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(2))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: 2,
		Root: blk2Root[:],
	}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, headState, blk2Root))
	require.NoError(t, service.saveHead(ctx, blk2Root))

	// Validations
	assert.Equal(t, types.Slot(2), service.HeadSlot(), "Head did not change")
	cachedRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, cachedRoot, blk2Root[:], "Head did not change")
	assert.DeepEqual(t, blk2, service.headBlock().Proto(), "Head did not change")
	assert.DeepSSZEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
}

func TestSaveHead_Different_Reorg(t *testing.T) {
	ctx := context.Background()
	hook := logTest.NewGlobal()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	// Chain setup
	// Old: 0 <-- 1
	// New: 0 <-- 2
	baseBlk := util.NewBeaconBlock()
	baseBlk.Block.Slot = 0
	baseBlkRoot, err := baseBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(baseBlk)))

	blk1 := util.NewBeaconBlock()
	blk1.Block.Slot = 1
	blk1.Block.ParentRoot = baseBlkRoot[:]
	blk1Root, err := blk1.Block.HashTreeRoot()
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk1)))

	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = 2
	blk2.Block.ParentRoot = baseBlkRoot[:]
	blk2Root, err := blk2.Block.HashTreeRoot()
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk2)))

	// Old head is "1"
	service.head = &head{
		slot:  1,
		root:  blk1Root,
		block: wrapper.WrappedPhase0SignedBeaconBlock(blk1),
	}

	// Update beacon-chain state to use "2"
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(2))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: 2,
		Root: blk2Root[:],
	}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, headState, blk2Root))
	require.NoError(t, service.saveHead(ctx, blk2Root))

	// Validations
	assert.Equal(t, types.Slot(2), service.HeadSlot(), "Head did not change")
	cachedRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, cachedRoot, blk2Root[:], "Head did not change")
	assert.DeepEqual(t, blk2, service.headBlock().Proto(), "Head did not change")
	assert.DeepSSZEqual(t, headState.CloneInnerState(), service.headState(ctx).CloneInnerState(), "Head did not change")
	require.LogsContain(t, hook, "Chain reorg occurred")
}

func TestCacheJustifiedStateBalances_CanCache(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	state, _ := util.DeterministicGenesisState(t, 100)
	r := [32]byte{'a'}
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: r[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(context.Background(), state, r))
	require.NoError(t, service.cacheJustifiedStateBalances(context.Background(), r))
	require.DeepEqual(t, service.getJustifiedBalances(), state.Balances(), "Incorrect justified balances")
}

func TestUpdateHead_MissingJustifiedRoot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	b := util.NewBeaconBlock()
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
		bState, _ := util.DeterministicGenesisState(t, 10)
		notifier := &mock.MockStateNotifier{RecordEvents: true}
		srv := &Service{
			cfg: &config{
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
		bState, _ := util.DeterministicGenesisState(t, 10)
		notifier := &mock.MockStateNotifier{RecordEvents: true}
		genesisRoot := [32]byte{1}
		srv := &Service{
			cfg: &config{
				StateNotifier: notifier,
			},
			genesisRoot: genesisRoot,
		}
		epoch1Start, err := slots.EpochStart(1)
		require.NoError(t, err)
		epoch2Start, err := slots.EpochStart(1)
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
			EpochTransition:           true,
			PreviousDutyDependentRoot: genesisRoot[:],
			CurrentDutyDependentRoot:  make([]byte, 32),
		}
		require.DeepSSZEqual(t, wanted, eventHead)
	})
}

func TestSaveOrphanedAtts(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyInsertOrphanedAtts: true,
	})
	defer resetCfg()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	service.genesisTime = time.Now()

	// Chain setup
	// Old: 0 <-- 1
	// New: 0 <-- 2
	beaconState, keys := util.DeterministicGenesisState(t, 64)
	blkGenesis, err := util.GenerateFullBlock(beaconState, keys, util.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	blkGenesis.Block.Slot = 0
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blkGenesis)))
	blkGensisRoot, err := blkGenesis.Block.HashTreeRoot()
	require.NoError(t, err)

	// We just need some valid attestation BeaconDB.SaveBlock succeed for blk1
	atts := blkGenesis.Block.Body.Attestations

	blk1 := util.NewBeaconBlock()
	blk1.Block.Slot = 1
	blk1.Block.Body.Attestations = atts
	blk1.Block.ParentRoot = blkGensisRoot[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk1)))
	blk1Root, err := blk1.Block.HashTreeRoot()
	require.NoError(t, err)

	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = 2
	blk2.Block.ParentRoot = blkGensisRoot[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk2)))
	blk2Root, err := blk2.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, service.saveOrphanedAtts(ctx, blk1Root, blk2Root))

	// Validations
	require.NotEqual(t, 0, len(blk1.Block.Body.Attestations))
	require.Equal(t, len(blk1.Block.Body.Attestations), service.cfg.AttPool.AggregatedAttestationCount())
	require.DeepSSZEqual(t, blk1.Block.Body.Attestations, service.cfg.AttPool.AggregatedAttestations())
}

func TestSaveOrphanedAtts_CanFilter(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyInsertOrphanedAtts: true,
	})
	defer resetCfg()

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	service.genesisTime = time.Now().Add(time.Duration(-1*int64(params.BeaconConfig().SlotsPerEpoch+1)*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second)

	// Chain setup
	// Old: 0 <-- 1
	// New: 0 <-- 2
	beaconState, keys := util.DeterministicGenesisState(t, 64)
	blkGenesis, err := util.GenerateFullBlock(beaconState, keys, util.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	blkGenesis.Block.Slot = 0
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blkGenesis)))
	blkGensisRoot, err := blkGenesis.Block.HashTreeRoot()
	require.NoError(t, err)

	// We just need some valid attestation BeaconDB.SaveBlock succeed for blk1
	atts := blkGenesis.Block.Body.Attestations

	blk1 := util.NewBeaconBlock()
	blk1.Block.Slot = 1
	blk1.Block.Body.Attestations = atts
	blk1.Block.ParentRoot = blkGensisRoot[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk1)))
	blk1Root, err := blk1.Block.HashTreeRoot()
	require.NoError(t, err)

	blk2 := util.NewBeaconBlock()
	blk2.Block.Slot = 2
	blk2.Block.ParentRoot = blkGensisRoot[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk2)))
	blk2Root, err := blk2.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, service.saveOrphanedAtts(ctx, blk1Root, blk2Root))

	// Validations
	require.NotEqual(t, 0, blk1.Block.Body.Attestations)
	require.Equal(t, 0, service.cfg.AttPool.AggregatedAttestationCount())
}

func TestGetOrphanedBlocks(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyInsertOrphanedAtts: true,
	})
	defer resetCfg()

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	service.genesisTime = time.Now()

	genesis, keys := util.DeterministicGenesisState(t, 64)
	blkG, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blkG)))
	blk100 := util.NewBeaconBlock()
	blk100.Block.Slot = 100
	assert.NoError(t, err)
	blk101 := util.NewBeaconBlock()
	blk101.Block.Slot = 101
	blk102 := util.NewBeaconBlock()
	blk102.Block.Slot = 102

	// Setup simple chain: 1 <-- 100 <-- 101 <-- 102
	headBlk := blkG
	for _, nextBlk := range []*ethpb.SignedBeaconBlock{blk100, blk101, blk102} {
		headBlkRoot, err := headBlk.Block.HashTreeRoot()
		nextBlk.Block.ParentRoot = headBlkRoot[:]
		assert.NoError(t, err)
		headBlk = nextBlk
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlk)))
	}

	blk100Root, err := blk100.Block.HashTreeRoot()
	assert.NoError(t, err)
	blk102Root, err := blk102.Block.HashTreeRoot()
	assert.NoError(t, err)

	orphanedBlks, err := service.getOrphanedBlocks(ctx, blk102Root, blk100Root)
	assert.NoError(t, err)
	require.Equal(t, 2, len(orphanedBlks))
}

func TestGetOrphanedBlocks_Empty(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyInsertOrphanedAtts: true,
	})
	defer resetCfg()

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	service.genesisTime = time.Now()

	genesis, keys := util.DeterministicGenesisState(t, 64)
	blkGenesis, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	blkGRoot, err := blkGenesis.Block.HashTreeRoot()
	assert.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blkGenesis)))

	// Get orphaned blocks from head to head
	orphanedBlks, err := service.getOrphanedBlocks(ctx, blkGRoot, blkGRoot)
	assert.NoError(t, err)
	require.Equal(t, 0, len(orphanedBlks))
}

func TestGetCommonBase(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyInsertOrphanedAtts: true,
	})
	defer resetCfg()

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	service.genesisTime = time.Now()

	genesis, keys := util.DeterministicGenesisState(t, 64)
	blkG, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	blkGRoot, err := blkG.Block.HashTreeRoot()
	assert.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blkG)))

	blk100 := util.NewBeaconBlock()
	blk100.Block.Slot = 100
	blk101 := util.NewBeaconBlock()
	blk101.Block.Slot = 101
	blk102 := util.NewBeaconBlock()
	blk102.Block.Slot = 102
	blk103 := util.NewBeaconBlock()
	blk103.Block.Slot = 103
	blk104 := util.NewBeaconBlock()
	blk104.Block.Slot = 104
	blk105 := util.NewBeaconBlock()
	blk105.Block.Slot = 105

	// Setup simple chain: 1 <-- 100 <-- 101 <-- 103
	headBlk := blkG
	for _, nextBlk := range []*ethpb.SignedBeaconBlock{blk100, blk101, blk103} {
		headBlkRoot, err := headBlk.Block.HashTreeRoot()
		nextBlk.Block.ParentRoot = headBlkRoot[:]
		assert.NoError(t, err)
		headBlk = nextBlk
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlk)))
	}
	// Setup simple chain: 1 <-- 102 <-- 104 <-- 105
	headBlk = blkG
	for _, nextBlk := range []*ethpb.SignedBeaconBlock{blk102, blk104, blk105} {
		headBlkRoot, err := headBlk.Block.HashTreeRoot()
		nextBlk.Block.ParentRoot = headBlkRoot[:]
		assert.NoError(t, err)
		headBlk = nextBlk
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlk)))
	}

	blk103Root, err := blk103.Block.HashTreeRoot()
	assert.NoError(t, err)
	blk105Root, err := blk105.Block.HashTreeRoot()
	assert.NoError(t, err)

	commonRoot, err := service.commonAncestorRoot(ctx, blk105Root, blk103Root)
	require.Equal(t, commonRoot, blkGRoot)
}
