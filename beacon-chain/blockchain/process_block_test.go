package blockchain

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStore_OnBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Error(err)
	}
	validGenesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Error(err)
	}
	st := testutil.NewBeaconState()
	if err := service.beaconDB.SaveState(ctx, st.Copy(), validGenesisRoot); err != nil {
		t.Fatal(err)
	}
	roots, err := blockTree1(db, validGenesisRoot[:])
	if err != nil {
		t.Fatal(err)
	}
	random := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: validGenesisRoot[:]}}
	if err := db.SaveBlock(ctx, random); err != nil {
		t.Error(err)
	}
	randomParentRoot, err := stateutil.BlockRoot(random.Block)
	if err != nil {
		t.Error(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: st.Slot(), Root: randomParentRoot[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, st.Copy(), randomParentRoot); err != nil {
		t.Fatal(err)
	}
	randomParentRoot2 := roots[1]
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: st.Slot(), Root: randomParentRoot2[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, st.Copy(), bytesutil.ToBytes32(randomParentRoot2)); err != nil {
		t.Fatal(err)
	}

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
			wantErrString: "provided block root does not have block saved in the db",
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
			wantErrString: "block from slot 0 is not a descendent of the current finalized block",
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
			if err != nil {
				t.Error(err)
			}
			_, err = service.onBlock(ctx, &ethpb.SignedBeaconBlock{Block: tt.blk}, root)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrString) {
				t.Errorf("Store.OnBlock() error = %v, wantErr = %v", err, tt.wantErrString)
			}
		})
	}
}

func TestRemoveStateSinceLastFinalized(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Save 100 blocks in DB, each has a state.
	numBlocks := 100
	totalBlocks := make([]*ethpb.SignedBeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	for i := 0; i < len(totalBlocks); i++ {
		totalBlocks[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot: uint64(i),
			},
		}
		r, err := stateutil.BlockRoot(totalBlocks[i].Block)
		if err != nil {
			t.Fatal(err)
		}
		s := testutil.NewBeaconState()
		if err := s.SetSlot(uint64(i)); err != nil {
			t.Fatal(err)
		}
		if err := service.beaconDB.SaveState(ctx, s, r); err != nil {
			t.Fatal(err)
		}
		if err := service.beaconDB.SaveBlock(ctx, totalBlocks[i]); err != nil {
			t.Fatal(err)
		}
		blockRoots = append(blockRoots, r)
		if err := service.beaconDB.SaveHeadBlockRoot(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	// New finalized epoch: 1
	finalizedEpoch := uint64(1)
	finalizedSlot := finalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	endSlot := helpers.StartSlot(finalizedEpoch+1) - 1 // Inclusive
	if err := service.rmStatesOlderThanLastFinalized(ctx, 0, endSlot); err != nil {
		t.Fatal(err)
	}
	for _, r := range blockRoots {
		s, err := service.beaconDB.State(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		// Also verifies genesis state didnt get deleted
		if s != nil && s.Slot() != finalizedSlot && s.Slot() != 0 && s.Slot() < endSlot {
			t.Errorf("State with slot %d should not be in DB", s.Slot())
		}
	}

	// New finalized epoch: 5
	newFinalizedEpoch := uint64(5)
	newFinalizedSlot := newFinalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	endSlot = helpers.StartSlot(newFinalizedEpoch+1) - 1 // Inclusive
	if err := service.rmStatesOlderThanLastFinalized(ctx, helpers.StartSlot(finalizedEpoch+1)-1, endSlot); err != nil {
		t.Fatal(err)
	}
	for _, r := range blockRoots {
		s, err := service.beaconDB.State(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		// Also verifies genesis state didnt get deleted
		if s != nil && s.Slot() != newFinalizedSlot && s.Slot() != finalizedSlot && s.Slot() != 0 && s.Slot() < endSlot {
			t.Errorf("State with slot %d should not be in DB", s.Slot())
		}
	}
}

func TestRemoveStateSinceLastFinalized_EmptyStartSlot(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	service.genesisTime = time.Now()

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{})
	if err != nil {
		t.Fatal(err)
	}
	if !update {
		t.Error("Should be able to update justified, received false")
	}

	lastJustifiedBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{ParentRoot: []byte{'G'}}}
	lastJustifiedRoot, err := stateutil.BlockRoot(lastJustifiedBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	newJustifiedBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: lastJustifiedRoot[:]}}
	newJustifiedRoot, err := stateutil.BlockRoot(newJustifiedBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, newJustifiedBlk); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, lastJustifiedBlk); err != nil {
		t.Fatal(err)
	}

	diff := (params.BeaconConfig().SlotsPerEpoch - 1) * params.BeaconConfig().SecondsPerSlot
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: lastJustifiedRoot[:]}
	update, err = service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	if err != nil {
		t.Fatal(err)
	}
	if !update {
		t.Error("Should be able to update justified, received false")
	}
}

func TestShouldUpdateJustified_ReturnFalse(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	lastJustifiedBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{ParentRoot: []byte{'G'}}}
	lastJustifiedRoot, err := stateutil.BlockRoot(lastJustifiedBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	newJustifiedBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{ParentRoot: lastJustifiedRoot[:]}}
	newJustifiedRoot, err := stateutil.BlockRoot(newJustifiedBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, newJustifiedBlk); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, lastJustifiedBlk); err != nil {
		t.Fatal(err)
	}

	diff := (params.BeaconConfig().SlotsPerEpoch - 1) * params.BeaconConfig().SecondsPerSlot
	service.genesisTime = time.Unix(time.Now().Unix()-int64(diff), 0)
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: lastJustifiedRoot[:]}

	update, err := service.shouldUpdateCurrentJustified(ctx, &ethpb.Checkpoint{Root: newJustifiedRoot[:]})
	if err != nil {
		t.Fatal(err)
	}
	if update {
		t.Error("Should not be able to update justified, received true")
	}
}

func TestCachedPreState_CanGetFromStateSummary(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	s, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 1, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
	if err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'A'}
	b := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r[:]}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: 1, Root: r[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.stateGen.SaveState(ctx, r, s); err != nil {
		t.Fatal(err)
	}

	received, err := service.verifyBlkPreState(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.InnerStateUnsafe(), received.InnerStateUnsafe()) {
		t.Error("cached state not the same")
	}
}

func TestCachedPreState_CanGetFromDB(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	r := [32]byte{'A'}
	b := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r[:]}

	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	_, err = service.verifyBlkPreState(ctx, b)
	wanted := "could not reconstruct parent state"
	if err.Error() != wanted {
		t.Error("Did not get wanted error")
	}

	s, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: 1, Root: r[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.stateGen.SaveState(ctx, r, s); err != nil {
		t.Fatal(err)
	}

	received, err := service.verifyBlkPreState(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	if s.Slot() != received.Slot() {
		t.Error("cached state not the same")
	}
}

func TestSaveInitState_CanSaveDelete(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for i := uint64(0); i < 64; i++ {
		b := &ethpb.BeaconBlock{Slot: i}
		s := testutil.NewBeaconState()
		if err := s.SetSlot(i); err != nil {
			t.Fatal(err)
		}
		r, err := ssz.HashTreeRoot(b)
		if err != nil {
			t.Fatal(err)
		}
		service.initSyncState[r] = s
	}

	// Set finalized root as slot 32
	finalizedRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: 32})
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := s.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: 1, Root: finalizedRoot[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.saveInitState(ctx, s); err != nil {
		t.Fatal(err)
	}

	// Verify finalized state is saved in DB
	finalizedState, err := service.beaconDB.State(ctx, finalizedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if finalizedState == nil {
		t.Error("finalized state can't be nil")
	}
}

func TestUpdateJustified_CouldUpdateBest(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	signedBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, signedBlock); err != nil {
		t.Fatal(err)
	}
	r, err := stateutil.BlockRoot(signedBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: []byte{'A'}}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: []byte{'A'}}
	st := testutil.NewBeaconState()
	service.initSyncState[r] = st.Copy()
	if err := db.SaveState(ctx, st.Copy(), r); err != nil {
		t.Fatal(err)
	}

	// Could update
	s := testutil.NewBeaconState()
	if err := s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: r[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.updateJustified(context.Background(), s); err != nil {
		t.Fatal(err)
	}

	if service.bestJustifiedCheckpt.Epoch != s.CurrentJustifiedCheckpoint().Epoch {
		t.Error("Incorrect justified epoch in service")
	}

	// Could not update
	service.bestJustifiedCheckpt.Epoch = 2
	if err := service.updateJustified(context.Background(), s); err != nil {
		t.Fatal(err)
	}

	if service.bestJustifiedCheckpt.Epoch != 2 {
		t.Error("Incorrect justified epoch in service")
	}
}

func TestFilterBlockRoots_CanFilter(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	fBlock := &ethpb.BeaconBlock{}
	fRoot, err := stateutil.BlockRoot(fBlock)
	if err != nil {
		t.Fatal(err)
	}
	hBlock := &ethpb.BeaconBlock{Slot: 1}
	headRoot, err := stateutil.BlockRoot(hBlock)
	if err != nil {
		t.Fatal(err)
	}
	st := testutil.NewBeaconState()
	if err := service.beaconDB.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: fBlock}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, st.Copy(), fRoot); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Root: fRoot[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: hBlock}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, st.Copy(), headRoot); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveHeadBlockRoot(ctx, headRoot); err != nil {
		t.Fatal(err)
	}

	roots := [][32]byte{{'C'}, {'D'}, headRoot, {'E'}, fRoot, {'F'}}
	wanted := [][32]byte{{'C'}, {'D'}, {'E'}, {'F'}}

	received, err := service.filterBlockRoots(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(wanted, received) {
		t.Error("Did not filter correctly")
	}
}

func TestPersistCache_CanSave(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	st := testutil.NewBeaconState()

	for i := uint64(0); i < initialSyncCacheSize; i++ {
		if err := st.SetSlot(i); err != nil {
			t.Fatal(err)
		}
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		service.initSyncState[root] = st.Copy()
		service.boundaryRoots = append(service.boundaryRoots, root)
	}

	if err = service.persistCachedStates(ctx, initialSyncCacheSize); err != nil {
		t.Fatal(err)
	}

	for i := uint64(0); i < initialSyncCacheSize-minimumCacheSize; i++ {
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		state, err := db.State(context.Background(), root)
		if err != nil {
			t.Errorf("State with root of %#x , could not be retrieved: %v", root, err)
		}
		if state == nil {
			t.Errorf("State with root of %#x , does not exist", root)
		}
		if state.Slot() != i {
			t.Errorf("Incorrect slot retrieved. Wanted %d but got %d", i, state.Slot())
		}
	}
}

func TestFillForkChoiceMissingBlocks_CanSave(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	service.forkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	service.finalizedCheckpt = &ethpb.Checkpoint{}

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Error(err)
	}
	validGenesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Error(err)
	}
	st := testutil.NewBeaconState()

	if err := service.beaconDB.SaveState(ctx, st.Copy(), validGenesisRoot); err != nil {
		t.Fatal(err)
	}
	roots, err := blockTree1(db, validGenesisRoot[:])
	if err != nil {
		t.Fatal(err)
	}

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	block := &ethpb.BeaconBlock{Slot: 9, ParentRoot: roots[8]}
	if err := service.fillInForkChoiceMissingBlocks(context.Background(), block, beaconState); err != nil {
		t.Fatal(err)
	}

	// 5 nodes from the block tree 1. B0 - B3 - B4 - B6 - B8
	if len(service.forkChoiceStore.Nodes()) != 5 {
		t.Error("Miss match nodes")
	}

	if !service.forkChoiceStore.HasNode(bytesutil.ToBytes32(roots[4])) {
		t.Error("Didn't save node")
	}
	if !service.forkChoiceStore.HasNode(bytesutil.ToBytes32(roots[6])) {
		t.Error("Didn't save node")
	}
	if !service.forkChoiceStore.HasNode(bytesutil.ToBytes32(roots[8])) {
		t.Error("Didn't save node")
	}
}

func TestFillForkChoiceMissingBlocks_FilterFinalized(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	service.forkChoiceStore = protoarray.New(0, 0, [32]byte{'A'})
	// Set finalized epoch to 1.
	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1}

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Error(err)
	}
	validGenesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Error(err)
	}
	st := testutil.NewBeaconState()

	if err := service.beaconDB.SaveState(ctx, st.Copy(), validGenesisRoot); err != nil {
		t.Fatal(err)
	}

	// Define a tree branch, slot 63 <- 64 <- 65
	b63 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 63}}
	if err := service.beaconDB.SaveBlock(ctx, b63); err != nil {
		t.Fatal(err)
	}
	r63, err := stateutil.BlockRoot(b63.Block)
	if err != nil {
		t.Fatal(err)
	}
	b64 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 64, ParentRoot: r63[:]}}
	if err := service.beaconDB.SaveBlock(ctx, b64); err != nil {
		t.Fatal(err)
	}
	r64, err := stateutil.BlockRoot(b64.Block)
	if err != nil {
		t.Fatal(err)
	}
	b65 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 65, ParentRoot: r64[:]}}
	if err := service.beaconDB.SaveBlock(ctx, b65); err != nil {
		t.Fatal(err)
	}

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.fillInForkChoiceMissingBlocks(context.Background(), b65.Block, beaconState); err != nil {
		t.Fatal(err)
	}

	// There should be 2 nodes, block 65 and block 64.
	if len(service.forkChoiceStore.Nodes()) != 2 {
		t.Error("Miss match nodes")
	}

	// Block with slot 63 should be in fork choice because it's less than finalized epoch 1.
	if !service.forkChoiceStore.HasNode(r63) {
		t.Error("Didn't save node")
	}
}

// blockTree1 constructs the following tree:
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree1(db db.Database, genesisRoot []byte) ([][]byte, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, err := ssz.HashTreeRoot(b0)
	if err != nil {
		return nil, err
	}
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, err := ssz.HashTreeRoot(b1)
	if err != nil {
		return nil, err
	}
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r0[:]}
	r3, err := ssz.HashTreeRoot(b3)
	if err != nil {
		return nil, err
	}
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r3[:]}
	r4, err := ssz.HashTreeRoot(b4)
	if err != nil {
		return nil, err
	}
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r4[:]}
	r5, err := ssz.HashTreeRoot(b5)
	if err != nil {
		return nil, err
	}
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, err := ssz.HashTreeRoot(b6)
	if err != nil {
		return nil, err
	}
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r5[:]}
	r7, err := ssz.HashTreeRoot(b7)
	if err != nil {
		return nil, err
	}
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, err := ssz.HashTreeRoot(b8)
	if err != nil {
		return nil, err
	}
	st := testutil.NewBeaconState()

	for _, b := range []*ethpb.BeaconBlock{b0, b1, b3, b4, b5, b6, b7, b8} {
		if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b}); err != nil {
			return nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(b.ParentRoot)); err != nil {
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

	if slot != 0 {
		t.Fatalf("Expected slot to be 0, got %d", slot)
	}
}
