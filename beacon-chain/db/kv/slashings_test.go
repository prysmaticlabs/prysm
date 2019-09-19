package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestStore_ProposerSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	prop := &ethpb.ProposerSlashing{
		ProposerIndex: 5,
	}
	slashingRoot, err := ssz.HashTreeRoot(prop)
	if err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.ProposerSlashing(ctx, slashingRoot)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil proposer slashing, received %v", retrieved)
	}
	if err := db.SaveProposerSlashing(ctx, prop); err != nil {
		t.Fatal(err)
	}
	if !db.HasProposerSlashing(ctx, slashingRoot) {
		t.Error("Expected proposer slashing to exist in the db")
	}
	retrieved, err = db.ProposerSlashing(ctx, slashingRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(prop, retrieved) {
		t.Errorf("Wanted %v, received %v", prop, retrieved)
	}
	if err := db.DeleteProposerSlashing(ctx, slashingRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasProposerSlashing(ctx, slashingRoot) {
		t.Error("Expected proposer slashing to have been deleted from the db")
	}
}

func TestStore_AttesterSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	att := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Crosslink: &ethpb.Crosslink{
					Shard: 5,
				},
			},
		},
		Attestation_2: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Crosslink: &ethpb.Crosslink{
					Shard: 7,
				},
			},
		},
	}
	slashingRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.AttesterSlashing(ctx, slashingRoot)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil attester slashing, received %v", retrieved)
	}
	if err := db.SaveAttesterSlashing(ctx, att); err != nil {
		t.Fatal(err)
	}
	if !db.HasAttesterSlashing(ctx, slashingRoot) {
		t.Error("Expected attester slashing to exist in the db")
	}
	retrieved, err = db.AttesterSlashing(ctx, slashingRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(att, retrieved) {
		t.Errorf("Wanted %v, received %v", att, retrieved)
	}
	if err := db.DeleteAttesterSlashing(ctx, slashingRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasAttesterSlashing(ctx, slashingRoot) {
		t.Error("Expected attester slashing to have been deleted from the db")
	}
}
