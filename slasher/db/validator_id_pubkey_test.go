package db

import (
	"bytes"
	"testing"
)

type publicKeyTestStruct struct {
	validatorID uint64
	pk          []byte
}

var pkTests []publicKeyTestStruct

func init() {
	pkTests = []publicKeyTestStruct{
		{
			validatorID: 1,
			pk:          []byte{1, 2, 3},
		},
		{
			validatorID: 2,
			pk:          []byte{4, 5, 6},
		},
		{
			validatorID: 3,
			pk:          []byte{7, 8, 9},
		},
	}
}

func TestNilDBValidatorPublicKey(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	validatorID := uint64(1)

	pk, err := db.ValidatorPubKey(validatorID)
	if err != nil {
		t.Fatal("nil ValidatorPubKey should not return error")
	}
	if pk != nil {
		t.Fatal("ValidatorPubKey should return nil")
	}

}

func TestSavePubKey(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range pkTests {
		err := db.SavePubKey(tt.validatorID, tt.pk)
		if err != nil {
			t.Fatalf("save validator public key failed: %v", err)
		}

		pk, err := db.ValidatorPubKey(tt.validatorID)
		if err != nil {
			t.Fatalf("failed to get validator public key: %v", err)
		}

		if pk == nil || !bytes.Equal(pk, tt.pk) {
			t.Fatalf("get should return validator public key: %v", tt.pk)
		}
	}

}

func TestDeletePublicKey(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range pkTests {

		err := db.SavePubKey(tt.validatorID, tt.pk)
		if err != nil {
			t.Fatalf("save validator public key failed: %v", err)
		}
	}

	for _, tt := range pkTests {
		pk, err := db.ValidatorPubKey(tt.validatorID)
		if err != nil {
			t.Fatalf("failed to get validator public key: %v", err)
		}

		if pk == nil || !bytes.Equal(pk, tt.pk) {
			t.Fatalf("get should return validator public key: %v", pk)
		}
		err = db.DeletePubKey(tt.validatorID)
		if err != nil {
			t.Fatalf("delete validator public key: %v", err)
		}
		pk, err = db.ValidatorPubKey(tt.validatorID)
		if err != nil {
			t.Fatal(err)
		}
		if pk != nil {
			t.Errorf("Expected validator public key to be deleted, received: %v", pk)
		}

	}

}
