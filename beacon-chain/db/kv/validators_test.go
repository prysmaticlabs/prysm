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
	if err := db.SaveValidatorIndex(ctx, pubKey, validatorIdx); err != nil {
		t.Fatal(err)
	}
	retrievedIdx, err := db.ValidatorIndex(ctx, pubKey)
	if err != nil {
		t.Fatal(err)
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

func TestStore_HasValidatorIndex(t *testing.T) {
}

func TestStore_HasValidatorLatestVote(t *testing.T) {

}

func TestStore_SaveValidatorIndex(t *testing.T) {

}

func TestStore_SaveValidatorLatestVote(t *testing.T) {

}

func TestStore_ValidatorIndex(t *testing.T) {
}

func TestStore_ValidatorLatestVote(t *testing.T) {

}
