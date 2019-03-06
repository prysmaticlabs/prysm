package db

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestVerifyContractAddress(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	ctx := context.Background()

	address := common.HexToAddress("0x0cd549b4abcbc0cb63012ea7de6fd34ebdccfd45")
	// There should be no error the first time.
	if err := db.VerifyContractAddress(ctx, address); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// There should be no error the second time.
	if err := db.VerifyContractAddress(ctx, address); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// But there should be an error with a different address.
	otherAddr := common.HexToAddress("0x247b06d9890ab9b032ec318ca436aef262d0f08a")
	if err := db.VerifyContractAddress(ctx, otherAddr); err == nil {
		t.Fatal("Expected error, but didn't receive one")
	}

}
