package db

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveAndRetrieveLatestMessage(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	validator := uint64(1)
	wanted := &pb.LatestMessage{
		Epoch: 100,
		Root: []byte{'A'},
	}

	if err := db.SaveLatestMessage(context.Background(), validator, wanted); err != nil {
		t.Fatalf("Failed to save latest message: %v", err)
	}

	received, err := db.LatestMessage(validator)
	if err != nil {
		t.Fatalf("Failed to get latest message: %v", err)
	}
	if !proto.Equal(wanted, received) {
		t.Error("Did not receive wanted message")
	}

	validator = uint64(2)
	received, err = db.LatestMessage(validator)
	if err != nil {
		t.Fatalf("Failed to get latest message: %v", err)
	}
	if received != nil {
		t.Error("Unsaved latest message exists in db")
	}
}

func TestHasLatestMessage(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	validator := uint64(3)
	if err := db.SaveLatestMessage(context.Background(), validator, &pb.LatestMessage{}); err != nil {
		t.Fatalf("Failed to save latest message: %v", err)
	}
	if !db.HasLatestMessage(validator) {
		t.Error("Wanted true for has latest message")
	}

	validator = uint64(4)
	if db.HasLatestMessage(validator) {
		t.Error("Wanted false for has latest message")
	}
}