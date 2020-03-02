package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestArchivedPointIndexRoot_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	i1 := uint64(100)
	r1 := [32]byte{'A'}

	received := db.ArchivedPointRoot(ctx, i1)
	if r1 == received {
		t.Fatal("Should not have been saved")
	}

	if err := db.SaveArchivedPointRoot(ctx, r1, i1); err != nil {
		t.Fatal(err)
	}
	received = db.ArchivedPointRoot(ctx, i1)
	if r1 != received {
		t.Error("Should have been saved")
	}
}

func TestArchivedPointIndexState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	i1 := uint64(100)
	s := &pb.BeaconState{Slot: 100}
	st, err := state.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}
	received, err := db.ArchivedPointState(ctx, i1)
	if err != nil {
		t.Fatal(err)
	}
	if received != nil {
		t.Fatal("Should not have been saved")
	}

	if err := db.SaveArchivedPointState(ctx, st, i1); err != nil {
		t.Fatal(err)
	}
	received, err = db.ArchivedPointState(ctx, i1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(received, st) {
		t.Error("Should have been saved")
	}
}

func TestArchivedPointIndexHas_CanRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	i1 := uint64(100)
	s := &pb.BeaconState{Slot: 100}
	st, err := state.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}
	r1 := [32]byte{'A'}

	if db.HasArchivedPoint(ctx, i1) {
		t.Fatal("Should have have an archived point")
	}

	if err := db.SaveArchivedPointState(ctx, st, i1); err != nil {
		t.Fatal(err)
	}
	if db.HasArchivedPoint(ctx, i1) {
		t.Fatal("Should have have an archived point")
	}

	if err := db.SaveArchivedPointRoot(ctx, r1, i1); err != nil {
		t.Fatal(err)
	}
	if !db.HasArchivedPoint(ctx, i1) {
		t.Fatal("Should have an archived point")
	}
}
