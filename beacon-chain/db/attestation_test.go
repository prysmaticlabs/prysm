package db

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestSaveAndRetrieveAttestation_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 0,
			},
		},
	}

	if err := db.SaveAttestation(context.Background(), a); err != nil {
		t.Fatalf("Failed to save attestation: %v", err)
	}

	aHash, err := hashutil.HashProto(a)
	if err != nil {
		t.Fatalf("Failed to hash Attestation: %v", err)
	}
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

func TestRetrieveAttestations_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	// Generate 100 unique attestations to save in DB.
	attestations := make([]*ethpb.Attestation, 100)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: uint64(i),
				},
			},
		}
		if err := db.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	retrievedAttestations, err := db.Attestations()
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	sort.Slice(retrievedAttestations, func(i, j int) bool {
		return retrievedAttestations[i].Data.Crosslink.Shard < retrievedAttestations[j].Data.Crosslink.Shard
	})

	if !reflect.DeepEqual(retrievedAttestations, attestations) {
		t.Fatal("Retrieved attestations did not match generated attestations")
	}
}

func TestDeleteAttestation_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 0,
			},
		},
	}

	if err := db.SaveAttestation(context.Background(), a); err != nil {
		t.Fatalf("Could not save attestation: %v", err)
	}

	aHash, err := hashutil.HashProto(a)
	if err != nil {
		t.Fatalf("Failed to hash Attestation: %v", err)
	}
	aPrime, err := db.Attestation(aHash)
	if err != nil {
		t.Fatalf("Could not call Attestation: %v", err)
	}

	if !reflect.DeepEqual(aPrime, a) {
		t.Errorf("Saved attestation and retrieved attestation are not equal")
	}

	if err := db.DeleteAttestation(a); err != nil {
		t.Fatalf("Could not delete attestation: %v", err)
	}

	if db.HasAttestation(aHash) {
		t.Error("Deleted attestation still there")
	}
}

func TestNilAttestation_OK(t *testing.T) {
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

func TestHasAttestation_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 0,
			},
		},
	}
	aHash, err := hashutil.HashProto(a)
	if err != nil {
		t.Fatalf("Failed to hash Attestation: %v", err)
	}

	if db.HasAttestation(aHash) {
		t.Fatal("Expected HasAttestation to return false")
	}

	if err := db.SaveAttestation(context.Background(), a); err != nil {
		t.Fatalf("Failed to save attestation: %v", err)
	}
	if !db.HasAttestation(aHash) {
		t.Fatal("Expected HasAttestation to return true")
	}
}
