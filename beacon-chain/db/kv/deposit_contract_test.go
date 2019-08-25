package kv

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestStore_DepositContract(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	contractAddress := common.Address{1, 2, 3}
	retrieved, err := db.DepositContractAddress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil contract address, received %v", retrieved)
	}
	if err := db.SaveDepositContractAddress(ctx, contractAddress); err != nil {
		t.Fatal(err)
	}
	retrieved, err = db.DepositContractAddress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if common.BytesToAddress(retrieved) != contractAddress {
		t.Errorf("Expected address %#x, received %#x", contractAddress, retrieved)
	}
	otherAddress := common.Address{4, 5, 6}
	if err := db.SaveDepositContractAddress(ctx, otherAddress); err == nil {
		t.Error("Should not have been able to override old deposit contract address")
	}
}
