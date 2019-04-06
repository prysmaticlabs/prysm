package db

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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

func TestSaveAndRetrieveValidatorsIndexes_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	p1 := []byte{'A', 'B', 'C'}
	p2 := []byte{'D', 'E', 'F'}

	if err := db.SaveValidatorIndex(p1, 1); err != nil {
		t.Fatalf("Failed to save validator index: %v", err)
	}
	if err := db.SaveValidatorIndex(p2, 2); err != nil {
		t.Fatalf("Failed to save validator index: %v", err)
	}
	ids := [][]byte{}
	ids = append(ids, p1, p2)
	indexes, err := db.ValidatorIndices(ids)
	if err != nil {
		t.Fatalf("Failed to call Attestation: %v", err)
	}
	if bytes.Equal(indexes[0], p1) {
		t.Fatalf("Saved index and retrieved index are not equal: %#x and %#x", 1, indexes[0])
	}
	if bytes.Equal(indexes[1], p2) {
		t.Fatalf("Saved index and retrieved index are not equal: %#x and %#x", 2, indexes[1])
	}
	if err := db.DeleteValidatorIndex(p2); err != nil {
		t.Fatalf("Could not delete validator index: %v", err)
	}
	indexes, err = db.ValidatorIndices(ids)
	want := fmt.Sprintf("one or more of the validators %#x does not exist", ids)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Want: %v, got: %v", want, err.Error())
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

func TestHasValidator(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	pk := []byte("pk")

	// Populate the db with some public key
	if err := db.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)
		h := hashutil.Hash(pk)
		return bkt.Put(h[:], []byte("data"))
	}); err != nil {
		t.Fatal(err)
	}

	if !db.HasValidator(pk) {
		t.Error("Database did not have expected validator")
	}

	if db.HasValidator([]byte("bogus")) {
		t.Error("Database returned true for validator that did not exist")
	}
}

func TestHasAllValidators(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	knownPubKeys := [][]byte{
		[]byte("pk1"),
		[]byte("pk2"),
	}
	unknownPubKeys := [][]byte{
		[]byte("pk3"),
		[]byte("pk4"),
	}

	// Populate the db with some public key
	if err := db.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)
		for _, pk := range knownPubKeys {
			h := hashutil.Hash(pk)
			if err := bkt.Put(h[:], []byte("data")); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if !db.HasAllValidators(knownPubKeys) {
		t.Error("Database did not have expected validators")
	}

	if db.HasAllValidators(append(knownPubKeys, unknownPubKeys...)) {
		t.Error("Database returned true when there are pubkeys that did not exist")
	}
}

func TestHasAnyValidator(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	knownPubKeys := [][]byte{
		[]byte("pk1"),
		[]byte("pk2"),
	}
	unknownPubKeys := [][]byte{
		[]byte("pk3"),
		[]byte("pk4"),
	}

	// Populate the db with some public key
	if err := db.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)
		for _, pk := range knownPubKeys {
			h := hashutil.Hash(pk)
			if err := bkt.Put(h[:], []byte("data")); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if !db.HasAnyValidators(append(knownPubKeys, unknownPubKeys...)) {
		t.Error("Database did not have expected validators")
	}

	if db.HasAnyValidators(unknownPubKeys) {
		t.Error("Database returned true when there are only pubkeys that did not exist")
	}
}
