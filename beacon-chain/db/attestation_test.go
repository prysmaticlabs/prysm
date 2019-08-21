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

	aDataHash, err := hashutil.HashProto(a.Data)
	if err != nil {
		t.Fatalf("Failed to hash AttestationDeprecated: %v", err)
	}
	aPrime, err := db.AttestationDeprecated(aDataHash)
	if err != nil {
		t.Fatalf("Failed to call AttestationDeprecated: %v", err)
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

	retrievedAttestations, err := db.AttestationsDeprecated()
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

	aDataHash, err := hashutil.HashProto(a.Data)
	if err != nil {
		t.Fatalf("Failed to hash AttestationDeprecated: %v", err)
	}
	aPrime, err := db.AttestationDeprecated(aDataHash)
	if err != nil {
		t.Fatalf("Could not call AttestationDeprecated: %v", err)
	}

	if !reflect.DeepEqual(aPrime, a) {
		t.Errorf("Saved attestation and retrieved attestation are not equal")
	}

	if err := db.DeleteAttestationDeprecated(a); err != nil {
		t.Fatalf("Could not delete attestation: %v", err)
	}

	if db.HasAttestationDeprecated(aDataHash) {
		t.Error("Deleted attestation still there")
	}
}

func TestNilAttestation_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	nilHash := [32]byte{}
	a, err := db.AttestationDeprecated(nilHash)
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
	aDataHash, err := hashutil.HashProto(a.Data)
	if err != nil {
		t.Fatalf("Failed to hash AttestationDeprecated: %v", err)
	}

	if db.HasAttestationDeprecated(aDataHash) {
		t.Fatal("Expected HasAttestationDeprecated to return false")
	}

	if err := db.SaveAttestation(context.Background(), a); err != nil {
		t.Fatalf("Failed to save attestation: %v", err)
	}
	if !db.HasAttestationDeprecated(aDataHash) {
		t.Fatal("Expected HasAttestationDeprecated to return true")
	}
}
