package db

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestStore_AttesterSlashingNilBucket(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	as := &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello")}}
	has, _, err := db.HasAttesterSlashing(as)
	if err != nil {
		t.Fatalf("HasAttesterSlashing should not return error: %v", err)
	}
	if has {
		t.Fatal("HasAttesterSlashing should return false")
	}

	p, err := db.AttesterSlashings(SlashingStatus(Active))
	if err != nil {
		t.Fatalf("failed to get attester slashing: %v", err)
	}
	if p != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestStore_SaveAttesterSlashing(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello")}},
		},
		{
			ss: Included,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello2")}},
		},
		{
			ss: Reverted,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello3")}},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(tt.ss, tt.as)
		if err != nil {
			t.Fatalf("save attester slashing failed: %v", err)
		}

		attesterSlashings, err := db.AttesterSlashings(tt.ss)
		if err != nil {
			t.Fatalf("failed to get attester slashings: %v", err)
		}

		if attesterSlashings == nil || !reflect.DeepEqual(attesterSlashings[0], tt.as) {
			t.Fatalf("attester slashing: %v should be part of attester slashings response: %v", tt.as, attesterSlashings)
		}
	}

}

func TestStore_DeleteAttesterSlashingWithStatus(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello")}},
		},
		{
			ss: Included,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello2")}},
		},
		{
			ss: Reverted,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello3")}},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(tt.ss, tt.as)
		if err != nil {
			t.Fatalf("save attester slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, _, err := db.HasAttesterSlashing(tt.as)
		if err != nil {
			t.Fatalf("failed to get attester slashing: %v", err)
		}
		if !has {
			t.Fatalf("failed to find attester slashing: %v", tt.as)
		}

		err = db.DeleteAttesterSlashingWithStatus(tt.ss, tt.as)
		if err != nil {
			t.Fatalf("delete attester slashings failed: %v", err)
		}
		has, _, err = db.HasAttesterSlashing(tt.as)
		if err != nil {
			t.Fatalf("error while trying to get non existing attester slashing: %v", err)
		}
		if has {
			t.Fatalf("attester slashing: %v should have been deleted", tt.as)
		}

	}

}

func TestStore_UpdateAttesterSlashingStatus(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello")}},
		},
		{
			ss: Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello2")}},
		},
		{
			ss: Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte("hello3")}},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(tt.ss, tt.as)
		if err != nil {
			t.Fatalf("save attester slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, st, err := db.HasAttesterSlashing(tt.as)
		if err != nil {
			t.Fatalf("failed to get attester slashing: %v", err)
		}
		if !has {
			t.Fatalf("failed to find attester slashing: %v", tt.as)
		}
		if st != tt.ss {
			t.Fatalf("failed to find attester slashing with the correct status: %v", tt.as)
		}

		err = db.SaveAttesterSlashing(SlashingStatus(Included), tt.as)
		has, st, err = db.HasAttesterSlashing(tt.as)
		if err != nil {
			t.Fatalf("failed to get attester slashing: %v", err)
		}
		if !has {
			t.Fatalf("failed to find attester slashing: %v", tt.as)
		}
		if st != Included {
			t.Fatalf("failed to find attester slashing with the correct status: %v", tt.as)
		}

	}

}
