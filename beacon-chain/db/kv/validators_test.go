package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

func TestStore_ValidatorLatestVoteCRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	validatorIdx := uint64(100)
	latestVote := &pb.ValidatorLatestVote{
		Epoch: 1,
		Root:  []byte("root"),
	}
	retrievedVote, err := db.ValidatorLatestVote(ctx, validatorIdx)
	if err != nil {
		t.Fatal(err)
	}
	if retrievedVote != nil {
		t.Errorf("Expected nil validator latest vote, received %v", retrievedVote)
	}
	if err := db.SaveValidatorLatestVote(ctx, validatorIdx, latestVote); err != nil {
		t.Fatal(err)
	}
	if !db.HasValidatorLatestVote(ctx, validatorIdx) {
		t.Error("Expected validator latest vote to exist in the db")
	}
	retrievedVote, err = db.ValidatorLatestVote(ctx, validatorIdx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(latestVote, retrievedVote) {
		t.Errorf("Wanted %d, received %d", latestVote, retrievedVote)
	}
	if err := db.DeleteValidatorLatestVote(ctx, validatorIdx); err != nil {
		t.Fatal(err)
	}
	if db.HasValidatorLatestVote(ctx, validatorIdx) {
		t.Error("Expected validator latest vote to have been deleted from the db")
	}
}

func TestStore_ValidatorLatestVoteCRUD_NoCache(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	validatorIdx := uint64(100)
	latestVote := &pb.ValidatorLatestVote{
		Epoch: 1,
		Root:  []byte("root"),
	}
	retrievedVote, err := db.ValidatorLatestVote(ctx, validatorIdx)
	if err != nil {
		t.Fatal(err)
	}
	if retrievedVote != nil {
		t.Errorf("Expected nil validator latest vote, received %v", retrievedVote)
	}
	if err := db.SaveValidatorLatestVote(ctx, validatorIdx, latestVote); err != nil {
		t.Fatal(err)
	}
	db.votesCache.Delete(string(validatorIdx))
	if !db.HasValidatorLatestVote(ctx, validatorIdx) {
		t.Error("Expected validator latest vote to exist in the db")
	}
	retrievedVote, err = db.ValidatorLatestVote(ctx, validatorIdx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(latestVote, retrievedVote) {
		t.Errorf("Wanted %d, received %d", latestVote, retrievedVote)
	}
	if err := db.DeleteValidatorLatestVote(ctx, validatorIdx); err != nil {
		t.Fatal(err)
	}
	if db.HasValidatorLatestVote(ctx, validatorIdx) {
		t.Error("Expected validator latest vote to have been deleted from the db")
	}
}
