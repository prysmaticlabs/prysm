package db

import (
	"testing"
)

func TestSaveAndGetSlot(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	retSlot, err := db.GetSimulatorSlot()

	if err != nil {
		t.Fatalf("get slot failed: %v", err)
	}
	if retSlot != 0 {
		t.Fatalf("empty db does not have a simulator slot of 0")
	}

	slot := uint64(10)

	err = db.SaveSimulatorSlot(slot)
	if err != nil {
		t.Fatalf("save slot failed: %v", err)
	}

	retSlot, err = db.GetSimulatorSlot()

	if err != nil {
		t.Fatalf("get slot failed: %v", err)
	}

	if retSlot != slot {
		t.Errorf("retrieved slot not the same as the one saved to disk %d : %d", retSlot, slot)
	}
}
