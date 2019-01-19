package db

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	att "github.com/prysmaticlabs/prysm/beacon-chain/core/attestations"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveAndRetrieveAttestation(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	a := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:  0,
			Shard: 0,
		},
	}

	if err := db.SaveAttestation(a); err != nil {
		t.Fatalf("Failed to save attestation: %v", err)
	}

	aHash := att.Key(a.GetData())
	aPrime, err := db.Attestation(aHash)
	if err != nil {
		t.Fatalf("Failed to call Attestation: %v", err)
	}

	aEnc, err := proto.Marshal(a)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	aPrimeEnc, err := proto.Marshal(aPrime)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	if !bytes.Equal(aEnc, aPrimeEnc) {
		t.Fatalf("Saved attestation and retrieved attestation are not equal: %#x and %#x", aEnc, aPrimeEnc)
	}
}

func TestNilAttestation(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	nilHash := [32]byte{}
	a, err := db.Attestation(nilHash)
	if err != nil {
		t.Fatalf("Failed to retrieve nilHash: %v", err)
	}
	if a != nil {
		t.Fatal("Expected nilHash to return no attestation")
	}
}

func TestHasAttestation(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	a := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:  0,
			Shard: 0,
		},
	}
	hash := att.Key(a.GetData())

	if db.HasAttestation(hash) {
		t.Fatal("Expected HasAttestation to return false")
	}

	if err := db.SaveAttestation(a); err != nil {
		t.Fatalf("Failed to save attestation: %v", err)
	}
	if !db.HasAttestation(hash) {
		t.Fatal("Expected HasAttestation to return true")
	}
}
