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

	if db.LastArchivedRoot(ctx) != [32]byte{'A'} {
		t.Error("Did not get wanted root")
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
