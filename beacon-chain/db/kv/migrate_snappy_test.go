package kv

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// Sanity check that an object can be accessed after migration.
func TestStore_MigrateSnappy(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	block := &ethpb.BeaconBlock{
		Slot: 200,
	}
	root, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	path := db.databasePath
	db.Close()

	c := featureconfig.Get()
	c.EnableSnappyDBCompression = true
	featureconfig.Init(c)

	db2, err := NewKVStore(path)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	defer teardownDB(t, db2)

	blk, err := db.Block(ctx, root)
	if err != nil {
		t.Fatal(err)
	}

	if !ssz.DeepEqual(blk, block) {
		t.Fatal("Blocks not same")
	}
}
