package db

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestBeaconDB_HasProposerSlashing(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	d := &pb.ProposerSlashing{
		ProposerIndex: 100,
		Header_2:      &pb.BeaconBlockHeader{},
		Header_1:      &pb.BeaconBlockHeader{},
	}
	hash, err := hashutil.HashProto(d)
	if err != nil {
		t.Fatalf("could not hash proposer slashing request: %v", err)
	}

	if db.HasProposerSlashing(hash) {
		t.Fatal("Expected HasProposerSlashing to return false")
	}

	if err := db.SaveProposerSlashing(context.Background(), d); err != nil {
		t.Fatalf("Failed to save proposer slashing request: %v", err)
	}
	if !db.HasProposerSlashing(hash) {
		t.Fatal("Expected HasProposerSlashing to return true")
	}
}
