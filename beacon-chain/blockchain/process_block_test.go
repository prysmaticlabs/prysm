package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/time"
)

func TestStore_OnBlock_ProtoArray(t *testing.T) {
	ctx := context.Background()

	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)
	random := util.NewBeaconBlock()
	random.Block.Slot = 1
	random.Block.ParentRoot = validGenesisRoot[:]
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(random)))
	randomParentRoot, err := random.Block.HashTreeRoot()
	assert.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: st.Slot(), Root: randomParentRoot[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), randomParentRoot))
	randomParentRoot2 := roots[1]
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: st.Slot(), Root: randomParentRoot2}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), bytesutil.ToBytes32(randomParentRoot2)))

	tests := []struct {
		name          string
		blk           *ethpb.SignedBeaconBlock
		s             state.BeaconState
		time          uint64
		wantErrString string
	}{
		{
			name:          "parent block root does not have a state",
			blk:           util.NewBeaconBlock(),
			s:             st.Copy(),
			wantErrString: "could not reconstruct parent state",
		},
		{
			name: "block is from the future",
			blk: func() *ethpb.SignedBeaconBlock {
				b := util.NewBeaconBlock()
				b.Block.ParentRoot = randomParentRoot2
				b.Block.Slot = params.BeaconConfig().FarFutureSlot
				return b
			}(),
			s:             st.Copy(),
			wantErrString: "is in the far distant future",
		},
		{
			name: "could not get finalized block",
			blk: func() *ethpb.SignedBeaconBlock {
				b := util.NewBeaconBlock()
				b.Block.ParentRoot = randomParentRoot[:]
				return b
			}(),
			s:             st.Copy(),
			wantErrString: "is not a descendant of the current finalized block",
		},
		{
			name: "same slot as finalized block",
			blk: func() *ethpb.SignedBeaconBlock {
				b := util.NewBeaconBlock()
				b.Block.Slot = 0
				b.Block.ParentRoot = randomParentRoot2
				return b
			}(),
			s:             st.Copy(),
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: validGenesisRoot[:]})
			service.store.SetBestJustifiedCheckpt(&ethpb.Checkpoint{Root: validGenesisRoot[:]})
			service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: roots[0]})
			service.store.SetPrevFinalizedCheckpt(&ethpb.Checkpoint{Root: validGenesisRoot[:]})

			root, err := tt.blk.Block.HashTreeRoot()
			assert.NoError(t, err)
			err = service.onBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(tt.blk), root)
			assert.ErrorContains(t, tt.wantErrString, err)
		})
	}
}

func TestStore_OnBlock_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()

	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New(0, 0)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)
	random := util.NewBeaconBlock()
	random.Block.Slot = 1
	random.Block.ParentRoot = validGenesisRoot[:]
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(random)))
	randomParentRoot, err := random.Block.HashTreeRoot()
	assert.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: st.Slot(), Root: randomParentRoot[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), randomParentRoot))
	randomParentRoot2 := roots[1]
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: st.Slot(), Root: randomParentRoot2}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), bytesutil.ToBytes32(randomParentRoot2)))

	tests := []struct {
		name          string
		blk           *ethpb.SignedBeaconBlock
		s             state.BeaconState
		time          uint64
		wantErrString string
	}{
		{
			name:          "parent block root does not have a state",
			blk:           util.NewBeaconBlock(),
			s:             st.Copy(),
			wantErrString: "could not reconstruct parent state",
		},
		{
			name: "block is from the future",
			blk: func() *ethpb.SignedBeaconBlock {
				b := util.NewBeaconBlock()
				b.Block.ParentRoot = randomParentRoot2
				b.Block.Slot = params.BeaconConfig().FarFutureSlot
				return b
			}(),
			s:             st.Copy(),
			wantErrString: "is in the far distant future",
		},
		{
			name: "could not get finalized block",
			blk: func() *ethpb.SignedBeaconBlock {
				b := util.NewBeaconBlock()
				b.Block.ParentRoot = randomParentRoot[:]
				return b
			}(),
			s:             st.Copy(),
			wantErrString: "is not a descendant of the current finalized block",
		},
		{
			name: "same slot as finalized block",
			blk: func() *ethpb.SignedBeaconBlock {
				b := util.NewBeaconBlock()
				b.Block.Slot = 0
				b.Block.ParentRoot = randomParentRoot2
				return b
			}(),
			s:             st.Copy(),
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: validGenesisRoot[:]})
			service.store.SetBestJustifiedCheckpt(&ethpb.Checkpoint{Root: validGenesisRoot[:]})
			service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: roots[0]})
			service.store.SetPrevFinalizedCheckpt(&ethpb.Checkpoint{Root: validGenesisRoot[:]})

			root, err := tt.blk.Block.HashTreeRoot()
			assert.NoError(t, err)
			err = service.onBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(tt.blk), root)
			assert.ErrorContains(t, tt.wantErrString, err)
		})
	}
}

func TestStore_OnBlock_ProposerBoostEarly(t *testing.T) {
	ctx := context.Background()

	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New(0, 0)
	opts := []Option{
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	require.NoError(t, service.cfg.ForkChoiceStore.BoostProposerRoot(ctx, 0, [32]byte{'A'}, time.Now()))
	_, err = service.cfg.ForkChoiceStore.Head(ctx, 0,
		params.BeaconConfig().ZeroHash, []uint64{}, 0)
	require.ErrorContains(t, "could not apply proposer boost score: invalid proposer boost root", err)
}

func TestStore_OnBlockBatch_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})

	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{})
	service.saveInitSyncBlock(gRoot, wrapper.WrappedPhase0SignedBeaconBlock(genesis))

	st, keys := util.DeterministicGenesisState(t, 64)

	bState := st.Copy()

	var blks []block.SignedBeaconBlock
	var blkRoots [][32]byte
	var firstState state.BeaconState
	for i := 1; i < 10; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wrapper.WrappedPhase0SignedBeaconBlock(b))
		require.NoError(t, err)
		if i == 1 {
			firstState = bState.Copy()
		}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		service.saveInitSyncBlock(root, wrapper.WrappedPhase0SignedBeaconBlock(b))
		blks = append(blks, wrapper.WrappedPhase0SignedBeaconBlock(b))
		blkRoots = append(blkRoots, root)
	}

	rBlock, err := blks[0].PbPhase0Block()
	assert.NoError(t, err)
	rBlock.Block.ParentRoot = gRoot[:]
	require.NoError(t, beaconDB.SaveBlock(context.Background(), blks[0]))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, blkRoots[0], firstState))
	_, _, err = service.onBlockBatch(ctx, blks, blkRoots[1:])
	require.ErrorIs(t, errWrongBlockCount, err)
	_, _, err = service.onBlockBatch(ctx, blks[1:], blkRoots[1:])
	require.NoError(t, err)
}

func TestStore_OnBlockBatch_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})

	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	service.saveInitSyncBlock(gRoot, wrapper.WrappedPhase0SignedBeaconBlock(genesis))

	st, keys := util.DeterministicGenesisState(t, 64)

	bState := st.Copy()

	var blks []block.SignedBeaconBlock
	var blkRoots [][32]byte
	var firstState state.BeaconState
	for i := 1; i < 10; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wrapper.WrappedPhase0SignedBeaconBlock(b))
		require.NoError(t, err)
		if i == 1 {
			firstState = bState.Copy()
		}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		service.saveInitSyncBlock(root, wrapper.WrappedPhase0SignedBeaconBlock(b))
		blks = append(blks, wrapper.WrappedPhase0SignedBeaconBlock(b))
		blkRoots = append(blkRoots, root)
	}

	rBlock, err := blks[0].PbPhase0Block()
	assert.NoError(t, err)
	rBlock.Block.ParentRoot = gRoot[:]
	require.NoError(t, beaconDB.SaveBlock(context.Background(), blks[0]))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, blkRoots[0], firstState))
	_, _, err = service.onBlockBatch(ctx, blks, blkRoots[1:])
	require.ErrorIs(t, errWrongBlockCount, err)
	_, _, err = service.onBlockBatch(ctx, blks[1:], blkRoots[1:])
	require.NoError(t, err)
}

func TestStore_OnBlockBatch_NotifyNewPayload(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesis := blocks.NewGenesisBlock(params.BeaconConfig().ZeroHash[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})
	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	service.saveInitSyncBlock(gRoot, wrapper.WrappedPhase0SignedBeaconBlock(genesis))
	st, keys := util.DeterministicGenesisState(t, 64)
	bState := st.Copy()

	var blks []block.SignedBeaconBlock
	var blkRoots [][32]byte
	var firstState state.BeaconState
	blkCount := 4
	for i := 1; i <= blkCount; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wrapper.WrappedPhase0SignedBeaconBlock(b))
		require.NoError(t, err)
		if i == 1 {
			firstState = bState.Copy()
		}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		service.saveInitSyncBlock(root, wrapper.WrappedPhase0SignedBeaconBlock(b))
		blks = append(blks, wrapper.WrappedPhase0SignedBeaconBlock(b))
		blkRoots = append(blkRoots, root)
	}

	rBlock, err := blks[0].PbPhase0Block()
	assert.NoError(t, err)
	rBlock.Block.ParentRoot = gRoot[:]
	require.NoError(t, beaconDB.SaveBlock(context.Background(), blks[0]))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, blkRoots[0], firstState))
	cp1, cp2, err := service.onBlockBatch(ctx, blks[1:], blkRoots[1:])
	require.NoError(t, err)
	require.Equal(t, blkCount-1, len(cp1))
	require.Equal(t, blkCount-1, len(cp2))
}

func TestRemoveStateSinceLastFinalized_EmptyStartSlot(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.genesisTime = time.Now()

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: make([]byte, 32)})
	require.NoError(t, err)
	assert.Equal(t, true, update, "Should be able to update justified")
	lastJustifiedBlk := util.NewBeaconBlock()
	lastJustifiedBlk.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	lastJustifiedRoot, err := lastJustifiedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	newJustifiedBlk := util.NewBeaconBlock()
	newJustifiedBlk.Block.Slot = 1
	newJustifiedBlk.Block.ParentRoot = bytesutil.PadTo(lastJustifiedRoot[:], 32)
	newJustifiedRoot, err := newJustifiedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(newJustifiedBlk)))
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(lastJustifiedBlk)))

	diff := params.BeaconConfig().SlotsPerEpoch.Sub(1).Mul(params.BeaconConfig().SecondsPerSlot)
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: lastJustifiedRoot[:]})
	update, err = service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	require.NoError(t, err)
	assert.Equal(t, true, update, "Should be able to update justified")
}

func TestShouldUpdateJustified_ReturnFalse_ProtoArray(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{})
	lastJustifiedBlk := util.NewBeaconBlock()
	lastJustifiedBlk.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	lastJustifiedRoot, err := lastJustifiedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	newJustifiedBlk := util.NewBeaconBlock()
	newJustifiedBlk.Block.ParentRoot = bytesutil.PadTo(lastJustifiedRoot[:], 32)
	newJustifiedRoot, err := newJustifiedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(newJustifiedBlk)))
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(lastJustifiedBlk)))

	diff := params.BeaconConfig().SlotsPerEpoch.Sub(1).Mul(params.BeaconConfig().SecondsPerSlot)
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: lastJustifiedRoot[:]})

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	require.NoError(t, err)
	assert.Equal(t, false, update, "Should not be able to update justified, received true")
}

func TestShouldUpdateJustified_ReturnFalse_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	lastJustifiedBlk := util.NewBeaconBlock()
	lastJustifiedBlk.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	lastJustifiedRoot, err := lastJustifiedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	newJustifiedBlk := util.NewBeaconBlock()
	newJustifiedBlk.Block.ParentRoot = bytesutil.PadTo(lastJustifiedRoot[:], 32)
	newJustifiedRoot, err := newJustifiedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(newJustifiedBlk)))
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(lastJustifiedBlk)))

	diff := params.BeaconConfig().SlotsPerEpoch.Sub(1).Mul(params.BeaconConfig().SecondsPerSlot)
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: lastJustifiedRoot[:]})

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	require.NoError(t, err)
	assert.Equal(t, false, update, "Should not be able to update justified, received true")
}

func TestCachedPreState_CanGetFromStateSummary_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	s, err := v1.InitializeFromProto(&ethpb.BeaconState{Slot: 1, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
	require.NoError(t, err)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})
	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{})
	service.saveInitSyncBlock(gRoot, wrapper.WrappedPhase0SignedBeaconBlock(genesis))

	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	b.Block.ParentRoot = gRoot[:]
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: gRoot[:]}))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, gRoot, s))
	require.NoError(t, service.verifyBlkPreState(ctx, wrapper.WrappedPhase0BeaconBlock(b.Block)))
}

func TestCachedPreState_CanGetFromStateSummary_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	s, err := v1.InitializeFromProto(&ethpb.BeaconState{Slot: 1, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
	require.NoError(t, err)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})
	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	service.saveInitSyncBlock(gRoot, wrapper.WrappedPhase0SignedBeaconBlock(genesis))

	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	b.Block.ParentRoot = gRoot[:]
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: gRoot[:]}))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, gRoot, s))
	require.NoError(t, service.verifyBlkPreState(ctx, wrapper.WrappedPhase0BeaconBlock(b.Block)))
}

func TestCachedPreState_CanGetFromDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})
	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{})
	service.saveInitSyncBlock(gRoot, wrapper.WrappedPhase0SignedBeaconBlock(genesis))

	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})
	err = service.verifyBlkPreState(ctx, wrapper.WrappedPhase0BeaconBlock(b.Block))
	wanted := "could not reconstruct parent state"
	assert.ErrorContains(t, wanted, err)

	b.Block.ParentRoot = gRoot[:]
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{Slot: 1})
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: gRoot[:]}))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, gRoot, s))
	require.NoError(t, service.verifyBlkPreState(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b).Block()))
}

func TestUpdateJustified_CouldUpdateBest(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New(0, 0, [32]byte{})),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	signedBlock := util.NewBeaconBlock()
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(signedBlock)))
	r, err := signedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: []byte{'A'}})
	service.store.SetBestJustifiedCheckpt(&ethpb.Checkpoint{Root: []byte{'A'}})
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, st.Copy(), r))

	// Could update
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: r[:]}))
	require.NoError(t, service.updateJustified(context.Background(), s))

	assert.Equal(t, s.CurrentJustifiedCheckpoint().Epoch, service.store.BestJustifiedCheckpt().Epoch, "Incorrect justified epoch in service")

	// Could not update
	service.store.SetBestJustifiedCheckpt(&ethpb.Checkpoint{Root: []byte{'A'}, Epoch: 2})
	require.NoError(t, service.updateJustified(context.Background(), s))

	assert.Equal(t, types.Epoch(2), service.store.BestJustifiedCheckpt().Epoch, "Incorrect justified epoch in service")
}

func TestFillForkChoiceMissingBlocks_CanSave_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: make([]byte, 32)})

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	block := util.NewBeaconBlock()
	block.Block.Slot = 9
	block.Block.ParentRoot = roots[8]

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(block).Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	assert.Equal(t, 5, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[4])), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[6])), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[8])), "Didn't save node")
}

func TestFillForkChoiceMissingBlocks_CanSave_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: make([]byte, 32)})

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	block := util.NewBeaconBlock()
	block.Block.Slot = 9
	block.Block.ParentRoot = roots[8]

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(block).Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	assert.Equal(t, 5, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[4])), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[6])), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[8])), "Didn't save node")
}

func TestFillForkChoiceMissingBlocks_RootsMatch_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: make([]byte, 32)})

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	block := util.NewBeaconBlock()
	block.Block.Slot = 9
	block.Block.ParentRoot = roots[8]

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(block).Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	assert.Equal(t, 5, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	// Ensure all roots and their respective blocks exist.
	wantedRoots := [][]byte{roots[0], roots[3], roots[4], roots[6], roots[8]}
	for i, rt := range wantedRoots {
		assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(rt)), fmt.Sprintf("Didn't save node: %d", i))
		assert.Equal(t, true, service.cfg.BeaconDB.HasBlock(context.Background(), bytesutil.ToBytes32(rt)))
	}
}

func TestFillForkChoiceMissingBlocks_RootsMatch_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: make([]byte, 32)})

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	block := util.NewBeaconBlock()
	block.Block.Slot = 9
	block.Block.ParentRoot = roots[8]

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(block).Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	assert.Equal(t, 5, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	// Ensure all roots and their respective blocks exist.
	wantedRoots := [][]byte{roots[0], roots[3], roots[4], roots[6], roots[8]}
	for i, rt := range wantedRoots {
		assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(rt)), fmt.Sprintf("Didn't save node: %d", i))
		assert.Equal(t, true, service.cfg.BeaconDB.HasBlock(context.Background(), bytesutil.ToBytes32(rt)))
	}
}

func TestFillForkChoiceMissingBlocks_FilterFinalized_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	// Set finalized epoch to 1.
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: 1})

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	assert.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))

	// Define a tree branch, slot 63 <- 64 <- 65
	b63 := util.NewBeaconBlock()
	b63.Block.Slot = 63
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b63)))
	r63, err := b63.Block.HashTreeRoot()
	require.NoError(t, err)
	b64 := util.NewBeaconBlock()
	b64.Block.Slot = 64
	b64.Block.ParentRoot = r63[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b64)))
	r64, err := b64.Block.HashTreeRoot()
	require.NoError(t, err)
	b65 := util.NewBeaconBlock()
	b65.Block.Slot = 65
	b65.Block.ParentRoot = r64[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b65)))

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(b65).Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// There should be 2 nodes, block 65 and block 64.
	assert.Equal(t, 2, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")

	// Block with slot 63 should be in fork choice because it's less than finalized epoch 1.
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(r63), "Didn't save node")
}

func TestFillForkChoiceMissingBlocks_FilterFinalized_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = doublylinkedtree.New(0, 0)
	// Set finalized epoch to 1.
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: 1})

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	assert.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))

	// Define a tree branch, slot 63 <- 64 <- 65
	b63 := util.NewBeaconBlock()
	b63.Block.Slot = 63
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b63)))
	r63, err := b63.Block.HashTreeRoot()
	require.NoError(t, err)
	b64 := util.NewBeaconBlock()
	b64.Block.Slot = 64
	b64.Block.ParentRoot = r63[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b64)))
	r64, err := b64.Block.HashTreeRoot()
	require.NoError(t, err)
	b65 := util.NewBeaconBlock()
	b65.Block.Slot = 65
	b65.Block.ParentRoot = r64[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b65)))

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(b65).Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// There should be 2 nodes, block 65 and block 64.
	assert.Equal(t, 2, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")

	// Block with slot 63 should be in fork choice because it's less than finalized epoch 1.
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(r63), "Didn't save node")
}

// blockTree1 constructs the following tree:
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree1(t *testing.T, beaconDB db.Database, genesisRoot []byte) ([][]byte, error) {
	genesisRoot = bytesutil.PadTo(genesisRoot, 32)
	b0 := util.NewBeaconBlock()
	b0.Block.Slot = 0
	b0.Block.ParentRoot = genesisRoot
	r0, err := b0.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = r0[:]
	r1, err := b1.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 3
	b3.Block.ParentRoot = r0[:]
	r3, err := b3.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b4 := util.NewBeaconBlock()
	b4.Block.Slot = 4
	b4.Block.ParentRoot = r3[:]
	r4, err := b4.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b5 := util.NewBeaconBlock()
	b5.Block.Slot = 5
	b5.Block.ParentRoot = r4[:]
	r5, err := b5.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b6 := util.NewBeaconBlock()
	b6.Block.Slot = 6
	b6.Block.ParentRoot = r4[:]
	r6, err := b6.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b7 := util.NewBeaconBlock()
	b7.Block.Slot = 7
	b7.Block.ParentRoot = r5[:]
	r7, err := b7.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b8 := util.NewBeaconBlock()
	b8.Block.Slot = 8
	b8.Block.ParentRoot = r6[:]
	r8, err := b8.Block.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	for _, b := range []*ethpb.SignedBeaconBlock{b0, b1, b3, b4, b5, b6, b7, b8} {
		beaconBlock := util.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		if err := beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(beaconBlock)); err != nil {
			return nil, err
		}
		if err := beaconDB.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, errors.Wrap(err, "could not save state")
		}
	}
	if err := beaconDB.SaveState(context.Background(), st.Copy(), r1); err != nil {
		return nil, err
	}
	if err := beaconDB.SaveState(context.Background(), st.Copy(), r7); err != nil {
		return nil, err
	}
	if err := beaconDB.SaveState(context.Background(), st.Copy(), r8); err != nil {
		return nil, err
	}
	return [][]byte{r0[:], r1[:], nil, r3[:], r4[:], r5[:], r6[:], r7[:], r8[:]}, nil
}

func TestCurrentSlot_HandlesOverflow(t *testing.T) {
	svc := Service{genesisTime: prysmTime.Now().Add(1 * time.Hour)}

	slot := svc.CurrentSlot()
	require.Equal(t, types.Slot(0), slot, "Unexpected slot")
}
func TestAncestorByDB_CtxErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	cancel()
	_, err = service.ancestorByDB(ctx, [32]byte{}, 0)
	require.ErrorContains(t, "context canceled", err)
}

func TestAncestor_HandleSkipSlot(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b100 := util.NewBeaconBlock()
	b100.Block.Slot = 100
	b100.Block.ParentRoot = r1[:]
	r100, err := b100.Block.HashTreeRoot()
	require.NoError(t, err)
	b200 := util.NewBeaconBlock()
	b200.Block.Slot = 200
	b200.Block.ParentRoot = r100[:]
	r200, err := b200.Block.HashTreeRoot()
	require.NoError(t, err)
	for _, b := range []*ethpb.SignedBeaconBlock{b1, b100, b200} {
		beaconBlock := util.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(beaconBlock)))
	}

	// Slots 100 to 200 are skip slots. Requesting root at 150 will yield root at 100. The last physical block.
	r, err := service.ancestor(context.Background(), r200[:], 150)
	require.NoError(t, err)
	if bytesutil.ToBytes32(r) != r100 {
		t.Error("Did not get correct root")
	}

	// Slots 1 to 100 are skip slots. Requesting root at 50 will yield root at 1. The last physical block.
	r, err = service.ancestor(context.Background(), r200[:], 50)
	require.NoError(t, err)
	if bytesutil.ToBytes32(r) != r1 {
		t.Error("Did not get correct root")
	}
}

func TestAncestor_CanUseForkchoice(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b100 := util.NewBeaconBlock()
	b100.Block.Slot = 100
	b100.Block.ParentRoot = r1[:]
	r100, err := b100.Block.HashTreeRoot()
	require.NoError(t, err)
	b200 := util.NewBeaconBlock()
	b200.Block.Slot = 200
	b200.Block.ParentRoot = r100[:]
	r200, err := b200.Block.HashTreeRoot()
	require.NoError(t, err)
	for _, b := range []*ethpb.SignedBeaconBlock{b1, b100, b200} {
		beaconBlock := util.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(context.Background(), b.Block.Slot, r, bytesutil.ToBytes32(b.Block.ParentRoot), params.BeaconConfig().ZeroHash, 0, 0)) // Saves blocks to fork choice store.
	}

	r, err := service.ancestor(context.Background(), r200[:], 150)
	require.NoError(t, err)
	if bytesutil.ToBytes32(r) != r100 {
		t.Error("Did not get correct root")
	}
}

func TestAncestor_CanUseDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b100 := util.NewBeaconBlock()
	b100.Block.Slot = 100
	b100.Block.ParentRoot = r1[:]
	r100, err := b100.Block.HashTreeRoot()
	require.NoError(t, err)
	b200 := util.NewBeaconBlock()
	b200.Block.Slot = 200
	b200.Block.ParentRoot = r100[:]
	r200, err := b200.Block.HashTreeRoot()
	require.NoError(t, err)
	for _, b := range []*ethpb.SignedBeaconBlock{b1, b100, b200} {
		beaconBlock := util.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(beaconBlock))) // Saves blocks to DB.
	}

	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(context.Background(), 200, r200, r200, params.BeaconConfig().ZeroHash, 0, 0))

	r, err := service.ancestor(context.Background(), r200[:], 150)
	require.NoError(t, err)
	if bytesutil.ToBytes32(r) != r100 {
		t.Error("Did not get correct root")
	}
}

func TestEnsureRootNotZeroHashes(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsNoDB()
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.originBlockRoot = [32]byte{'a'}

	r := service.ensureRootNotZeros(params.BeaconConfig().ZeroHash)
	assert.Equal(t, service.originBlockRoot, r, "Did not get wanted justified root")
	root := [32]byte{'b'}
	r = service.ensureRootNotZeros(root)
	assert.Equal(t, root, r, "Did not get wanted justified root")
}

func TestVerifyBlkDescendant(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()

	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.Body.Graffiti = bytesutil.PadTo([]byte{'a'}, 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))

	type args struct {
		parentRoot    [32]byte
		finalizedRoot [32]byte
	}
	tests := []struct {
		name      string
		args      args
		wantedErr string
	}{
		{
			name: "could not get finalized block in block service cache",
			args: args{
				finalizedRoot: [32]byte{'a'},
			},
			wantedErr: "nil finalized block",
		},
		{
			name: "could not get finalized block root in DB",
			args: args{
				finalizedRoot: r,
				parentRoot:    [32]byte{'a'},
			},
			wantedErr: "could not get finalized block root",
		},
		{
			name: "is not descendant",
			args: args{
				finalizedRoot: r1,
				parentRoot:    r,
			},
			wantedErr: "is not a descendant of the current finalized block slot",
		},
		{
			name: "is descendant",
			args: args{
				finalizedRoot: r,
				parentRoot:    r,
			},
		},
	}
	for _, tt := range tests {
		service, err := NewService(ctx, opts...)
		require.NoError(t, err)
		service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: tt.args.finalizedRoot[:]})
		err = service.VerifyBlkDescendant(ctx, tt.args.parentRoot)
		if tt.wantedErr != "" {
			assert.ErrorContains(t, tt.wantedErr, err)
		} else if err != nil {
			assert.NoError(t, err)
		}
	}
}

func TestUpdateJustifiedInitSync(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gBlk := util.NewBeaconBlock()
	gRoot, err := gBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(gBlk)))
	require.NoError(t, service.cfg.BeaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: gRoot[:]}))
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, beaconState, gRoot))
	service.originBlockRoot = gRoot
	currentCp := &ethpb.Checkpoint{Epoch: 1}
	service.store.SetJustifiedCheckpt(currentCp)
	newCp := &ethpb.Checkpoint{Epoch: 2, Root: gRoot[:]}

	require.NoError(t, service.updateJustifiedInitSync(ctx, newCp))

	assert.DeepSSZEqual(t, currentCp, service.PreviousJustifiedCheckpt(), "Incorrect previous justified checkpoint")
	assert.DeepSSZEqual(t, newCp, service.CurrentJustifiedCheckpt(), "Incorrect current justified checkpoint in cache")
	cp, err := service.cfg.BeaconDB.JustifiedCheckpoint(ctx)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, newCp, cp, "Incorrect current justified checkpoint in db")
}

func TestHandleEpochBoundary_BadMetrics(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsNoDB()
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))
	service.head = &head{state: (*v1.BeaconState)(nil)}

	require.ErrorContains(t, "failed to initialize precompute: nil inner state", service.handleEpochBoundary(ctx, s))
}

func TestHandleEpochBoundary_UpdateFirstSlot(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsNoDB()
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	s, _ := util.DeterministicGenesisState(t, 1024)
	service.head = &head{state: s}
	require.NoError(t, s.SetSlot(2*params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.handleEpochBoundary(ctx, s))
	require.Equal(t, 3*params.BeaconConfig().SlotsPerEpoch, service.nextEpochBoundarySlot)
}

func TestOnBlock_CanFinalize(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0, [32]byte{'a'})
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
		WithStateNotifier(&mock.MockStateNotifier{}),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, keys := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	gBlk, err := service.cfg.BeaconDB.GenesisBlock(ctx)
	require.NoError(t, err)
	gRoot, err := gBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})

	testState := gs.Copy()
	for i := types.Slot(1); i <= 4*params.BeaconConfig().SlotsPerEpoch; i++ {
		blk, err := util.GenerateFullBlock(testState, keys, util.DefaultBlockGenConfig(), i)
		require.NoError(t, err)
		r, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk), r))
		testState, err = service.cfg.StateGen.StateByRoot(ctx, r)
		require.NoError(t, err)
	}
	require.Equal(t, types.Epoch(3), service.CurrentJustifiedCheckpt().Epoch)
	require.Equal(t, types.Epoch(2), service.FinalizedCheckpt().Epoch)

	// The update should persist in DB.
	j, err := service.cfg.BeaconDB.JustifiedCheckpoint(ctx)
	require.NoError(t, err)
	require.Equal(t, j.Epoch, service.CurrentJustifiedCheckpt().Epoch)
	f, err := service.cfg.BeaconDB.FinalizedCheckpoint(ctx)
	require.NoError(t, err)
	require.Equal(t, f.Epoch, service.FinalizedCheckpt().Epoch)
}

func TestOnBlock_CallNewPayloadAndForkchoiceUpdated(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0, [32]byte{'a'})
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
		WithStateNotifier(&mock.MockStateNotifier{}),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, keys := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	gBlk, err := service.cfg.BeaconDB.GenesisBlock(ctx)
	require.NoError(t, err)
	gRoot, err := gBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})

	testState := gs.Copy()
	for i := types.Slot(1); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		blk, err := util.GenerateFullBlock(testState, keys, util.DefaultBlockGenConfig(), i)
		require.NoError(t, err)
		r, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk), r))
		testState, err = service.cfg.StateGen.StateByRoot(ctx, r)
		require.NoError(t, err)
	}
}

func TestInsertFinalizedDeposits(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts = append(opts, WithDepositCache(depositCache))
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	gBlk, err := service.cfg.BeaconDB.GenesisBlock(ctx)
	require.NoError(t, err)
	gRoot, err := gBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: gRoot[:]})
	gs = gs.Copy()
	assert.NoError(t, gs.SetEth1Data(&ethpb.Eth1Data{DepositCount: 10}))
	assert.NoError(t, service.cfg.StateGen.SaveState(ctx, [32]byte{'m', 'o', 'c', 'k'}, gs))
	zeroSig := [96]byte{}
	for i := uint64(0); i < uint64(4*params.BeaconConfig().SlotsPerEpoch); i++ {
		root := []byte(strconv.Itoa(int(i)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, &ethpb.Deposit{Data: &ethpb.Deposit_Data{
			PublicKey:             bytesutil.FromBytes48([fieldparams.BLSPubkeyLength]byte{}),
			WithdrawalCredentials: params.BeaconConfig().ZeroHash[:],
			Amount:                0,
			Signature:             zeroSig[:],
		}, Proof: [][]byte{root}}, 100+i, int64(i), bytesutil.ToBytes32(root)))
	}
	assert.NoError(t, service.insertFinalizedDeposits(ctx, [32]byte{'m', 'o', 'c', 'k'}))
	fDeposits := depositCache.FinalizedDeposits(ctx)
	assert.Equal(t, 9, int(fDeposits.MerkleTrieIndex), "Finalized deposits not inserted correctly")
	deps := depositCache.AllDeposits(ctx, big.NewInt(109))
	for _, d := range deps {
		assert.DeepEqual(t, [][]byte(nil), d.Proof, "Proofs are not empty")
	}
}

func TestRemoveBlockAttestationsInPool_Canonical(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyPruneCanonicalAtts: true,
	})
	defer resetCfg()

	genesis, keys := util.DeterministicGenesisState(t, 64)
	b, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
	require.NoError(t, service.cfg.BeaconDB.SaveGenesisBlockRoot(ctx, r))

	atts := b.Block.Body.Attestations
	require.NoError(t, service.cfg.AttPool.SaveAggregatedAttestations(atts))
	require.NoError(t, service.pruneCanonicalAttsFromPool(ctx, r, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	require.Equal(t, 0, service.cfg.AttPool.AggregatedAttestationCount())
}

func TestRemoveBlockAttestationsInPool_NonCanonical(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		CorrectlyPruneCanonicalAtts: true,
	})
	defer resetCfg()

	genesis, keys := util.DeterministicGenesisState(t, 64)
	b, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)

	atts := b.Block.Body.Attestations
	require.NoError(t, service.cfg.AttPool.SaveAggregatedAttestations(atts))
	require.NoError(t, service.pruneCanonicalAttsFromPool(ctx, r, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	require.Equal(t, 1, service.cfg.AttPool.AggregatedAttestationCount())
}

func Test_getStateVersionAndPayload(t *testing.T) {
	tests := []struct {
		name    string
		st      state.BeaconState
		version int
		header  *ethpb.ExecutionPayloadHeader
	}{
		{
			name: "phase 0 state",
			st: func() state.BeaconState {
				s, _ := util.DeterministicGenesisState(t, 1)
				return s
			}(),
			version: version.Phase0,
			header:  (*ethpb.ExecutionPayloadHeader)(nil),
		},
		{
			name: "altair state",
			st: func() state.BeaconState {
				s, _ := util.DeterministicGenesisStateAltair(t, 1)
				return s
			}(),
			version: version.Altair,
			header:  (*ethpb.ExecutionPayloadHeader)(nil),
		},
		{
			name: "bellatrix state",
			st: func() state.BeaconState {
				s, _ := util.DeterministicGenesisStateBellatrix(t, 1)
				require.NoError(t, s.SetLatestExecutionPayloadHeader(&ethpb.ExecutionPayloadHeader{
					BlockNumber: 1,
				}))
				return s
			}(),
			version: version.Bellatrix,
			header: &ethpb.ExecutionPayloadHeader{
				BlockNumber: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, header, err := getStateVersionAndPayload(tt.st)
			require.NoError(t, err)
			require.Equal(t, tt.version, version)
			require.DeepEqual(t, tt.header, header)
		})
	}
}
