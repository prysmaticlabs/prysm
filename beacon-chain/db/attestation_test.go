package db

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveAndRetrieveAttestation(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	a := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:  0,
		Shard: 0,
	})

	if err := db.SaveAttestation(a); err != nil {
		t.Fatalf("Failed to save attestation: %v", err)
	}

	aHash := a.Key()
	aPrime, err := db.GetAttestation(aHash)
	if err != nil {
		t.Fatalf("Failed to call GetAttestation: %v", err)
	}

	aEnc, err := a.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	aPrimeEnc, err := aPrime.Marshal()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	if !bytes.Equal(aEnc, aPrimeEnc) {
		t.Fatalf("Saved attestation and retrieved attestation are not equal: %#x and %#x", aEnc, aPrimeEnc)
	}
}

func TestGetNilAttestation(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	nilHash := [32]byte{}
	a, err := db.GetAttestation(nilHash)
	if err != nil {
		t.Fatalf("Failed to retrieve nilHash: %v", err)
	}
	if a != nil {
		t.Fatal("Expected nilHash to return no attestation")
	}
}

func TestGetHasAttestation(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	a := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:  0,
		Shard: 0,
	})
	hash := a.Key()

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
