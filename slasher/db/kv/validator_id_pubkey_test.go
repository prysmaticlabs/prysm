package kv

import (
	"bytes"
	"context"
	"flag"
	"testing"

	"github.com/urfave/cli"
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
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	validatorID := uint64(1)

	pk, err := db.ValidatorPubKey(ctx, validatorID)
	if err != nil {
		t.Fatal("nil ValidatorPubKey should not return error")
	}
	if pk != nil {
		t.Fatal("ValidatorPubKey should return nil")
	}

}

func TestSavePubKey(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range pkTests {
		err := db.SavePubKey(ctx, tt.validatorID, tt.pk)
		if err != nil {
			t.Fatalf("save validator public key failed: %v", err)
		}

		pk, err := db.ValidatorPubKey(ctx, tt.validatorID)
		if err != nil {
			t.Fatalf("failed to get validator public key: %v", err)
		}

		if pk == nil || !bytes.Equal(pk, tt.pk) {
			t.Fatalf("get should return validator public key: %v", tt.pk)
		}
	}

}

func TestDeletePublicKey(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range pkTests {

		err := db.SavePubKey(ctx, tt.validatorID, tt.pk)
		if err != nil {
			t.Fatalf("save validator public key failed: %v", err)
		}
	}

	for _, tt := range pkTests {
		pk, err := db.ValidatorPubKey(ctx, tt.validatorID)
		if err != nil {
			t.Fatalf("failed to get validator public key: %v", err)
		}

		if pk == nil || !bytes.Equal(pk, tt.pk) {
			t.Fatalf("get should return validator public key: %v", pk)
		}
		err = db.DeletePubKey(ctx, tt.validatorID)
		if err != nil {
			t.Fatalf("delete validator public key: %v", err)
		}
		pk, err = db.ValidatorPubKey(ctx, tt.validatorID)
		if err != nil {
			t.Fatal(err)
		}
		if pk != nil {
			t.Errorf("Expected validator public key to be deleted, received: %v", pk)
		}

	}

}
