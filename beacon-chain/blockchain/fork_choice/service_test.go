package forkchoice

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// blockTree constructs the following tree
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree(db *db.BeaconDB) ([][]byte, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0}
	r0, _ := ssz.SigningRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.SigningRoot(b1)
	b3 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r3, _ := ssz.SigningRoot(b3)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r3[:]}
	r4, _ := ssz.SigningRoot(b4)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r4[:]}
	r5, _ := ssz.SigningRoot(b5)
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, _ := ssz.SigningRoot(b6)
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r5[:]}
	r7, _ := ssz.SigningRoot(b7)
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, _ := ssz.SigningRoot(b8)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b4, b5, b6, b7, b8} {
		if err := db.SaveBlock(b); err != nil {
			return nil, err
		}
	}
	return [][]byte{r0[:],r1[:],r3[:],nil,r4[:],r5[:],r6[:],r7[:],r8[:]}, nil
}

func TestStore_GensisStoreOk(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	genesisTime := uint64(9999)
	genesisState := &pb.BeaconState{GenesisTime: genesisTime}
	genesisStateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		t.Fatal(err)
	}

	genesisBlk := &ethpb.BeaconBlock{StateRoot: genesisStateRoot[:]}
	genesisBlkRoot, err := ssz.SigningRoot(genesisBlk)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.GensisStore(genesisState); err != nil {
		t.Fatal(err)
	}

	if store.time != genesisTime {
		t.Errorf("Wanted time %d, got time %d", genesisTime, store.time)
	}

	genesisCheckpt := &ethpb.Checkpoint{Epoch: 0, Root: genesisBlkRoot[:]}
	if !reflect.DeepEqual(store.justifiedCheckpt, genesisCheckpt) {
		t.Error("Justified check point from genesis store did not match")
	}
	if !reflect.DeepEqual(store.finalizedCheckpt, genesisCheckpt) {
		t.Error("Finalized check point from genesis store did not match")
	}

	b, err := store.db.Block(genesisBlkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, genesisBlk) {
		t.Error("Incorrect genesis block saved from store")
	}

	s, err := store.db.CheckpointState(ctx, genesisCheckpt)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, genesisState) {
		t.Error("Incorrect genesis state saved from store")
	}
}

func TestStore_AncestorOk(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db);
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
		args    *args
		want    []byte
	}{
		{args: &args{roots[1], 0}, want:roots[0]},
		{args: &args{roots[8], 0}, want:roots[0]},
		{args: &args{roots[8], 4}, want:roots[4]},
		{args: &args{roots[7], 4}, want:roots[4]},
		{args: &args{roots[7], 0}, want:roots[0]},
	}
	for _, tt := range tests {
			got, err := store.Ancestor(tt.args.root, tt.args.slot)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Store.Ancestor() = %v, want %v", got, tt.want)
			}
	}
}

func TestStore_AncestorOutOfRange(t *testing.T) {
	ctx := context.Background()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	store := NewForkChoiceService(ctx, db)

	roots, err := blockTree(db);
	if err != nil {
		t.Fatal(err)
	}

	//    /- B1
	// B0           /- B5 - B7
	//    \- B3 - B4 - B6 - B8
	wanted := fmt.Sprintf("could not retrieve ancestor for slot %d", 2)
	if _, err := store.Ancestor(roots[8], 2); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}
