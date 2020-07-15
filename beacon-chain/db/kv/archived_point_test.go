package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	st := testutil.NewBeaconState()
	if err := st.SetSlot(i1); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, st, r1); err != nil {
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

	st := testutil.NewBeaconState()
	if err := db.SaveState(ctx, st, [32]byte{'A'}); err != nil {
		t.Error(err)
	}

	if db.LastArchivedRoot(ctx) != [32]byte{'A'} {
		t.Error("Did not get wanted root")
	}

	if err := st.SetSlot(2); err != nil {
		t.Error(err)
	}

	if err := db.SaveState(ctx, st, [32]byte{'B'}); err != nil {
		t.Error(err)
	}

	if db.LastArchivedRoot(ctx) != [32]byte{'B'} {
		t.Error("Did not get wanted root")
	}

	if err := st.SetSlot(3); err != nil {
		t.Error(err)
	}

	if err := db.SaveState(ctx, st, [32]byte{'C'}); err != nil {
		t.Fatal(err)
	}

	i, err = db.LastArchivedSlot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if i != 3 {
		t.Error("Did not get correct index")
	}
}
