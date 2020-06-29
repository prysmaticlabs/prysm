package kv

import (
	"context"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStore_Backup(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	head := &eth.SignedBeaconBlock{Block: &eth.BeaconBlock{Slot: 5000}}

	if err := db.SaveBlock(ctx, head); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(head.Block)
	if err != nil {
		t.Fatal(err)
	}
	st := testutil.NewBeaconState()
	if err := db.SaveState(ctx, st, root); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, root); err != nil {
		t.Fatal(err)
	}

	if err := db.Backup(ctx); err != nil {
		t.Fatal(err)
	}

	dataDirEndIndex := strings.LastIndex(db.databasePath, "/")
	files, err := ioutil.ReadDir(path.Join(db.databasePath[:dataDirEndIndex], backupsDirectoryName))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("No backups created.")
	}
}
