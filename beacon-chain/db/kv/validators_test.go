package kv

import (
	"context"
	"strconv"
	"testing"
)

func TestStore_ValidatorIndexCRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	validatorIdx := uint64(100)
	pubKey := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}
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

func TestStore_SaveValidatorIndices(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	numVals := 10
	indices := make([]uint64, numVals)
	keys := make([][48]byte, numVals)
	for i := 0; i < numVals; i++ {
		indices[i] = uint64(i)
		pub := [48]byte{}
		copy(pub[:], strconv.Itoa(i))
		keys[i] = pub
	}
	ctx := context.Background()
	if err := db.SaveValidatorIndices(ctx, keys, indices); err != nil {
		t.Error(err)
	}
	if err := db.SaveValidatorIndices(ctx, keys[:len(keys)-1], indices); err == nil {
		t.Error("Expected error when saving different number of keys and indices, received nil")
	}
	for i := 0; i < numVals; i++ {
		if !db.HasValidatorIndex(ctx, keys[i][:]) {
			t.Errorf("Expected validator index %d to have been saved to the db", i)
		}
	}
}
