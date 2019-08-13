package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestStore_AttestationCRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard:      5,
				ParentRoot: []byte("parent"),
				StartEpoch: 1,
				EndEpoch:   2,
			},
		},
	}
	ctx := context.Background()
	if err := db.SaveAttestation(ctx, att); err != nil {
		t.Fatal(err)
	}
	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		t.Fatal(err)
	}
	if !db.HasAttestation(ctx, attRoot) {
		t.Error("Expected attestation to exist in the db")
	}
	retrievedAtt, err := db.Attestation(ctx, attRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(att, retrievedAtt) {
		t.Errorf("Wanted %v, received %v", att, retrievedAtt)
	}
	if err := db.DeleteAttestation(ctx, attRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasAttestation(ctx, attRoot) {
		t.Error("Expected attestation to have been deleted from the db")
	}
}

func TestStore_Attestations_FiltersCorrectly(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	atts := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:      5,
					ParentRoot: []byte("parent"),
					StartEpoch: 1,
					EndEpoch:   2,
				},
			},
		},
		{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:      5,
					ParentRoot: []byte("parent2"),
					StartEpoch: 10,
					EndEpoch:   11,
				},
			},
		},
		{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:      4,
					ParentRoot: []byte("parent3"),
					StartEpoch: 1,
					EndEpoch:   20,
				},
			},
		},
	}
	ctx := context.Background()
	if err := db.SaveAttestations(ctx, atts); err != nil {
		t.Fatal(err)
	}
	retrievedAtts, err := db.Attestations(ctx, filters.NewFilter().SetShard(5))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 2 {
		t.Errorf("Expected 2 attestations with shard 5, received %d", len(retrievedAtts))
	}
	retrievedAtts, err = db.Attestations(ctx, filters.NewFilter().SetStartEpoch(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 2 {
		t.Errorf("Expected 2 attestations with start epoch 1, received %d", len(retrievedAtts))
	}
	retrievedAtts, err = db.Attestations(ctx, filters.NewFilter().SetParentRoot([]byte("parent3")))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 1 {
		t.Errorf("Expected 1 attestation with parent root %v, received %d", []byte("parent3"), len(retrievedAtts))
	}
	// More complex filter, multiple attributes.
	f := filters.NewFilter().SetShard(5).SetStartEpoch(1)
	retrievedAtts, err = db.Attestations(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 1 {
		t.Errorf("Expected 1 attestation for a complex filter, received %d", len(retrievedAtts))
	}
	// No filter, should return all attestations.
	retrievedAtts, err = db.Attestations(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 3 {
		t.Errorf("Expected 3 attestation, received %d", len(retrievedAtts))
	}
}
