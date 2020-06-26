package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

func TestStore_ProposerSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	prop := &ethpb.ProposerSlashing{
		Header_1: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
				BodyRoot:      make([]byte, 32),
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
		Header_2: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
				BodyRoot:      make([]byte, 32),
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
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
	if err := db.deleteProposerSlashing(ctx, slashingRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasProposerSlashing(ctx, slashingRoot) {
		t.Error("Expected proposer slashing to have been deleted from the db")
	}
}

func TestStore_AttesterSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	att := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Slot:            5,
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
			},
			Signature: make([]byte, 96),
		},
		Attestation_2: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Slot:            7,
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
			},
			Signature: make([]byte, 96),
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
	if err := db.deleteAttesterSlashing(ctx, slashingRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasAttesterSlashing(ctx, slashingRoot) {
		t.Error("Expected attester slashing to have been deleted from the db")
	}
}
