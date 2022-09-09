package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution"
	mockExecution "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestStore_OnBlock_ProtoArray(t *testing.T) {
	ctx := context.Background()

	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesis)
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
	util.SaveBlock(t, ctx, beaconDB, random)
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
			fRoot := bytesutil.ToBytes32(roots[0])
			require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Root: fRoot}))
			root, err := tt.blk.Block.HashTreeRoot()
			assert.NoError(t, err)
			wsb, err := consensusblocks.NewSignedBeaconBlock(tt.blk)
			require.NoError(t, err)
			err = service.onBlock(ctx, wsb, root)
			assert.ErrorContains(t, tt.wantErrString, err)
		})
	}
}

func TestStore_OnBlock_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()

	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesis)
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
	util.SaveBlock(t, ctx, beaconDB, random)
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
			fRoot := bytesutil.ToBytes32(roots[0])
			require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Root: fRoot}))
			root, err := tt.blk.Block.HashTreeRoot()
			assert.NoError(t, err)
			wsb, err := consensusblocks.NewSignedBeaconBlock(tt.blk)
			require.NoError(t, err)
			err = service.onBlock(ctx, wsb, root)
			assert.ErrorContains(t, tt.wantErrString, err)
		})
	}
}

func TestStore_OnBlockBatch_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	bState := st.Copy()

	var blks []interfaces.SignedBeaconBlock
	var blkRoots [][32]byte
	for i := 0; i < 97; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wsb)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.saveInitSyncBlock(ctx, root, wsb))
		blks = append(blks, wsb)
		blkRoots = append(blkRoots, root)
	}
	err = service.onBlockBatch(ctx, blks, blkRoots[1:])
	require.ErrorIs(t, errWrongBlockCount, err)
	err = service.onBlockBatch(ctx, blks, blkRoots)
	require.NoError(t, err)
	jcp := service.CurrentJustifiedCheckpt()
	jroot := bytesutil.ToBytes32(jcp.Root)
	require.Equal(t, blkRoots[63], jroot)
	require.Equal(t, types.Epoch(2), service.cfg.ForkChoiceStore.JustifiedCheckpoint().Epoch)
}

func TestStore_OnBlockBatch_PruneOK_Protoarray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	bState := st.Copy()

	var blks []interfaces.SignedBeaconBlock
	var blkRoots [][32]byte
	for i := 0; i < 320; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wsb)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.saveInitSyncBlock(ctx, root, wsb))
		blks = append(blks, wsb)
		blkRoots = append(blkRoots, root)
	}
	err = service.onBlockBatch(ctx, blks, blkRoots)
	require.NoError(t, err)
	require.Equal(t, 65, service.ForkChoicer().NodeCount())
}

func TestStore_OnBlockBatch_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	bState := st.Copy()

	var blks []interfaces.SignedBeaconBlock
	var blkRoots [][32]byte
	for i := 0; i < 97; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wsb)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.saveInitSyncBlock(ctx, root, wsb))
		wsb, err = consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		blks = append(blks, wsb)
		blkRoots = append(blkRoots, root)
	}
	err = service.onBlockBatch(ctx, blks, blkRoots[1:])
	require.ErrorIs(t, errWrongBlockCount, err)
	err = service.onBlockBatch(ctx, blks, blkRoots)
	require.NoError(t, err)
	jcp := service.CurrentJustifiedCheckpt()
	jroot := bytesutil.ToBytes32(jcp.Root)
	require.Equal(t, blkRoots[63], jroot)
	require.Equal(t, types.Epoch(2), service.cfg.ForkChoiceStore.JustifiedCheckpoint().Epoch)
}

func TestStore_OnBlockBatch_NotifyNewPayload(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	st, keys := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	bState := st.Copy()

	var blks []interfaces.SignedBeaconBlock
	var blkRoots [][32]byte
	blkCount := 4
	for i := 0; i <= blkCount; i++ {
		b, err := util.GenerateFullBlock(bState, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		bState, err = transition.ExecuteStateTransition(ctx, bState, wsb)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.saveInitSyncBlock(ctx, root, wsb))
		blks = append(blks, wsb)
		blkRoots = append(blkRoots, root)
	}
	err = service.onBlockBatch(ctx, blks, blkRoots)
	require.NoError(t, err)
}

func TestCachedPreState_CanGetFromStateSummary_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(1))
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))

	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: root[:]}))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, root, st))
	require.NoError(t, service.verifyBlkPreState(ctx, wsb.Block()))
}

func TestCachedPreState_CanGetFromStateSummary_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(1))
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))

	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: root[:]}))
	require.NoError(t, service.cfg.StateGen.SaveState(ctx, root, st))
	require.NoError(t, service.verifyBlkPreState(ctx, wsb.Block()))
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
	service.cfg.ForkChoiceStore = protoarray.New()

	st, _ := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))

	roots, err := blockTree1(t, beaconDB, service.originBlockRoot[:])
	require.NoError(t, err)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 9
	blk.Block.ParentRoot = roots[8]
	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)

	r0 := bytesutil.ToBytes32(roots[0])
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 0, Root: r0}))
	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 4 nodes from the block tree 1. B3 - B4 - B6 - B8
	// plus 1 node for genesis block root.
	// block 0 is not inserted because its slot is 0 which is invalid
	assert.Equal(t, 5, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(service.originBlockRoot), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[3])), "Didn't save node")
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
	service.cfg.ForkChoiceStore = doublylinkedtree.New()

	st, _ := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))

	roots, err := blockTree1(t, beaconDB, service.originBlockRoot[:])
	require.NoError(t, err)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 9
	blk.Block.ParentRoot = roots[8]
	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)

	// save invalid block at slot 0 because doubly linked tree enforces that
	// the parent of the last block inserted is the tree node.
	fcp := &ethpb.Checkpoint{Epoch: 0, Root: service.originBlockRoot[:]}
	r0 := bytesutil.ToBytes32(roots[0])
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, r0, service.originBlockRoot, [32]byte{}, fcp, fcp)
	require.NoError(t, err)
	require.NoError(t, service.ForkChoicer().InsertNode(ctx, state, blkRoot))
	fcp2 := &forkchoicetypes.Checkpoint{Epoch: 0, Root: r0}
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(fcp2))

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	// plus 1 node for genesis block root.
	assert.Equal(t, 6, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(service.originBlockRoot), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(r0), "Didn't save node")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(bytesutil.ToBytes32(roots[3])), "Didn't save node")
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
	service.cfg.ForkChoiceStore = protoarray.New()

	st, _ := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))
	roots, err := blockTree1(t, beaconDB, service.originBlockRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 9
	blk.Block.ParentRoot = roots[8]

	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)

	r0 := bytesutil.ToBytes32(roots[0])
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 0, Root: r0}))

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 4 nodes from the block tree 1. B3 - B4 - B6 - B8
	// plus the origin block root
	assert.Equal(t, 5, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	// Ensure all roots and their respective blocks exist.
	wantedRoots := [][]byte{roots[3], roots[4], roots[6], roots[8]}
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
	service.cfg.ForkChoiceStore = doublylinkedtree.New()

	st, _ := util.DeterministicGenesisState(t, 64)
	require.NoError(t, service.saveGenesisData(ctx, st))

	roots, err := blockTree1(t, beaconDB, service.originBlockRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 9
	blk.Block.ParentRoot = roots[8]

	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)

	// save invalid block at slot 0 because doubly linked tree enforces that
	// the parent of the last block inserted is the tree node.
	fcp := &ethpb.Checkpoint{Epoch: 0, Root: service.originBlockRoot[:]}
	r0 := bytesutil.ToBytes32(roots[0])
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, r0, service.originBlockRoot, [32]byte{}, fcp, fcp)
	require.NoError(t, err)
	require.NoError(t, service.ForkChoicer().InsertNode(ctx, state, blkRoot))
	fcp2 := &forkchoicetypes.Checkpoint{Epoch: 0, Root: r0}
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(fcp2))

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	// plus the origin block root
	assert.Equal(t, 6, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
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
	service.cfg.ForkChoiceStore = protoarray.New()

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesis)
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	assert.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))

	// Define a tree branch, slot 63 <- 64 <- 65 <- 66
	b63 := util.NewBeaconBlock()
	b63.Block.Slot = 63
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b63)
	r63, err := b63.Block.HashTreeRoot()
	require.NoError(t, err)
	b64 := util.NewBeaconBlock()
	b64.Block.Slot = 64
	b64.Block.ParentRoot = r63[:]
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b64)
	r64, err := b64.Block.HashTreeRoot()
	require.NoError(t, err)
	b65 := util.NewBeaconBlock()
	b65.Block.Slot = 65
	b65.Block.ParentRoot = r64[:]
	r65, err := b65.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b65)
	b66 := util.NewBeaconBlock()
	b66.Block.Slot = 66
	b66.Block.ParentRoot = r65[:]
	wsb := util.SaveBlock(t, ctx, service.cfg.BeaconDB, b66)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	// Set finalized epoch to 2.
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 2, Root: r64}))
	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// We should have saved 1 node: block 65
	assert.Equal(t, 1, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(r65), "Didn't save node")
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
	service.cfg.ForkChoiceStore = doublylinkedtree.New()

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesis)
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	assert.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))

	// Define a tree branch, slot 63 <- 64 <- 65
	b63 := util.NewBeaconBlock()
	b63.Block.Slot = 63
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b63)
	r63, err := b63.Block.HashTreeRoot()
	require.NoError(t, err)
	b64 := util.NewBeaconBlock()
	b64.Block.Slot = 64
	b64.Block.ParentRoot = r63[:]
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b64)
	r64, err := b64.Block.HashTreeRoot()
	require.NoError(t, err)
	b65 := util.NewBeaconBlock()
	b65.Block.Slot = 65
	b65.Block.ParentRoot = r64[:]
	r65, err := b65.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b65)
	b66 := util.NewBeaconBlock()
	b66.Block.Slot = 66
	b66.Block.ParentRoot = r65[:]
	wsb := util.SaveBlock(t, ctx, service.cfg.BeaconDB, b66)

	beaconState, _ := util.DeterministicGenesisState(t, 32)

	// Set finalized epoch to 2.
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 2, Root: r64}))
	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// There should be 1 node: block 65
	assert.Equal(t, 1, service.cfg.ForkChoiceStore.NodeCount(), "Miss match nodes")
	assert.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(r65), "Didn't save node")
}

func TestFillForkChoiceMissingBlocks_FinalizedSibling_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ForkChoiceStore = doublylinkedtree.New()

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesis)
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 9
	blk.Block.ParentRoot = roots[8]

	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), wsb.Block(), beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.Equal(t, errNotDescendantOfFinalized.Error(), err.Error())
}

// blockTree1 constructs the following tree:
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
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
		wsb, err := consensusblocks.NewSignedBeaconBlock(beaconBlock)
		require.NoError(t, err)
		if err := beaconDB.SaveBlock(context.Background(), wsb); err != nil {
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

	fcs := doublylinkedtree.New()
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
		util.SaveBlock(t, context.Background(), beaconDB, beaconBlock)
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
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	for _, b := range []*ethpb.SignedBeaconBlock{b1, b100, b200} {
		beaconBlock := util.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		st, blkRoot, err := prepareForkchoiceState(context.Background(), b.Block.Slot, r, bytesutil.ToBytes32(b.Block.ParentRoot), params.BeaconConfig().ZeroHash, ojc, ofc)
		require.NoError(t, err)
		require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
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

	fcs := doublylinkedtree.New()
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
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	for _, b := range []*ethpb.SignedBeaconBlock{b1, b100, b200} {
		beaconBlock := util.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		util.SaveBlock(t, context.Background(), beaconDB, beaconBlock)
	}

	st, blkRoot, err := prepareForkchoiceState(context.Background(), 200, r200, r200, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

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

	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	b := util.NewBeaconBlock()
	b.Block.Slot = 32
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, beaconDB, b)

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 32
	b1.Block.Body.Graffiti = bytesutil.PadTo([]byte{'a'}, 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, beaconDB, b1)

	type args struct {
		parentRoot    [32]byte
		finalizedRoot [32]byte
	}
	tests := []struct {
		name             string
		args             args
		wantedErr        string
		invalidBlockRoot bool
	}{
		{
			name: "could not get finalized block in block service cache",
			args: args{
				finalizedRoot: [32]byte{'a'},
			},
			wantedErr: "block not found in cache or db",
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
			wantedErr:        "is not a descendant of the current finalized block slot",
			invalidBlockRoot: true,
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
		require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Root: tt.args.finalizedRoot, Epoch: 1}))
		err = service.VerifyFinalizedBlkDescendant(ctx, tt.args.parentRoot)
		if tt.wantedErr != "" {
			assert.ErrorContains(t, tt.wantedErr, err)
			if tt.invalidBlockRoot {
				require.Equal(t, true, IsInvalidBlock(err))
			}
		} else if err != nil {
			assert.NoError(t, err)
		}
	}
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

func TestOnBlock_CanFinalize_WithOnTick(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithAttestationPool(attestations.NewPool()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, keys := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	require.NoError(t, fcs.UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Root: service.originBlockRoot}))

	testState := gs.Copy()
	for i := types.Slot(1); i <= 4*params.BeaconConfig().SlotsPerEpoch; i++ {
		blk, err := util.GenerateFullBlock(testState, keys, util.DefaultBlockGenConfig(), i)
		require.NoError(t, err)
		r, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		require.NoError(t, fcs.NewSlot(ctx, i))
		require.NoError(t, service.onBlock(ctx, wsb, r))
		testState, err = service.cfg.StateGen.StateByRoot(ctx, r)
		require.NoError(t, err)
	}
	cp := service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(3), cp.Epoch)
	cp = service.FinalizedCheckpt()
	require.Equal(t, types.Epoch(2), cp.Epoch)

	// The update should persist in DB.
	j, err := service.cfg.BeaconDB.JustifiedCheckpoint(ctx)
	require.NoError(t, err)
	cp = service.CurrentJustifiedCheckpt()
	require.Equal(t, j.Epoch, cp.Epoch)
	f, err := service.cfg.BeaconDB.FinalizedCheckpoint(ctx)
	require.NoError(t, err)
	cp = service.FinalizedCheckpt()
	require.Equal(t, f.Epoch, cp.Epoch)
}

func TestOnBlock_CanFinalize(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithAttestationPool(attestations.NewPool()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, keys := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))

	testState := gs.Copy()
	for i := types.Slot(1); i <= 4*params.BeaconConfig().SlotsPerEpoch; i++ {
		blk, err := util.GenerateFullBlock(testState, keys, util.DefaultBlockGenConfig(), i)
		require.NoError(t, err)
		r, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, r))
		testState, err = service.cfg.StateGen.StateByRoot(ctx, r)
		require.NoError(t, err)
	}
	cp := service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(3), cp.Epoch)
	cp = service.FinalizedCheckpt()
	require.Equal(t, types.Epoch(2), cp.Epoch)

	// The update should persist in DB.
	j, err := service.cfg.BeaconDB.JustifiedCheckpoint(ctx)
	require.NoError(t, err)
	cp = service.CurrentJustifiedCheckpt()
	require.Equal(t, j.Epoch, cp.Epoch)
	f, err := service.cfg.BeaconDB.FinalizedCheckpoint(ctx)
	require.NoError(t, err)
	cp = service.FinalizedCheckpt()
	require.Equal(t, f.Epoch, cp.Epoch)
}

func TestOnBlock_NilBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	err = service.onBlock(ctx, nil, [32]byte{})
	require.Equal(t, true, IsInvalidBlock(err))
}

func TestOnBlock_InvalidSignature(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
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

	blk, err := util.GenerateFullBlock(gs, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	blk.Signature = []byte{'a'} // Mutate the signature.
	r, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, r)
	require.Equal(t, true, IsInvalidBlock(err))
}

func TestOnBlock_CallNewPayloadAndForkchoiceUpdated(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithAttestationPool(attestations.NewPool()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, keys := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	testState := gs.Copy()
	for i := types.Slot(1); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		blk, err := util.GenerateFullBlock(testState, keys, util.DefaultBlockGenConfig(), i)
		require.NoError(t, err)
		r, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, r))
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
	gs = gs.Copy()
	assert.NoError(t, gs.SetEth1Data(&ethpb.Eth1Data{DepositCount: 10}))
	assert.NoError(t, gs.SetEth1DepositIndex(8))
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
	assert.Equal(t, 7, int(fDeposits.MerkleTrieIndex), "Finalized deposits not inserted correctly")
	deps := depositCache.AllDeposits(ctx, big.NewInt(107))
	for _, d := range deps {
		assert.DeepEqual(t, [][]byte(nil), d.Proof, "Proofs are not empty")
	}
}

func TestInsertFinalizedDeposits_MultipleFinalizedRoutines(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts = append(opts, WithDepositCache(depositCache))
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	gs = gs.Copy()
	assert.NoError(t, gs.SetEth1Data(&ethpb.Eth1Data{DepositCount: 7}))
	assert.NoError(t, gs.SetEth1DepositIndex(6))
	assert.NoError(t, service.cfg.StateGen.SaveState(ctx, [32]byte{'m', 'o', 'c', 'k'}, gs))
	gs2 := gs.Copy()
	assert.NoError(t, gs2.SetEth1Data(&ethpb.Eth1Data{DepositCount: 15}))
	assert.NoError(t, gs2.SetEth1DepositIndex(13))
	assert.NoError(t, service.cfg.StateGen.SaveState(ctx, [32]byte{'m', 'o', 'c', 'k', '2'}, gs2))
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
	// Insert 3 deposits before hand.
	depositCache.InsertFinalizedDeposits(ctx, 2)

	assert.NoError(t, service.insertFinalizedDeposits(ctx, [32]byte{'m', 'o', 'c', 'k'}))
	fDeposits := depositCache.FinalizedDeposits(ctx)
	assert.Equal(t, 5, int(fDeposits.MerkleTrieIndex), "Finalized deposits not inserted correctly")

	deps := depositCache.AllDeposits(ctx, big.NewInt(105))
	for _, d := range deps {
		assert.DeepEqual(t, [][]byte(nil), d.Proof, "Proofs are not empty")
	}

	// Insert New Finalized State with higher deposit count.
	assert.NoError(t, service.insertFinalizedDeposits(ctx, [32]byte{'m', 'o', 'c', 'k', '2'}))
	fDeposits = depositCache.FinalizedDeposits(ctx)
	assert.Equal(t, 12, int(fDeposits.MerkleTrieIndex), "Finalized deposits not inserted correctly")
	deps = depositCache.AllDeposits(ctx, big.NewInt(112))
	for _, d := range deps {
		assert.DeepEqual(t, [][]byte(nil), d.Proof, "Proofs are not empty")
	}
}

func TestRemoveBlockAttestationsInPool_Canonical(t *testing.T) {
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
	wsb, err := consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, service.pruneCanonicalAttsFromPool(ctx, r, wsb))
	require.Equal(t, 0, service.cfg.AttPool.AggregatedAttestationCount())
}

func TestRemoveBlockAttestationsInPool_NonCanonical(t *testing.T) {
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
	wsb, err := consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, service.pruneCanonicalAttsFromPool(ctx, r, wsb))
	require.Equal(t, 1, service.cfg.AttPool.AggregatedAttestationCount())
}

func Test_getStateVersionAndPayload(t *testing.T) {
	tests := []struct {
		name    string
		st      state.BeaconState
		version int
		header  *enginev1.ExecutionPayloadHeader
	}{
		{
			name: "phase 0 state",
			st: func() state.BeaconState {
				s, _ := util.DeterministicGenesisState(t, 1)
				return s
			}(),
			version: version.Phase0,
			header:  (*enginev1.ExecutionPayloadHeader)(nil),
		},
		{
			name: "altair state",
			st: func() state.BeaconState {
				s, _ := util.DeterministicGenesisStateAltair(t, 1)
				return s
			}(),
			version: version.Altair,
			header:  (*enginev1.ExecutionPayloadHeader)(nil),
		},
		{
			name: "bellatrix state",
			st: func() state.BeaconState {
				s, _ := util.DeterministicGenesisStateBellatrix(t, 1)
				wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader{
					BlockNumber: 1,
				})
				require.NoError(t, err)
				require.NoError(t, s.SetLatestExecutionPayloadHeader(wrappedHeader))
				return s
			}(),
			version: version.Bellatrix,
			header: &enginev1.ExecutionPayloadHeader{
				BlockNumber: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, header, err := getStateVersionAndPayload(tt.st)
			require.NoError(t, err)
			require.Equal(t, tt.version, ver)
			require.DeepEqual(t, tt.header, header)
		})
	}
}

func Test_validateMergeTransitionBlock(t *testing.T) {
	cfg := params.BeaconConfig()
	cfg.TerminalTotalDifficulty = "2"
	cfg.TerminalBlockHash = params.BeaconConfig().ZeroHash
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
		WithAttestationPool(attestations.NewPool()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	aHash := common.BytesToHash([]byte("a"))
	bHash := common.BytesToHash([]byte("b"))

	tests := []struct {
		name         string
		stateVersion int
		header       *enginev1.ExecutionPayloadHeader
		payload      *enginev1.ExecutionPayload
		errString    string
	}{
		{
			name:         "state older than Bellatrix, nil payload",
			stateVersion: 1,
			payload:      nil,
			errString:    "attempted to wrap nil",
		},
		{
			name:         "state older than Bellatrix, empty payload",
			stateVersion: 1,
			payload: &enginev1.ExecutionPayload{
				ParentHash:    make([]byte, fieldparams.RootLength),
				FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
				StateRoot:     make([]byte, fieldparams.RootLength),
				ReceiptsRoot:  make([]byte, fieldparams.RootLength),
				LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
				PrevRandao:    make([]byte, fieldparams.RootLength),
				BaseFeePerGas: make([]byte, fieldparams.RootLength),
				BlockHash:     make([]byte, fieldparams.RootLength),
			},
		},
		{
			name:         "state older than Bellatrix, non empty payload",
			stateVersion: 1,
			payload: &enginev1.ExecutionPayload{
				ParentHash: aHash[:],
			},
		},
		{
			name:         "state is Bellatrix, nil payload",
			stateVersion: 2,
			payload:      nil,
			errString:    "attempted to wrap nil",
		},
		{
			name:         "state is Bellatrix, empty payload",
			stateVersion: 2,
			payload: &enginev1.ExecutionPayload{
				ParentHash:    make([]byte, fieldparams.RootLength),
				FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
				StateRoot:     make([]byte, fieldparams.RootLength),
				ReceiptsRoot:  make([]byte, fieldparams.RootLength),
				LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
				PrevRandao:    make([]byte, fieldparams.RootLength),
				BaseFeePerGas: make([]byte, fieldparams.RootLength),
				BlockHash:     make([]byte, fieldparams.RootLength),
			},
		},
		{
			name:         "state is Bellatrix, non empty payload, empty header",
			stateVersion: 2,
			payload: &enginev1.ExecutionPayload{
				ParentHash: aHash[:],
			},
			header: &enginev1.ExecutionPayloadHeader{
				ParentHash:       make([]byte, fieldparams.RootLength),
				FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
				StateRoot:        make([]byte, fieldparams.RootLength),
				ReceiptsRoot:     make([]byte, fieldparams.RootLength),
				LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
				PrevRandao:       make([]byte, fieldparams.RootLength),
				BaseFeePerGas:    make([]byte, fieldparams.RootLength),
				BlockHash:        make([]byte, fieldparams.RootLength),
				TransactionsRoot: make([]byte, fieldparams.RootLength),
			},
		},
		{
			name:         "state is Bellatrix, non empty payload, non empty header",
			stateVersion: 2,
			payload: &enginev1.ExecutionPayload{
				ParentHash: aHash[:],
			},
			header: &enginev1.ExecutionPayloadHeader{
				BlockNumber: 1,
			},
		},
		{
			name:         "state is Bellatrix, non empty payload, nil header",
			stateVersion: 2,
			payload: &enginev1.ExecutionPayload{
				ParentHash: aHash[:],
			},
			errString: "attempted to wrap nil object",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &mockExecution.EngineClient{BlockByHashMap: map[[32]byte]*enginev1.ExecutionBlock{}}
			e.BlockByHashMap[aHash] = &enginev1.ExecutionBlock{
				Header: gethtypes.Header{
					ParentHash: bHash,
				},
				TotalDifficulty: "0x2",
			}
			e.BlockByHashMap[bHash] = &enginev1.ExecutionBlock{
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("3")),
				},
				TotalDifficulty: "0x1",
			}
			service.cfg.ExecutionEngineCaller = e
			b := util.HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{})
			b.Block.Body.ExecutionPayload = tt.payload
			blk, err := consensusblocks.NewSignedBeaconBlock(b)
			require.NoError(t, err)
			err = service.validateMergeTransitionBlock(ctx, tt.stateVersion, tt.header, blk)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_insertSlashingsToForkChoiceStore(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	beaconState, privKeys := util.DeterministicGenesisState(t, 100)
	att1 := util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1},
		},
		AttestingIndices: []uint64{0, 1},
	})
	domain, err := signing.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorsRoot())
	require.NoError(t, err)
	signingRoot, err := signing.ComputeSigningRoot(att1.Data, domain)
	assert.NoError(t, err, "Could not get signing root of beacon block header")
	sig0 := privKeys[0].Sign(signingRoot[:])
	sig1 := privKeys[1].Sign(signingRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()

	att2 := util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0, 1},
	})
	signingRoot, err = signing.ComputeSigningRoot(att2.Data, domain)
	assert.NoError(t, err, "Could not get signing root of beacon block header")
	sig0 = privKeys[0].Sign(signingRoot[:])
	sig1 = privKeys[1].Sign(signingRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()
	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}
	b := util.NewBeaconBlock()
	b.Block.Body.AttesterSlashings = slashings
	wb, err := consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	service.InsertSlashingsToForkChoiceStore(ctx, wb.Block().Body().AttesterSlashings())
}

func TestOnBlock_ProcessBlocksParallel(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithDepositCache(depositCache),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithAttestationPool(attestations.NewPool()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, keys := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))

	blk1, err := util.GenerateFullBlock(gs, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	r1, err := blk1.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb1, err := consensusblocks.NewSignedBeaconBlock(blk1)
	require.NoError(t, err)
	blk2, err := util.GenerateFullBlock(gs, keys, util.DefaultBlockGenConfig(), 2)
	require.NoError(t, err)
	r2, err := blk2.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb2, err := consensusblocks.NewSignedBeaconBlock(blk2)
	require.NoError(t, err)
	blk3, err := util.GenerateFullBlock(gs, keys, util.DefaultBlockGenConfig(), 3)
	require.NoError(t, err)
	r3, err := blk3.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb3, err := consensusblocks.NewSignedBeaconBlock(blk3)
	require.NoError(t, err)
	blk4, err := util.GenerateFullBlock(gs, keys, util.DefaultBlockGenConfig(), 4)
	require.NoError(t, err)
	r4, err := blk4.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb4, err := consensusblocks.NewSignedBeaconBlock(blk4)
	require.NoError(t, err)

	logHook := logTest.NewGlobal()
	for i := 0; i < 10; i++ {
		fc := &ethpb.Checkpoint{}
		st, blkRoot, err := prepareForkchoiceState(ctx, 0, wsb1.Block().ParentRoot(), [32]byte{}, [32]byte{}, fc, fc)
		require.NoError(t, err)
		require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
		var wg sync.WaitGroup
		wg.Add(4)
		go func() {
			require.NoError(t, service.onBlock(ctx, wsb1, r1))
			wg.Done()
		}()
		go func() {
			require.NoError(t, service.onBlock(ctx, wsb2, r2))
			wg.Done()
		}()
		go func() {
			require.NoError(t, service.onBlock(ctx, wsb3, r3))
			wg.Done()
		}()
		go func() {
			require.NoError(t, service.onBlock(ctx, wsb4, r4))
			wg.Done()
		}()
		wg.Wait()
		require.LogsDoNotContain(t, logHook, "New head does not exist in DB. Do nothing")
		require.NoError(t, service.cfg.BeaconDB.DeleteBlock(ctx, r1))
		require.NoError(t, service.cfg.BeaconDB.DeleteBlock(ctx, r2))
		require.NoError(t, service.cfg.BeaconDB.DeleteBlock(ctx, r3))
		require.NoError(t, service.cfg.BeaconDB.DeleteBlock(ctx, r4))
		service.cfg.ForkChoiceStore = doublylinkedtree.New()
	}
}

func Test_verifyBlkFinalizedSlot_invalidBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	require.NoError(t, service.ForkChoicer().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: 1}))
	blk := util.HydrateBeaconBlock(&ethpb.BeaconBlock{Slot: 1})
	wb, err := consensusblocks.NewBeaconBlock(blk)
	require.NoError(t, err)
	err = service.verifyBlkFinalizedSlot(wb)
	require.Equal(t, true, IsInvalidBlock(err))
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 17 is the last block in Epoch
// 2. Block 18 justifies block 12 (the first in Epoch 2) and Block 19 returns
// INVALID from FCU, with LVH block 17. No head is viable. We check
// that the node is optimistic and that we can actually import a block on top of
// 17 and recover.
func TestStore_NoViableHead_FCU_Protoarray(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	for i := 12; i < 18; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}
	// Check that we haven't justified the second epoch yet
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), jc.Epoch)

	// import a block that justifies the second epoch
	driftGenesisTime(service, 18, 0)
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(validHeadState, keys, util.DefaultBlockGenConfig(), 18)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	firstInvalidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, firstInvalidRoot)
	require.NoError(t, err)
	jc = service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)

	sjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), sjc.Epoch)
	lvh := b.Block.Body.ExecutionPayload.ParentHash
	// check our head
	require.Equal(t, firstInvalidRoot, service.ForkChoicer().CachedHeadRoot())

	// import another block to find out that it was invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrInvalidPayloadStatus, ForkChoiceUpdatedResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)
	// Check that forkchoice's head is the last invalid block imported. The
	// store's headroot is the previous head (since the invalid block did
	// not finish importing) one and that the node is optimistic
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, firstInvalidRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	// Check the newly imported block is head, it justified the right
	// checkpoint and the node is no longer optimistic
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, jc.Epoch, sjc.Epoch)
	require.Equal(t, jc.Root, bytesutil.ToBytes32(sjc.Root))
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 17 is the last block in Epoch
// 2. Block 18 justifies block 12 (the first in Epoch 2) and Block 19 returns
// INVALID from FCU, with LVH block 17. No head is viable. We check
// that the node is optimistic and that we can actually import a block on top of
// 17 and recover.
func TestStore_NoViableHead_FCU_DoublyLinkedTree(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	for i := 12; i < 18; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}
	// Check that we haven't justified the second epoch yet
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), jc.Epoch)

	// import a block that justifies the second epoch
	driftGenesisTime(service, 18, 0)
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(validHeadState, keys, util.DefaultBlockGenConfig(), 18)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	firstInvalidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, firstInvalidRoot)
	require.NoError(t, err)
	jc = service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)

	sjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), sjc.Epoch)
	lvh := b.Block.Body.ExecutionPayload.ParentHash
	// check our head
	require.Equal(t, firstInvalidRoot, service.ForkChoicer().CachedHeadRoot())

	// import another block to find out that it was invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrInvalidPayloadStatus, ForkChoiceUpdatedResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)
	// Check that forkchoice's head is the last invalid block imported. The
	// store's headroot is the previous head (since the invalid block did
	// not finish importing) one and that the node is optimistic
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, firstInvalidRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	// Check the newly imported block is head, it justified the right
	// checkpoint and the node is no longer optimistic
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, jc.Epoch, sjc.Epoch)
	require.Equal(t, jc.Root, bytesutil.ToBytes32(sjc.Root))
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 17 is the last block in Epoch
// 2. Block 18 justifies block 12 (the first in Epoch 2) and Block 19 returns
// INVALID from NewPayload, with LVH block 17. No head is viable. We check
// that the node is optimistic and that we can actually import a block on top of
// 17 and recover.
func TestStore_NoViableHead_NewPayload_DoublyLinkedTree(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	for i := 12; i < 18; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}
	// Check that we haven't justified the second epoch yet
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), jc.Epoch)

	// import a block that justifies the second epoch
	driftGenesisTime(service, 18, 0)
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(validHeadState, keys, util.DefaultBlockGenConfig(), 18)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	firstInvalidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, firstInvalidRoot)
	require.NoError(t, err)
	jc = service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)

	sjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), sjc.Epoch)
	lvh := b.Block.Body.ExecutionPayload.ParentHash
	// check our head
	require.Equal(t, firstInvalidRoot, service.ForkChoicer().CachedHeadRoot())

	// import another block to find out that it was invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrInvalidPayloadStatus, NewPayloadResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)
	// Check that forkchoice's head and store's headroot are the previous head (since the invalid block did
	// not finish importing and it was never imported to forkchoice). Check
	// also that the node is optimistic
	require.Equal(t, firstInvalidRoot, service.ForkChoicer().CachedHeadRoot())
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, firstInvalidRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	// Check the newly imported block is head, it justified the right
	// checkpoint and the node is no longer optimistic
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, jc.Epoch, sjc.Epoch)
	require.Equal(t, jc.Root, bytesutil.ToBytes32(sjc.Root))
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 17 is the last block in Epoch
// 2. Block 18 justifies block 12 (the first in Epoch 2) and Block 19 returns
// INVALID from NewPayload, with LVH block 17. No head is viable. We check
// that the node is optimistic and that we can actually import a block on top of
// 17 and recover.
func TestStore_NoViableHead_NewPayload_Protoarray(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	for i := 12; i < 18; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}
	// Check that we haven't justified the second epoch yet
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), jc.Epoch)

	// import a block that justifies the second epoch
	driftGenesisTime(service, 18, 0)
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(validHeadState, keys, util.DefaultBlockGenConfig(), 18)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	firstInvalidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, firstInvalidRoot)
	require.NoError(t, err)
	jc = service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)

	sjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), sjc.Epoch)
	lvh := b.Block.Body.ExecutionPayload.ParentHash
	// check our head
	require.Equal(t, firstInvalidRoot, service.ForkChoicer().CachedHeadRoot())

	// import another block to find out that it was invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrInvalidPayloadStatus, NewPayloadResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)
	// Check that forkchoice's head and store's headroot are the previous head (since the invalid block did
	// not finish importing and it was never imported to forkchoice). Check
	// also that the node is optimistic
	require.Equal(t, firstInvalidRoot, service.ForkChoicer().CachedHeadRoot())
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, firstInvalidRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	// Check the newly imported block is head, it justified the right
	// checkpoint and the node is no longer optimistic
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, jc.Epoch, sjc.Epoch)
	require.Equal(t, jc.Root, bytesutil.ToBytes32(sjc.Root))
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 12 is the first block in Epoch
// 2 (and the merge block in this sequence). Block 18 justifies it and Block 19 returns
// INVALID from NewPayload, with LVH block 12. No head is viable. We check
// that the node is optimistic and that we can actually import a chain of blocks on top of
// 12 and recover. Notice that it takes two epochs to fully recover, and we stay
// optimistic for the whole time.
func TestStore_NoViableHead_Liveness_DoublyLinkedTree(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	// import the merge block
	driftGenesisTime(service, 12, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 12)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	lastValidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, lastValidRoot)
	require.NoError(t, err)
	// save the post state and the payload Hash of this block since it will
	// be the LVH
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	lvh := b.Block.Body.ExecutionPayload.BlockHash
	validjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), validjc.Epoch)

	// import blocks 13 through 18 to justify 12
	invalidRoots := make([][32]byte, 19-13)
	for i := 13; i < 19; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		invalidRoots[i-13], err = b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, invalidRoots[i-13])
		require.NoError(t, err)
	}
	// Check that we have justified the second epoch
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)
	invalidHeadRoot := service.ForkChoicer().CachedHeadRoot()

	// import block 19 to find out that the whole chain 13--18 was in fact
	// invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrInvalidPayloadStatus, NewPayloadResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)

	// Check that forkchoice's head and store's headroot are the previous head (since the invalid block did
	// not finish importing and it was never imported to forkchoice). Check
	// also that the node is optimistic
	require.Equal(t, invalidHeadRoot, service.ForkChoicer().CachedHeadRoot())
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, invalidHeadRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// Check that the invalid blocks are not in database
	for i := 0; i < 19-13; i++ {
		require.Equal(t, false, service.cfg.BeaconDB.HasBlock(ctx, invalidRoots[i]))
	}

	// Check that the node's justified checkpoint does not agree with the
	// last valid state's justified checkpoint
	sjc := service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(2), sjc.Epoch)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, root))
	// Check that the head is still INVALID and the node is still optimistic
	require.Equal(t, invalidHeadRoot, service.ForkChoicer().CachedHeadRoot())
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
	st, err = service.cfg.StateGen.StateByRoot(ctx, root)
	require.NoError(t, err)
	// Import blocks 21--30 (Epoch 3 was not enough to justify 2)
	for i := 21; i < 30; i++ {
		driftGenesisTime(service, int64(i), 0)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
		st, err = service.cfg.StateGen.StateByRoot(ctx, root)
		require.NoError(t, err)
	}
	// Head should still be INVALID and the node optimistic
	require.Equal(t, invalidHeadRoot, service.ForkChoicer().CachedHeadRoot())
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// Import block 30, it should justify Epoch 4 and become HEAD, the node
	// recovers
	driftGenesisTime(service, 30, 0)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 30)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(4), sjc.Epoch)
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 12 is the first block in Epoch
// 2 (and the merge block in this sequence). Block 18 justifies it and Block 19 returns
// INVALID from NewPayload, with LVH block 12. No head is viable. We check
// that the node is optimistic and that we can actually import a chain of blocks on top of
// 12 and recover. Notice that it takes two epochs to fully recover, and we stay
// optimistic for the whole time.
func TestStore_NoViableHead_Liveness_Protoarray(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	// import the merge block
	driftGenesisTime(service, 12, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 12)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	lastValidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, lastValidRoot)
	require.NoError(t, err)
	// save the post state and the payload Hash of this block since it will
	// be the LVH
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	lvh := b.Block.Body.ExecutionPayload.BlockHash
	validjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), validjc.Epoch)

	// import blocks 13 through 18 to justify 12, these are invalid blocks
	invalidRoots := make([][32]byte, 19-13)
	for i := 13; i < 19; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		invalidRoots[i-13], err = b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, invalidRoots[i-13])
		require.NoError(t, err)
	}
	// Check that we have justified the second epoch
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)
	invalidHeadRoot := service.ForkChoicer().CachedHeadRoot()

	// import block 19 to find out that the whole chain 13--18 was in fact
	// invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrInvalidPayloadStatus, NewPayloadResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)

	// Check that forkchoice's head and store's headroot are the previous head (since the invalid block did
	// not finish importing and it was never imported to forkchoice). Check
	// also that the node is optimistic
	require.Equal(t, invalidHeadRoot, service.ForkChoicer().CachedHeadRoot())
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, invalidHeadRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// Check that the invalid blocks are not in database
	for i := 0; i < 19-13; i++ {
		require.Equal(t, false, service.cfg.BeaconDB.HasBlock(ctx, invalidRoots[i]))
	}

	// Check that the node's justified checkpoint does not agree with the
	// last valid state's justified checkpoint
	sjc := service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(2), sjc.Epoch)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, root))
	// Check that the head is still INVALID and the node is still optimistic
	require.Equal(t, invalidHeadRoot, service.ForkChoicer().CachedHeadRoot())
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
	st, err = service.cfg.StateGen.StateByRoot(ctx, root)
	require.NoError(t, err)
	// Import blocks 21--30 (Epoch 3 was not enough to justify 2)
	for i := 21; i < 30; i++ {
		driftGenesisTime(service, int64(i), 0)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
		st, err = service.cfg.StateGen.StateByRoot(ctx, root)
		require.NoError(t, err)
	}
	// Head should still be INVALID and the node optimistic
	require.Equal(t, invalidHeadRoot, service.ForkChoicer().CachedHeadRoot())
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	// Import block 30, it should justify Epoch 4 and become HEAD, the node
	// recovers
	driftGenesisTime(service, 30, 0)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 30)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(4), sjc.Epoch)
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
}

type newForkChoicer func() forkchoice.ForkChoicer

func TestStore_NoViableHead_Reboot(t *testing.T) {
	cases := []struct {
		new  newForkChoicer
		name string
	}{
		{
			new:  func() forkchoice.ForkChoicer { return doublylinkedtree.New() },
			name: "doublylinkedtree",
		},
		{
			new:  func() forkchoice.ForkChoicer { return protoarray.New() },
			name: "protoarray",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			noViableHead_Reboot(t, c.new)
		})
	}
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 12 is the first block in Epoch
// 2 (and the merge block in this sequence). Block 18 justifies it and Block 19 returns
// INVALID from NewPayload, with LVH block 12. No head is viable. We check that
// the node can reboot from this state
func noViableHead_Reboot(t *testing.T, newfc newForkChoicer) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	attSrv, err := attestations.NewService(ctx, &attestations.Config{})
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(newfc()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
		WithAttestationService(attSrv),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	genesisState, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")
	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")
	require.NoError(t, service.saveGenesisData(ctx, genesisState))

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, genesisState, genesisRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, genesisRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	// import the merge block
	driftGenesisTime(service, 12, 0)
	st, err := service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 12)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	lastValidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, lastValidRoot)
	require.NoError(t, err)
	// save the post state and the payload Hash of this block since it will
	// be the LVH
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	lvh := b.Block.Body.ExecutionPayload.BlockHash
	validjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), validjc.Epoch)

	// import blocks 13 through 18 to justify 12
	for i := 13; i < 19; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}
	// Check that we have justified the second epoch
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)

	// import block 19 to find out that the whole chain 13--18 was in fact
	// invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrInvalidPayloadStatus, NewPayloadResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)

	// Check that the headroot/state are not in DB and restart the node
	blk, err := service.cfg.BeaconDB.HeadBlock(ctx)
	require.NoError(t, err) // HeadBlock returns no error when headroot == nil
	require.Equal(t, blk, nil)

	service.cfg.ForkChoiceStore = newfc()
	require.NoError(t, service.StartFromSavedState(genesisState))

	// Forkchoice has the genesisRoot loaded at startup
	require.Equal(t, genesisRoot, service.ensureRootNotZeros(service.ForkChoicer().CachedHeadRoot()))
	// Service's store has the finalized state as headRoot
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
	require.Equal(t, false, service.ForkChoicer().AllTipsAreInvalid())

	// Check that the node's justified checkpoint does not agree with the
	// last valid state's justified checkpoint
	sjc := service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(2), sjc.Epoch)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, root))
	// Check that the head is still INVALID and the node is optimistic
	require.Equal(t, genesisRoot, service.ensureRootNotZeros(service.ForkChoicer().CachedHeadRoot()))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisRoot, bytesutil.ToBytes32(headRoot))

	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
	require.Equal(t, true, service.ForkChoicer().AllTipsAreInvalid())
	st, err = service.cfg.StateGen.StateByRoot(ctx, root)
	require.NoError(t, err)

	// Import blocks 21--23
	for i := 21; i < 24; i++ {
		driftGenesisTime(service, int64(i), 0)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err = b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
		st, err = service.cfg.StateGen.StateByRoot(ctx, root)
		require.NoError(t, err)
	}
	// Head should still be INVALID and the node is optimistic
	require.Equal(t, genesisRoot, service.ensureRootNotZeros(service.ForkChoicer().CachedHeadRoot()))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisRoot, bytesutil.ToBytes32(headRoot))

	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
	require.Equal(t, true, service.ForkChoicer().AllTipsAreInvalid())

	// Import block 24, it should justify Epoch 3 and become HEAD, the node
	// recovers
	driftGenesisTime(service, 24, 0)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 24)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, bytesutil.ToBytes32(headRoot))

	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(3), sjc.Epoch)
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
	require.Equal(t, false, service.ForkChoicer().AllTipsAreInvalid())
}

// See the description in #10777 and #10782 for the full setup
// We sync optimistically a chain of blocks. Block 12 is the first block in Epoch
// 2 (and the merge block in this sequence). Block 18 justifies it and Block 19 returns
// INVALID from NewPayload, with LVH block 12. No head is viable. We check that
// the node can reboot from this state
func TestStore_NoViableHead_Reboot_Protoarray(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerEpoch = 6
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.SafeSlotsToImportOptimistically = 0
	params.OverrideBeaconConfig(config)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockEngine := &mockExecution.EngineClient{ErrNewPayload: execution.ErrAcceptedSyncingPayloadStatus, ErrForkchoiceUpdated: execution.ErrAcceptedSyncingPayloadStatus}
	attSrv, err := attestations.NewService(ctx, &attestations.Config{})
	require.NoError(t, err)
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(protoarray.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
		WithExecutionEngineCaller(mockEngine),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
		WithAttestationService(attSrv),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	genesisState, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")
	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")
	require.NoError(t, service.saveGenesisData(ctx, genesisState))

	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, genesisState, genesisRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, genesisRoot), "Could not save genesis state")

	for i := 1; i < 6; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}

	for i := 6; i < 12; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockAltair(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
	}

	// import the merge block
	driftGenesisTime(service, 12, 0)
	st, err := service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 12)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	lastValidRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, lastValidRoot)
	require.NoError(t, err)
	// save the post state and the payload Hash of this block since it will
	// be the LVH
	validHeadState, err := service.HeadState(ctx)
	require.NoError(t, err)
	lvh := b.Block.Body.ExecutionPayload.BlockHash
	validjc := validHeadState.CurrentJustifiedCheckpoint()
	require.Equal(t, types.Epoch(0), validjc.Epoch)

	// import blocks 13 through 18 to justify 12
	for i := 13; i < 19; i++ {
		driftGenesisTime(service, int64(i), 0)
		st, err := service.HeadState(ctx)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.onBlock(ctx, wsb, root))
	}
	// Check that we have justified the second epoch
	jc := service.ForkChoicer().JustifiedCheckpoint()
	require.Equal(t, types.Epoch(2), jc.Epoch)

	// import block 19 to find out that the whole chain 13--18 was in fact
	// invalid
	mockEngine = &mockExecution.EngineClient{ErrNewPayload: execution.ErrInvalidPayloadStatus, NewPayloadResp: lvh}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 19, 0)
	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 19)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.ErrorContains(t, "received an INVALID payload from execution engine", err)

	// Check that the headroot/state are not in DB and restart the node
	blk, err := service.cfg.BeaconDB.HeadBlock(ctx)
	require.NoError(t, err) // HeadBlock returns no error when headroot == nil
	require.Equal(t, blk, nil)

	service.cfg.ForkChoiceStore = protoarray.New()
	require.NoError(t, service.StartFromSavedState(genesisState))

	require.Equal(t, genesisRoot, service.ensureRootNotZeros(service.ForkChoicer().CachedHeadRoot()))
	// Service's store has the finalized state as headRoot
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisRoot, bytesutil.ToBytes32(headRoot))
	optimistic, err := service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
	require.Equal(t, false, service.ForkChoicer().AllTipsAreInvalid())

	// Check that the node's justified checkpoint does not agree with the
	// last valid state's justified checkpoint
	sjc := service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(2), sjc.Epoch)

	// import another block based on the last valid head state
	mockEngine = &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = mockEngine
	driftGenesisTime(service, 20, 0)
	b, err = util.GenerateFullBlockBellatrix(validHeadState, keys, &util.BlockGenConfig{}, 20)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, root))
	// Check that the head is still INVALID and the node is optimistic
	require.Equal(t, genesisRoot, service.ensureRootNotZeros(service.ForkChoicer().CachedHeadRoot()))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisRoot, bytesutil.ToBytes32(headRoot))

	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
	require.Equal(t, true, service.ForkChoicer().AllTipsAreInvalid())
	st, err = service.cfg.StateGen.StateByRoot(ctx, root)
	require.NoError(t, err)
	// Import blocks 21--23
	for i := 21; i < 24; i++ {
		driftGenesisTime(service, int64(i), 0)
		require.NoError(t, err)
		b, err := util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), types.Slot(i))
		require.NoError(t, err)
		wsb, err := consensusblocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		err = service.onBlock(ctx, wsb, root)
		require.NoError(t, err)
		st, err = service.cfg.StateGen.StateByRoot(ctx, root)
		require.NoError(t, err)
	}
	// Head should still be INVALID and the node is optimistic
	require.Equal(t, genesisRoot, service.ensureRootNotZeros(service.ForkChoicer().CachedHeadRoot()))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, genesisRoot, bytesutil.ToBytes32(headRoot))

	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
	require.Equal(t, true, service.ForkChoicer().AllTipsAreInvalid())

	// Import block 24, it should justify Epoch 3 and become HEAD, the node
	// recovers
	driftGenesisTime(service, 24, 0)
	b, err = util.GenerateFullBlockBellatrix(st, keys, util.DefaultBlockGenConfig(), 24)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	err = service.onBlock(ctx, wsb, root)
	require.NoError(t, err)
	require.Equal(t, root, service.ForkChoicer().CachedHeadRoot())
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, bytesutil.ToBytes32(headRoot))

	sjc = service.CurrentJustifiedCheckpt()
	require.Equal(t, types.Epoch(3), sjc.Epoch)
	optimistic, err = service.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
	require.Equal(t, false, service.ForkChoicer().AllTipsAreInvalid())
}

func TestOnBlock_HandleBlockAttestations(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithAttestationPool(attestations.NewPool()),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
		WithStateNotifier(&mock.MockStateNotifier{}),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	st, keys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	require.NoError(t, service.saveGenesisData(ctx, st))

	genesis := blocks.NewGenesisBlock(stateRoot[:])
	wsb, err := consensusblocks.NewSignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb), "Could not save genesis block")
	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, parentRoot), "Could not save genesis state")
	require.NoError(t, service.cfg.BeaconDB.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err := util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.onBlock(ctx, wsb, root))

	st, err = service.HeadState(ctx)
	require.NoError(t, err)
	b, err = util.GenerateFullBlock(st, keys, util.DefaultBlockGenConfig(), 2)
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)

	// prepare another block that is not inserted
	st3, err := transition.ExecuteStateTransition(ctx, st, wsb)
	require.NoError(t, err)
	b3, err := util.GenerateFullBlock(st3, keys, util.DefaultBlockGenConfig(), 3)
	require.NoError(t, err)
	wsb3, err := consensusblocks.NewSignedBeaconBlock(b3)
	require.NoError(t, err)

	require.Equal(t, 1, len(wsb.Block().Body().Attestations()))
	a := wsb.Block().Body().Attestations()[0]
	r := bytesutil.ToBytes32(a.Data.BeaconBlockRoot)
	require.Equal(t, true, service.cfg.ForkChoiceStore.HasNode(r))

	require.Equal(t, 1, len(wsb.Block().Body().Attestations()))
	a3 := wsb3.Block().Body().Attestations()[0]
	r3 := bytesutil.ToBytes32(a3.Data.BeaconBlockRoot)
	require.Equal(t, false, service.cfg.ForkChoiceStore.HasNode(r3))

	require.NoError(t, service.handleBlockAttestations(ctx, wsb.Block(), st)) // fine to use the same committe as st
	require.Equal(t, 0, service.cfg.AttPool.ForkchoiceAttestationCount())
	require.NoError(t, service.handleBlockAttestations(ctx, wsb3.Block(), st3)) // fine to use the same committe as st
	require.Equal(t, 1, len(service.cfg.AttPool.BlockAttestations()))
}

// Helper function to simulate the block being on time or delayed for proposer
// boost. It alters the genesisTime tracked by the store.
func driftGenesisTime(s *Service, slot int64, delay int64) {
	offset := int64(slot*int64(params.BeaconConfig().SecondsPerSlot) - delay)
	s.SetGenesisTime(time.Unix(time.Now().Unix()-offset, 0))
}
