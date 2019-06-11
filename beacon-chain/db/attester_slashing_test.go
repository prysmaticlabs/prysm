package db

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestBeaconDB_HasAttesterSlashing(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	d := &pb.AttesterSlashing{
		Attestation_1: &pb.IndexedAttestation{CustodyBit_0Indices: []uint64{0}},
		Attestation_2: &pb.IndexedAttestation{CustodyBit_0Indices: []uint64{1}},
	}
	hash, err := hashutil.HashProto(d)
	if err != nil {
		t.Fatalf("could not hash attester slashing request: %v", err)
	}

	if db.HasAttesterSlashing(hash) {
		t.Fatal("Expected HasAttesterSlashing to return false")
	}

	if err := db.SaveAttesterSlashing(context.Background(), d); err != nil {
		t.Fatalf("Failed to save attester slashing request: %v", err)
	}
	if !db.HasAttesterSlashing(hash) {
		t.Fatal("Expected HasAttesterSlashing to return true")
	}
}
