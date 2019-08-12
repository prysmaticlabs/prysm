package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"

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
	//if !db.HasValidatorLatestVote(ctx, validatorIdx) {
	//	t.Error("Expected validator latest vote to exist in the db")
	//}
	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		t.Fatal(err)
	}
	retrievedAtt, err := db.Attestation(ctx, attRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(att, retrievedAtt) {
		t.Errorf("Wanted %d, received %d", att, retrievedAtt)
	}
}
