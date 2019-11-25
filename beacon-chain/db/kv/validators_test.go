package kv

import (
	"context"
	"testing"
)

func TestStore_ValidatorIndexCRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	validatorIdx := uint64(100)
	pubKey := [48]byte{1, 2, 3, 4}
	ctx := context.Background()
	_, ok, err := db.ValidatorIndex(ctx, pubKey)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("Expected validator index to not exist")
	}
	if err := db.SaveValidatorIndex(ctx, pubKey, validatorIdx); err != nil {
		t.Fatal(err)
	}
	retrievedIdx, ok, err := db.ValidatorIndex(ctx, pubKey)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Expected validator index to have been properly retrieved")
	}
	if retrievedIdx != validatorIdx {
		t.Errorf("Wanted %d, received %d", validatorIdx, retrievedIdx)
	}
	if err := db.DeleteValidatorIndex(ctx, pubKey); err != nil {
		t.Fatal(err)
	}
	if db.HasValidatorIndex(ctx, pubKey) {
		t.Error("Expected validator index to have been deleted from the db")
	}
}
