package db

import "testing"

func TestSlotInDB_CanSaveGet(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	slot := uint64(76456)
	err := db.UpdateSlot(slot)
	if err != nil {
		t.Fatalf("Save slot failed: %v", err)
	}
	receivedSlot, err := db.Slot()
	if err != nil {
		t.Fatalf("Could not get slot: %v", err)
	}
	if receivedSlot != slot {
		t.Errorf("Wanted: %d, got: %d", slot, receivedSlot)
	}

	slot = uint64(9999999)
	err = db.UpdateSlot(slot)
	if err != nil {
		t.Fatalf("Save slot failed: %v", err)
	}
	receivedSlot, err = db.Slot()
	if err != nil {
		t.Fatalf("Could not get slot: %v", err)
	}
	if receivedSlot != slot {
		t.Errorf("Wanted: %d, got: %d", slot, receivedSlot)
	}
}
