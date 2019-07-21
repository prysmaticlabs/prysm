package db

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestBeaconDB_HasExit(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	d := &ethpb.VoluntaryExit{
		Epoch: 100,
	}
	hash, err := hashutil.HashProto(d)
	if err != nil {
		t.Fatalf("could not hash exit request: %v", err)
	}

	if db.HasExit(hash) {
		t.Fatal("Expected HasExit to return false")
	}

	if err := db.SaveExit(context.Background(), d); err != nil {
		t.Fatalf("Failed to save exit request: %v", err)
	}
	if !db.HasExit(hash) {
		t.Fatal("Expected HasExit to return true")
	}
}
