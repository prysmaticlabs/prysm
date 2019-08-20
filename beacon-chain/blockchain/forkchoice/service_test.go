package forkchoice

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestStore_GensisStoreOk(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	genesisTime := time.Unix(9999, 0)
	genesisState := &pb.BeaconState{GenesisTime: uint64(genesisTime.Unix())}
	genesisStateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		t.Fatal(err)
	}

	genesisBlk := blocks.NewGenesisBlock(genesisStateRoot[:])
	genesisBlkRoot, err := ssz.SigningRoot(genesisBlk)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.GenesisStore(ctx, genesisState); err != nil {
		t.Fatal(err)
	}

	genesisCheckpt := &ethpb.Checkpoint{Epoch: 0, Root: genesisBlkRoot[:]}
	if !reflect.DeepEqual(store.justifiedCheckpt, genesisCheckpt) {
		t.Error("Justified check point from genesis store did not match")
	}
	if !reflect.DeepEqual(store.finalizedCheckpt, genesisCheckpt) {
		t.Error("Finalized check point from genesis store did not match")
	}

	b, err := store.db.Block(ctx, genesisBlkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, genesisBlk) {
		t.Error("Incorrect genesis block saved from store")
	}

	h, err := hashutil.HashProto(genesisCheckpt)
	if err != nil {
		t.Fatal(err)
	}
	if store.checkptBlkRoot[h] != genesisBlkRoot {
		t.Error("Incorrect genesis check point to block root saved from store")
	}
}

func TestStore_AncestorOk(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
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
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
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
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}

	if err := store.GenesisStore(ctx, s); err != nil {
		t.Fatal(err)
	}

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
				t.Fatal(err)
			}
		case i > 66:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
				t.Fatal(err)
			}
		default:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[8]}); err != nil {
				t.Fatal(err)
			}
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
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	filter := filters.NewFilter().SetParentRoot(roots[0]).SetStartSlot(0)
	children, err := store.db.BlockRoots(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][]byte{roots[1], roots[3]}) {
		t.Error("Did not receive correct children roots")
	}

	filter = filters.NewFilter().SetParentRoot(roots[0]).SetStartSlot(2)
	children, err = store.db.BlockRoots(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(children, [][]byte{roots[3]}) {
		t.Error("Did not receive correct children roots")
	}
}

func TestStore_GetHead(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	store := NewForkChoiceService(ctx, kv)

	roots, err := blockTree1(db)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{ExitEpoch: 2, EffectiveBalance: 1e9}
	}

	s := &pb.BeaconState{Validators: validators}
	if err := store.GenesisStore(ctx, s); err != nil {
		t.Fatal(err)
	}
	if err := store.db.SaveState(ctx, s, bytesutil.ToBytes32(roots[0])); err != nil {
		t.Fatal(err)
	}
	store.justifiedCheckpt.Root = roots[0]
	h, err := hashutil.HashProto(store.justifiedCheckpt)
	if err != nil {
		t.Fatal(err)
	}
	store.checkptBlkRoot[h] = bytesutil.ToBytes32(roots[0])

	//    /- B1 (33 votes)
	// B0           /- B5 - B7 (33 votes)
	//    \- B3 - B4 - B6 - B8 (34 votes)
	for i := 0; i < len(validators); i++ {
		switch {
		case i < 33:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
				t.Fatal(err)
			}
		case i > 66:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
				t.Fatal(err)
			}
		default:
			if err := store.db.SaveValidatorLatestVote(ctx, uint64(i), &pb.ValidatorLatestVote{Root: roots[8]}); err != nil {
				t.Fatal(err)
			}
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
	if err := store.db.SaveValidatorLatestVote(ctx, 50, &pb.ValidatorLatestVote{Root: roots[7]}); err != nil {
		t.Fatal(err)
	}
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
		if err := store.db.SaveValidatorLatestVote(ctx, idx, &pb.ValidatorLatestVote{Root: roots[1]}); err != nil {
			t.Fatal(err)
		}
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
