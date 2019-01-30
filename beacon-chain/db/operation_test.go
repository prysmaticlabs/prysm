package db

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestHasDeposit(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	d := &pb.Deposit{
		DepositData: []byte{'A'},
	}
	hash, err := hashutil.HashProto(d)
	if err != nil {
		t.Fatalf("could not hash deposit: %v", err)
	}

	if db.HasDeposit(hash) {
		t.Fatal("Expected HasDeposit to return false")
	}

	if err := db.SaveDeposit(d); err != nil {
		t.Fatalf("Failed to save deposit: %v", err)
	}
	if !db.HasDeposit(hash) {
		t.Fatal("Expected HasDeposit to return true")
	}
}
