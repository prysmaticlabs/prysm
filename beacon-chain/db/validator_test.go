package db

import (
	"fmt"
	"strings"
	"testing"
)

func TestSaveAndRetrieveValidatorIndex_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	p1 := []byte{'A', 'B', 'C'}
	p2 := []byte{'D', 'E', 'F'}

	if err := db.SaveValidatorIndex(p1, 1); err != nil {
		t.Fatalf("Failed to save vallidator index: %v", err)
	}
	if err := db.SaveValidatorIndex(p2, 2); err != nil {
		t.Fatalf("Failed to save vallidator index: %v", err)
	}

	index1, err := db.ValidatorIndex(p1)
	if err != nil {
		t.Fatalf("Failed to call Attestation: %v", err)
	}
	if index1 != 1 {
		t.Fatalf("Saved index and retrieved index are not equal: %#x and %#x", 1, index1)
	}

	index2, err := db.ValidatorIndex(p2)
	if err != nil {
		t.Fatalf("Failed to call Attestation: %v", err)
	}
	if index2 != 2 {
		t.Fatalf("Saved index and retrieved index are not equal: %#x and %#x", 2, index2)
	}
}

func TestSaveAndDeleteValidatorIndex_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	p1 := []byte{'1', '2', '3'}

	if err := db.SaveValidatorIndex(p1, 3); err != nil {
		t.Fatalf("Failed to save vallidator index: %v", err)
	}
	index, err := db.ValidatorIndex(p1)
	if err != nil {
		t.Fatalf("Failed to call Attestation: %v", err)
	}
	if index != 3 {
		t.Fatalf("Saved index and retrieved index are not equal: %#x and %#x", 3, index)
	}

	if err := db.DeleteValidatorIndex(p1); err != nil {
		t.Fatalf("Could not delete attestation: %v", err)
	}
	_, err = db.ValidatorIndex(p1)
	want := fmt.Sprintf("validator %#x does not exist", p1)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Want: %v, got: %v", want, err.Error())
	}
}
