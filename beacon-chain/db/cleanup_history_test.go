package db

import (
	"testing"
)

func TestSaveCleanedFinalizedSlot(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	slot := uint64(100)
	if err := db.SaveCleanedFinalizedSlot(slot); err != nil {
		t.Fatalf("failed to save cleaned finalized slot %v", err)
	}
}

func TestGetCleanedFinalizedSlot_NotFound(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	_, err := db.GetCleanedFinalizedSlot()
	if err == nil {
		t.Fatalf("should expect error if last finalized slot not found in DB")
	}
}

func TestGetCleanedFinalizedSlot(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	slot := uint64(100)
	if err := db.SaveCleanedFinalizedSlot(slot); err != nil {
		t.Fatalf("failed to save cleaned finalized slot %v", err)
	}

	readSlot, err := db.GetCleanedFinalizedSlot()
	if err != nil {
		t.Fatalf("failed to read cleaned finalized slot from DB %v", err)
	}
	if readSlot != slot {
		t.Fatalf("got wrong result when reading cleaned finalized slot from DB")
	}
}
