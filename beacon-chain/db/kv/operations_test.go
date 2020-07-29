package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestStore_VoluntaryExits_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	exit := &ethpb.VoluntaryExit{
		Epoch: 5,
	}
	exitRoot, err := exit.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.VoluntaryExit(ctx, exitRoot)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil voluntary exit, received %v", retrieved)
	}
	if err := db.SaveVoluntaryExit(ctx, exit); err != nil {
		t.Fatal(err)
	}
	if !db.HasVoluntaryExit(ctx, exitRoot) {
		t.Error("Expected voluntary exit to exist in the db")
	}
	retrieved, err = db.VoluntaryExit(ctx, exitRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(exit, retrieved) {
		t.Errorf("Wanted %v, received %v", exit, retrieved)
	}
	if err := db.deleteVoluntaryExit(ctx, exitRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasVoluntaryExit(ctx, exitRoot) {
		t.Error("Expected voluntary exit to have been deleted from the db")
	}
}
