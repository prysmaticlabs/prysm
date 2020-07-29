package blockchain

import (
	"context"
	"strings"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_OnBlock(t *testing.T) {
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB:        db,
		StateGen:        stategen.New(db, sc),
		ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, db.SaveBlock(ctx, genesis))
	validGenesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err)
	st := testutil.NewBeaconState()
	require.NoError(t, service.beaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(db, validGenesisRoot[:])
	require.NoError(t, err)
	random := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: validGenesisRoot[:]}}
	assert.NoError(t, db.SaveBlock(ctx, random))
	randomParentRoot, err := stateutil.BlockRoot(random.Block)
	assert.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: st.Slot(), Root: randomParentRoot[:]}))
	require.NoError(t, service.beaconDB.SaveState(ctx, st.Copy(), randomParentRoot))
	randomParentRoot2 := roots[1]
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: st.Slot(), Root: randomParentRoot2[:]}))
	require.NoError(t, service.beaconDB.SaveState(ctx, st.Copy(), bytesutil.ToBytes32(randomParentRoot2)))

	tests := []struct {
		name          string
		blk           *ethpb.BeaconBlock
		s             *stateTrie.BeaconState
		time          uint64
		wantErrString string
	}{
		{
			name:          "parent block root does not have a state",
			blk:           &ethpb.BeaconBlock{},
			s:             st.Copy(),
			wantErrString: "could not reconstruct parent state",
		},
		{
			name:          "block is from the future",
			blk:           &ethpb.BeaconBlock{ParentRoot: randomParentRoot[:], Slot: params.BeaconConfig().FarFutureEpoch},
			s:             st.Copy(),
			wantErrString: "far distant future",
		},
		{
			name:          "could not get finalized block",
			blk:           &ethpb.BeaconBlock{ParentRoot: randomParentRoot[:]},
			s:             st.Copy(),
			wantErrString: "is not a descendent of the current finalized block",
		},
		{
			name:          "same slot as finalized block",
			blk:           &ethpb.BeaconBlock{Slot: 0, ParentRoot: randomParentRoot2},
			s:             st.Copy(),
			wantErrString: "block is equal or earlier than finalized block, slot 0 < slot 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service.justifiedCheckpt = &ethpb.Checkpoint{Root: validGenesisRoot[:]}
			service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: validGenesisRoot[:]}
			service.finalizedCheckpt = &ethpb.Checkpoint{Root: validGenesisRoot[:]}
			service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: validGenesisRoot[:]}
			service.finalizedCheckpt.Root = roots[0]

			root, err := stateutil.BlockRoot(tt.blk)
			assert.NoError(t, err)
			err = service.onBlock(ctx, &ethpb.SignedBeaconBlock{Block: tt.blk}, root)
			assert.ErrorContains(t, tt.wantErrString, err)
		})
	}
}

func TestStore_OnBlockBatch(t *testing.T) {
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, sc),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, db.SaveBlock(ctx, genesis))

	st, keys := testutil.DeterministicGenesisState(t, 64)

	bState := st.Copy()

	blks := []*ethpb.SignedBeaconBlock{}
	blkRoots := [][32]byte{}
	var firstState *stateTrie.BeaconState
	for i := 1; i < 10; i++ {
		b, err := testutil.GenerateFullBlock(bState, keys, testutil.DefaultBlockGenConfig(), uint64(i))
		require.NoError(t, err)
		bState, err = state.ExecuteStateTransition(ctx, bState, b)
		require.NoError(t, err)
		if i == 1 {
			firstState = bState.Copy()
		}
		root, err := stateutil.BlockRoot(b.Block)
		require.NoError(t, err)
		blks = append(blks, b)
		blkRoots = append(blkRoots, root)
	}
	require.NoError(t, db.SaveBlock(context.Background(), blks[0]))
	require.NoError(t, service.stateGen.SaveState(ctx, blkRoots[0], firstState))
	_, _, err = service.onBlockBatch(ctx, blks[1:], blkRoots[1:])
	require.NoError(t, err)
}

func TestRemoveStateSinceLastFinalized_EmptyStartSlot(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	service.genesisTime = time.Now()

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{})
	require.NoError(t, err)
	assert.Equal(t, true, update, "Should be able to update justified")
	lastJustifiedBlk := testutil.NewBeaconBlock()
	lastJustifiedBlk.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	lastJustifiedRoot, err := stateutil.BlockRoot(lastJustifiedBlk.Block)
	require.NoError(t, err)
	newJustifiedBlk := testutil.NewBeaconBlock()
	newJustifiedBlk.Block.Slot = 1
	newJustifiedBlk.Block.ParentRoot = bytesutil.PadTo(lastJustifiedRoot[:], 32)
	newJustifiedRoot, err := stateutil.BlockRoot(newJustifiedBlk.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, newJustifiedBlk))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, lastJustifiedBlk))

	diff := (params.BeaconConfig().SlotsPerEpoch - 1) * params.BeaconConfig().SecondsPerSlot
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: lastJustifiedRoot[:]}
	update, err = service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	require.NoError(t, err)
	assert.Equal(t, true, update, "Should be able to update justified")
}

func TestShouldUpdateJustified_ReturnFalse(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	lastJustifiedBlk := testutil.NewBeaconBlock()
	lastJustifiedBlk.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	lastJustifiedRoot, err := stateutil.BlockRoot(lastJustifiedBlk.Block)
	require.NoError(t, err)
	newJustifiedBlk := testutil.NewBeaconBlock()
	newJustifiedBlk.Block.ParentRoot = bytesutil.PadTo(lastJustifiedRoot[:], 32)
	newJustifiedRoot, err := stateutil.BlockRoot(newJustifiedBlk.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, newJustifiedBlk))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, lastJustifiedBlk))

	diff := (params.BeaconConfig().SlotsPerEpoch - 1) * params.BeaconConfig().SecondsPerSlot
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: lastJustifiedRoot[:]}

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	require.NoError(t, err)
	assert.Equal(t, false, update, "Should not be able to update justified, received true")
}

func TestCachedPreState_CanGetFromStateSummary(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, sc),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	s, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 1, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
	require.NoError(t, err)
	r := [32]byte{'A'}
	b := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r[:]}
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: 1, Root: r[:]}))
	require.NoError(t, service.stateGen.SaveState(ctx, r, s))
	require.NoError(t, service.verifyBlkPreState(ctx, b))
}

func TestCachedPreState_CanGetFromDB(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, sc),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	r := [32]byte{'A'}
	b := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r[:]}

	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	err = service.verifyBlkPreState(ctx, b)
	wanted := "could not reconstruct parent state"
	assert.ErrorContains(t, wanted, err)

	s, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 1})
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: 1, Root: r[:]}))
	require.NoError(t, service.stateGen.SaveState(ctx, r, s))
	require.NoError(t, service.verifyBlkPreState(ctx, b))
}

func TestUpdateJustified_CouldUpdateBest(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, StateGen: stategen.New(db, cache.NewStateSummaryCache())}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	signedBlock := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, signedBlock))
	r, err := stateutil.BlockRoot(signedBlock.Block)
	require.NoError(t, err)
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: []byte{'A'}}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: []byte{'A'}}
	st := testutil.NewBeaconState()
	service.initSyncState[r] = st.Copy()
	require.NoError(t, db.SaveState(ctx, st.Copy(), r))

	// Could update
	s := testutil.NewBeaconState()
	require.NoError(t, s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: r[:]}))
	require.NoError(t, service.updateJustified(context.Background(), s))

	assert.Equal(t, s.CurrentJustifiedCheckpoint().Epoch, service.bestJustifiedCheckpt.Epoch, "Incorrect justified epoch in service")

	// Could not update
	service.bestJustifiedCheckpt.Epoch = 2
	require.NoError(t, service.updateJustified(context.Background(), s))

	assert.Equal(t, uint64(2), service.bestJustifiedCheckpt.Epoch, "Incorrect justified epoch in service")
}

func TestFillForkChoiceMissingBlocks_CanSave(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	service.forkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	service.finalizedCheckpt = &ethpb.Checkpoint{}

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesis))
	validGenesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err)
	st := testutil.NewBeaconState()

	require.NoError(t, service.beaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))
	roots, err := blockTree1(db, validGenesisRoot[:])
	require.NoError(t, err)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	block := &ethpb.BeaconBlock{Slot: 9, ParentRoot: roots[8], Body: &ethpb.BeaconBlockBody{Graffiti: []byte{}}}

	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), block, beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	assert.Equal(t, 5, len(service.forkChoiceStore.Nodes()), "Miss match nodes")
	assert.Equal(t, true, service.forkChoiceStore.HasNode(bytesutil.ToBytes32(roots[4])), "Didn't save node")
	assert.Equal(t, true, service.forkChoiceStore.HasNode(bytesutil.ToBytes32(roots[6])), "Didn't save node")
	assert.Equal(t, true, service.forkChoiceStore.HasNode(bytesutil.ToBytes32(roots[8])), "Didn't save node")
}

func TestFillForkChoiceMissingBlocks_FilterFinalized(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	service.forkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	// Set finalized epoch to 1.
	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1}

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, db.SaveBlock(ctx, genesis))
	validGenesisRoot, err := stateutil.BlockRoot(genesis.Block)
	assert.NoError(t, err)
	st := testutil.NewBeaconState()

	require.NoError(t, service.beaconDB.SaveState(ctx, st.Copy(), validGenesisRoot))

	// Define a tree branch, slot 63 <- 64 <- 65
	b63 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 63, Body: &ethpb.BeaconBlockBody{}}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b63))
	r63, err := stateutil.BlockRoot(b63.Block)
	require.NoError(t, err)
	b64 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 64, ParentRoot: r63[:], Body: &ethpb.BeaconBlockBody{}}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b64))
	r64, err := stateutil.BlockRoot(b64.Block)
	require.NoError(t, err)
	b65 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 65, ParentRoot: r64[:], Body: &ethpb.BeaconBlockBody{}}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b65))

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	err = service.fillInForkChoiceMissingBlocks(
		context.Background(), b65.Block, beaconState.FinalizedCheckpoint(), beaconState.CurrentJustifiedCheckpoint())
	require.NoError(t, err)

	// There should be 2 nodes, block 65 and block 64.
	assert.Equal(t, 2, len(service.forkChoiceStore.Nodes()), "Miss match nodes")

	// Block with slot 63 should be in fork choice because it's less than finalized epoch 1.
	assert.Equal(t, true, service.forkChoiceStore.HasNode(r63), "Didn't save node")
}

// blockTree1 constructs the following tree:
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree1(db db.Database, genesisRoot []byte) ([][]byte, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, err := b0.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, err := b1.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r0[:]}
	r3, err := b3.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r3[:]}
	r4, err := b4.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r4[:]}
	r5, err := b5.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, err := b6.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r5[:]}
	r7, err := b7.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, err := b8.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	st := testutil.NewBeaconState()

	for _, b := range []*ethpb.BeaconBlock{b0, b1, b3, b4, b5, b6, b7, b8} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
		beaconBlock.Block.Body = &ethpb.BeaconBlockBody{}
		if err := db.SaveBlock(context.Background(), beaconBlock); err != nil {
			return nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, err
		}
	}
	if err := db.SaveState(context.Background(), st.Copy(), r1); err != nil {
		return nil, err
	}
	if err := db.SaveState(context.Background(), st.Copy(), r7); err != nil {
		return nil, err
	}
	if err := db.SaveState(context.Background(), st.Copy(), r8); err != nil {
		return nil, err
	}
	return [][]byte{r0[:], r1[:], nil, r3[:], r4[:], r5[:], r6[:], r7[:], r8[:]}, nil
}

func TestCurrentSlot_HandlesOverflow(t *testing.T) {
	svc := Service{genesisTime: roughtime.Now().Add(1 * time.Hour)}

	slot := svc.CurrentSlot()
	require.Equal(t, uint64(0), slot, "Unexpected slot")
}

func TestAncestor_HandleSkipSlot(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'a'}}
	r1, err := b1.HashTreeRoot()
	require.NoError(t, err)
	b100 := &ethpb.BeaconBlock{Slot: 100, ParentRoot: r1[:]}
	r100, err := b100.HashTreeRoot()
	require.NoError(t, err)
	b200 := &ethpb.BeaconBlock{Slot: 200, ParentRoot: r100[:]}
	r200, err := b200.HashTreeRoot()
	require.NoError(t, err)
	for _, b := range []*ethpb.BeaconBlock{b1, b100, b200} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
		beaconBlock.Block.Body = &ethpb.BeaconBlockBody{}
		require.NoError(t, db.SaveBlock(context.Background(), beaconBlock))
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

func TestEnsureRootNotZeroHashes(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	service.genesisRoot = [32]byte{'a'}

	r := service.ensureRootNotZeros(params.BeaconConfig().ZeroHash)
	assert.Equal(t, service.genesisRoot, r, "Did not get wanted justified root")
	root := [32]byte{'b'}
	r = service.ensureRootNotZeros(root)
	assert.Equal(t, root, r, "Did not get wanted justified root")
}

func TestFinalizedImpliesNewJustified(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	ctx := context.Background()
	type args struct {
		cachedCheckPoint        *ethpb.Checkpoint
		stateCheckPoint         *ethpb.Checkpoint
		diffFinalizedCheckPoint bool
	}
	tests := []struct {
		name string
		args args
		want *ethpb.Checkpoint
	}{
		{
			name: "Same justified, do nothing",
			args: args{
				cachedCheckPoint: &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}},
				stateCheckPoint:  &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}},
			},
			want: &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}},
		},
		{
			name: "Different justified, higher epoch, cache new justified",
			args: args{
				cachedCheckPoint: &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}},
				stateCheckPoint:  &ethpb.Checkpoint{Epoch: 2, Root: []byte{'b'}},
			},
			want: &ethpb.Checkpoint{Epoch: 2, Root: []byte{'b'}},
		},
		{
			name: "finalized has different justified, cache new justified",
			args: args{
				cachedCheckPoint:        &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}},
				stateCheckPoint:         &ethpb.Checkpoint{Epoch: 1, Root: []byte{'b'}},
				diffFinalizedCheckPoint: true,
			},
			want: &ethpb.Checkpoint{Epoch: 1, Root: []byte{'b'}},
		},
	}
	for _, test := range tests {
		beaconState := testutil.NewBeaconState()
		require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(test.args.stateCheckPoint))
		service, err := NewService(ctx, &Config{BeaconDB: db, StateGen: stategen.New(db, sc), ForkChoiceStore: protoarray.New(0, 0, [32]byte{})})
		require.NoError(t, err)
		service.justifiedCheckpt = test.args.cachedCheckPoint
		require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo(test.want.Root, 32)}))
		genesisState := testutil.NewBeaconState()
		require.NoError(t, service.beaconDB.SaveState(ctx, genesisState, bytesutil.ToBytes32(test.want.Root)))

		if test.args.diffFinalizedCheckPoint {
			b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'a'}}
			r1, err := b1.HashTreeRoot()
			require.NoError(t, err)
			b100 := &ethpb.BeaconBlock{Slot: 100, ParentRoot: r1[:]}
			r100, err := b100.HashTreeRoot()
			require.NoError(t, err)
			for _, b := range []*ethpb.BeaconBlock{b1, b100} {
				beaconBlock := testutil.NewBeaconBlock()
				beaconBlock.Block.Slot = b.Slot
				beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
				require.NoError(t, service.beaconDB.SaveBlock(context.Background(), beaconBlock))
			}
			service.finalizedCheckpt = &ethpb.Checkpoint{Root: []byte{'c'}, Epoch: 1}
			service.justifiedCheckpt.Root = r100[:]
		}

		require.NoError(t, service.finalizedImpliesNewJustified(ctx, beaconState))
		assert.Equal(t, true, attestationutil.CheckPointIsEqual(test.want, service.justifiedCheckpt), "Did not get wanted check point")
	}
}

func TestVerifyBlkDescendant(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	ctx := context.Background()

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 1
	r, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, b))

	b1 := testutil.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.Body.Graffiti = bytesutil.PadTo([]byte{'a'}, 32)
	r1, err := stateutil.BlockRoot(b1.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, b1))

	type args struct {
		parentRoot    [32]byte
		finalizedRoot [32]byte
		finalizedSlot uint64
	}
	tests := []struct {
		name        string
		args        args
		shouldError bool
		err         string
	}{
		{
			name: "could not get finalized block in block service cache",
			args: args{
				finalizedRoot: [32]byte{'a'},
			},
			shouldError: true,
			err:         "nil finalized block",
		},
		{
			name: "could not get finalized block root in DB",
			args: args{
				finalizedRoot: r,
				parentRoot:    [32]byte{'a'},
			},
			shouldError: true,
			err:         "could not get finalized block root",
		},
		{
			name: "is not descendant",
			args: args{
				finalizedRoot: r1,
				parentRoot:    r,
			},
			shouldError: true,
			err:         "is not a descendent of the current finalized block slot",
		},
		{
			name: "is descendant",
			args: args{
				finalizedRoot: r,
				parentRoot:    r,
			},
			shouldError: false,
		},
	}
	for _, test := range tests {
		service, err := NewService(ctx, &Config{BeaconDB: db, StateGen: stategen.New(db, sc), ForkChoiceStore: protoarray.New(0, 0, [32]byte{})})
		require.NoError(t, err)
		service.finalizedCheckpt = &ethpb.Checkpoint{
			Root: test.args.finalizedRoot[:],
		}
		err = service.VerifyBlkDescendant(ctx, test.args.parentRoot)
		if test.shouldError {
			if err == nil || !strings.Contains(err.Error(), test.err) {
				t.Error("Did not get wanted error")
			}
		} else if err != nil {
			t.Error(err)
		}
	}
}

func TestUpdateJustifiedInitSync(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	gBlk := testutil.NewBeaconBlock()
	gRoot, err := stateutil.BlockRoot(gBlk.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, gBlk))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: gRoot[:]}))
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, gRoot))
	service.genesisRoot = gRoot
	currentCp := &ethpb.Checkpoint{Epoch: 1}
	service.justifiedCheckpt = currentCp
	newCp := &ethpb.Checkpoint{Epoch: 2, Root: gRoot[:]}

	require.NoError(t, service.updateJustifiedInitSync(ctx, newCp))

	assert.DeepEqual(t, currentCp, service.prevJustifiedCheckpt, "Incorrect previous justified checkpoint")
	assert.DeepEqual(t, newCp, service.CurrentJustifiedCheckpt(), "Incorrect current justified checkpoint in cache")
	cp, err := service.beaconDB.JustifiedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.DeepEqual(t, newCp, cp, "Incorrect current justified checkpoint in db")
}
