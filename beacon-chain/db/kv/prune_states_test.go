package kv

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// Sanity check that states are pruned
func TestStore_PruneStates(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	numBlocks := 33
	blocks := make([]*ethpb.BeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	for i := 0; i < len(blocks); i++ {
		blocks[i] = &ethpb.BeaconBlock{
			Slot: uint64(i),
		}
		r, err := ssz.SigningRoot(blocks[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := db.SaveState(ctx, &pb.BeaconState{Slot: uint64(i)}, r); err != nil {
			t.Fatal(err)
		}
		if err := db.SaveBlock(ctx, blocks[i]); err != nil {
			t.Fatal(err)
		}
		blockRoots = append(blockRoots, r)
	}
	db.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Epoch: 1, Root: blockRoots[numBlocks-1][:]})

	path := db.databasePath
	db.Close()

	c := featureconfig.Get()
	c.PruneEpochBoundaryStates = true
	featureconfig.Init(c)

	db2, err := NewKVStore(path)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	defer teardownDB(t, db2)

	s, err := db2.State(ctx, blockRoots[31])
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Error("finalized state should not be deleted")
	}

	s, err = db2.State(ctx, blockRoots[30])
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Error("regular state should be deleted")
	}
}
