package kv

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestStore_Backup(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	head := &eth.SignedBeaconBlock{Block: &eth.BeaconBlock{Slot: 5000}}

	if err := db.SaveBlock(ctx, head); err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(head.Block)
	if err != nil {
		t.Fatal(err)
	}
	st, err := state.InitializeFromProto(&pb.BeaconState{})
	if err := db.SaveState(ctx, st, root); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, root); err != nil {
		t.Fatal(err)
	}

	if err := db.Backup(ctx); err != nil {
		t.Fatal(err)
	}

	files, err := ioutil.ReadDir(path.Join(db.databasePath, backupsDirectoryName))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("No backups created.")
	}
}
