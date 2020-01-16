package forkchoice

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
)

func TestStore_GenesisStoreOk(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	genesisTime := time.Unix(9999, 0)
	genesisState := &pb.BeaconState{GenesisTime: uint64(genesisTime.Unix())}
	genesisStateRoot, err := stateutil.HashTreeRootState(genesisState)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlk := blocks.NewGenesisBlock(genesisStateRoot[:])
	genesisBlkRoot, err := ssz.HashTreeRoot(genesisBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		t.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{Root: genesisBlkRoot[:]}
	if err := store.GenesisStore(ctx, checkPoint, checkPoint); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(store.justifiedCheckpt, checkPoint) {
		t.Error("Justified check point from genesis store did not match")
	}
	if !reflect.DeepEqual(store.finalizedCheckpt, checkPoint) {
		t.Error("Finalized check point from genesis store did not match")
	}

	cachedState, err := store.checkpointState.StateByCheckpoint(checkPoint)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cachedState, genesisState) {
		t.Error("Incorrect genesis state cached")
	}
}

func TestStore_AncestorOk(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		root []byte
		slot uint64
	}

	//    /- B1
	// B0           /- B5 - B7
	//    \- B3 - B4 - B6 - B8
	tests := []struct {
		args *args
		want []byte
	}{
		{args: &args{roots[1], 0}, want: roots[0]},
		{args: &args{roots[8], 0}, want: roots[0]},
		{args: &args{roots[8], 4}, want: roots[4]},
		{args: &args{roots[7], 4}, want: roots[4]},
		{args: &args{roots[7], 0}, want: roots[0]},
	}
	for _, tt := range tests {
		got, err := store.ancestor(ctx, tt.args.root, tt.args.slot)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Store.ancestor(ctx, ) = %v, want %v", got, tt.want)
		}
	}
}

func TestStore_AncestorNotPartOfTheChain(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	//    /- B1
	// B0           /- B5 - B7
	//    \- B3 - B4 - B6 - B8
	root, err := store.ancestor(ctx, roots[8], 1)
	if err != nil {
		t.Fatal(err)
	}
	if root != nil {
		t.Error("block at slot 1 is not part of the chain")
	}
	root, err = store.ancestor(ctx, roots[8], 2)
	if err != nil {
		t.Fatal(err)
	}
	if root != nil {
		t.Error("block at slot 2 is not part of the chain")
	}
}

func TestStore_LatestAttestingBalance(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators, RandaoMixes: make([]bytesutil.Bytes32Array, params.BeaconConfig().EpochsPerHistoricalVector)}
	stateRoot, err := stateutil.HashTreeRootState(s)
	if err != nil {
		t.Fatal(err)
	}
	b := blocks.NewGenesisBlock(stateRoot[:])
	blkRoot, err := ssz.HashTreeRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, s, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{Root: blkRoot[:]}
	if err := store.GenesisStore(ctx, checkPoint, checkPoint); err != nil {
		t.Fatal(err)
	}

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			store.latestVoteMap[uint64(i)] = &pb.ValidatorLatestVote{Root: roots[1]}
		case i > 66:
			store.latestVoteMap[uint64(i)] = &pb.ValidatorLatestVote{Root: roots[7]}
		default:
			store.latestVoteMap[uint64(i)] = &pb.ValidatorLatestVote{Root: roots[8]}
		}
	}

	tests := []struct {
		root []byte
		want uint64
	}{
		{root: roots[0], want: 100 * 1e9},
		{root: roots[1], want: 33 * 1e9},
		{root: roots[3], want: 67 * 1e9},
		{root: roots[4], want: 67 * 1e9},
		{root: roots[7], want: 33 * 1e9},
		{root: roots[8], want: 34 * 1e9},
	}
	for _, tt := range tests {
		got, err := store.latestAttestingBalance(ctx, tt.root)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("Store.latestAttestingBalance(ctx, ) = %v, want %v", got, tt.want)
		}
	}
}

func TestStore_ChildrenBlocksFromParentRoot(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	filter := filters.NewFilter().SetParentRoot(roots[0]).SetStartSlot(0)
	children, err := store.db.BlockRoots(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][32]byte{bytesutil.ToBytes32(roots[1]), bytesutil.ToBytes32(roots[3])}) {
		t.Error("Did not receive correct children roots")
	}

	filter = filters.NewFilter().SetParentRoot(roots[0]).SetStartSlot(2)
	children, err = store.db.BlockRoots(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][32]byte{bytesutil.ToBytes32(roots[3])}) {
		t.Error("Did not receive correct children roots")
	}
}

func TestStore_GetHead(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators, RandaoMixes: make([]bytesutil.Bytes32Array, params.BeaconConfig().EpochsPerHistoricalVector)}
	stateRoot, err := stateutil.HashTreeRootState(s)
	if err != nil {
		t.Fatal(err)
	}
	b := blocks.NewGenesisBlock(stateRoot[:])
	blkRoot, err := ssz.HashTreeRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{Root: blkRoot[:]}

	if err := store.GenesisStore(ctx, checkPoint, checkPoint); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		t.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: store.justifiedCheckpt,
		State:      s,
	}); err != nil {
		t.Fatal(err)
	}

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			store.latestVoteMap[uint64(i)] = &pb.ValidatorLatestVote{Root: roots[1]}
		case i > 66:
			store.latestVoteMap[uint64(i)] = &pb.ValidatorLatestVote{Root: roots[7]}
		default:
			store.latestVoteMap[uint64(i)] = &pb.ValidatorLatestVote{Root: roots[8]}
		}
	}

	// Default head is B8
	head, err := store.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, roots[8]) {
		t.Error("Incorrect head")
	}

	// 1 validator switches vote to B7 to gain 34%, enough to switch head
	store.latestVoteMap[uint64(50)] = &pb.ValidatorLatestVote{Root: roots[7]}

	head, err = store.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, roots[7]) {
		t.Error("Incorrect head")
	}

	// 18 validators switches vote to B1 to gain 51%, enough to switch head
	for i := 0; i < 18; i++ {
		idx := 50 + uint64(i)
		store.latestVoteMap[idx] = &pb.ValidatorLatestVote{Root: roots[1]}
	}
	head, err = store.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(head, roots[1]) {
		t.Log(head)
		t.Error("Incorrect head")
	}
}

func TestCacheGenesisState_Correct(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)
	config := &featureconfig.Flags{
		InitSyncCacheState: true,
	}
	featureconfig.Init(config)

	b := &ethpb.BeaconBlock{Slot: 1}
	r, _ := ssz.HashTreeRoot(b)
	s := &pb.BeaconState{GenesisTime: 99}

	store.db.SaveState(ctx, s, r)
	store.db.SaveGenesisBlockRoot(ctx, r)

	if err := store.cacheGenesisState(ctx); err != nil {
		t.Fatal(err)
	}

	for _, state := range store.initSyncState {
		if !reflect.DeepEqual(s, state) {
			t.Error("Did not get wanted state")
		}
	}
}

func TestStore_GetFilterBlockTree_CorrectLeaf(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	s := &pb.BeaconState{}
	stateRoot, err := stateutil.HashTreeRootState(s)
	if err != nil {
		t.Fatal(err)
	}
	b := blocks.NewGenesisBlock(stateRoot[:])
	blkRoot, err := ssz.HashTreeRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{Root: blkRoot[:]}

	if err := store.GenesisStore(ctx, checkPoint, checkPoint); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		t.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: store.justifiedCheckpt,
		State:      s,
	}); err != nil {
		t.Fatal(err)
	}

	tree, err := store.getFilterBlockTree(ctx)
	if err != nil {
		t.Fatal(err)
	}

	wanted := make(map[[32]byte]*ethpb.BeaconBlock)
	for _, root := range roots {
		root32 := bytesutil.ToBytes32(root)
		b, _ := store.db.Block(ctx, root32)
		if b != nil {
			wanted[root32] = b.Block
		}
	}
	if !reflect.DeepEqual(tree, wanted) {
		t.Error("Did not filter tree correctly")
	}
}

func TestStore_GetFilterBlockTree_IncorrectLeaf(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	s := &pb.BeaconState{}
	stateRoot, err := stateutil.HashTreeRootState(s)
	if err != nil {
		t.Fatal(err)
	}
	b := blocks.NewGenesisBlock(stateRoot[:])
	blkRoot, err := ssz.HashTreeRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}

	checkPoint := &ethpb.Checkpoint{Root: blkRoot[:]}

	if err := store.GenesisStore(ctx, checkPoint, checkPoint); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		t.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	if err := store.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: store.justifiedCheckpt,
		State:      s,
	}); err != nil {
		t.Fatal(err)
	}
	// Filter for incorrect leaves for 1, 7 and 8
	store.db.SaveState(ctx, &pb.BeaconState{CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}, bytesutil.ToBytes32(roots[1]))
	store.db.SaveState(ctx, &pb.BeaconState{CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}, bytesutil.ToBytes32(roots[7]))
	store.db.SaveState(ctx, &pb.BeaconState{CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}, bytesutil.ToBytes32(roots[8]))
	store.justifiedCheckpt.Epoch = 1
	tree, err := store.getFilterBlockTree(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 0 {
		t.Error("filtered tree should be 0 length")
	}

	// Set leave 1 as correct
	store.db.SaveState(ctx, &pb.BeaconState{CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 1, Root: store.justifiedCheckpt.Root}}, bytesutil.ToBytes32(roots[1]))
	tree, err = store.getFilterBlockTree(ctx)
	if err != nil {
		t.Fatal(err)
	}

	wanted := make(map[[32]byte]*ethpb.BeaconBlock)
	root32 := bytesutil.ToBytes32(roots[0])
	b, err = store.db.Block(ctx, root32)
	if err != nil {
		t.Fatal(err)
	}
	wanted[root32] = b.Block
	root32 = bytesutil.ToBytes32(roots[1])
	b, err = store.db.Block(ctx, root32)
	if err != nil {
		t.Fatal(err)
	}
	wanted[root32] = b.Block

	if !reflect.DeepEqual(tree, wanted) {
		t.Error("Did not filter tree correctly")
	}
}
