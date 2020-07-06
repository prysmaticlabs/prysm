package kv

import (
	"context"
	"testing"
)

func TestArchivedPointIndexRoot_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
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

func TestLastArchivedPoint_CanRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	i, err := db.LastArchivedSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Error("Did not get correct index")
	}

	if err := db.SaveArchivedPointRoot(ctx, [32]byte{'A'}, 1); err != nil {
		t.Fatal(err)
	}

	if db.LastArchivedRoot(ctx) != [32]byte{'A'} {
		t.Error("Did not get wanted root")
	}

	if err := db.SaveArchivedPointRoot(ctx, [32]byte{'B'}, 3); err != nil {
		t.Fatal(err)
	}

	if db.LastArchivedRoot(ctx) != [32]byte{'B'} {
		t.Error("Did not get wanted root")
	}

	i, err = db.LastArchivedSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if i != 3 {
		t.Error("Did not get correct index")
	}
}
