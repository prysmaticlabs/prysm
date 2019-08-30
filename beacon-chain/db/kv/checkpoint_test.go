package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestStore_JustifiedCheckpoint_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	cp := &ethpb.Checkpoint{
		Epoch: 10,
		Root:  []byte{'A'},
	}

	retrieved, err := db.JustifiedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil check point, received %v", retrieved)
	}
	if err := db.SaveJustifiedCheckpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}

	retrieved, err = db.JustifiedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(cp, retrieved) {
		t.Errorf("Wanted %v, received %v", cp, retrieved)
	}
}

func TestStore_FinalizedCheckpoint_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	cp := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  []byte{'B'},
	}

	retrieved, err := db.FinalizedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil check point, received %v", retrieved)
	}
	if err := db.SaveFinalizedCheckpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}

	retrieved, err = db.FinalizedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(cp, retrieved) {
		t.Errorf("Wanted %v, received %v", cp, retrieved)
	}
}
