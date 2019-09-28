package kv

import (
	"context"
	"reflect"
	"sync"
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
	attDataRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		t.Fatal(err)
	}
	retrievedAtt, err := db.Attestation(ctx, attDataRoot)
	if err != nil {
		t.Fatal(err)
	}
	if retrievedAtt != nil {
		t.Errorf("Expected nil attestation, received %v", retrievedAtt)
	}
	if err := db.SaveAttestation(ctx, att); err != nil {
		t.Fatal(err)
	}
	if !db.HasAttestation(ctx, attDataRoot) {
		t.Error("Expected attestation to exist in the db")
	}
	retrievedAtt, err = db.Attestation(ctx, attDataRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(att, retrievedAtt) {
		t.Errorf("Wanted %v, received %v", att, retrievedAtt)
	}
	if err := db.DeleteAttestation(ctx, attDataRoot); err != nil {
		t.Fatal(err)
	}
	if db.HasAttestation(ctx, attDataRoot) {
		t.Error("Expected attestation to have been deleted from the db")
	}
}

func TestStore_AttestationsBatchDelete(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	numAtts := 1000
	totalAtts := make([]*ethpb.Attestation, numAtts)
	// We track the data roots for the even indexed attestations.
	attDataRoots := make([][32]byte, 0)
	oddAtts := make([]*ethpb.Attestation, 0)
	for i := 0; i < len(totalAtts); i++ {
		totalAtts[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("head"),
				Crosslink: &ethpb.Crosslink{
					Shard:      uint64(i),
					ParentRoot: []byte("parent"),
					StartEpoch: 1,
					EndEpoch:   2,
				},
			},
		}
		if i%2 == 0 {
			r, err := ssz.HashTreeRoot(totalAtts[i].Data)
			if err != nil {
				t.Fatal(err)
			}
			attDataRoots = append(attDataRoots, r)
		} else {
			oddAtts = append(oddAtts, totalAtts[i])
		}
	}
	if err := db.SaveAttestations(ctx, totalAtts); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.Attestations(ctx, filters.NewFilter().SetHeadBlockRoot([]byte("head")))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrieved) != numAtts {
		t.Errorf("Received %d attestations, wanted 1000", len(retrieved))
	}
	// We delete all even indexed attestation.
	if err := db.DeleteAttestations(ctx, attDataRoots); err != nil {
		t.Fatal(err)
	}
	// When we retrieve the data, only the odd indexed attestations should remain.
	retrieved, err = db.Attestations(ctx, filters.NewFilter().SetHeadBlockRoot([]byte("head")))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(retrieved, oddAtts) {
		t.Errorf("Wanted %v, received %v", oddAtts, retrieved)
	}
}

func TestStore_BoltDontPanic(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	var wg sync.WaitGroup

	for i := 0; i <= 100; i++ {
		att := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:      5,
					ParentRoot: []byte("parent"),
					StartEpoch: uint64(i + 1),
					EndEpoch:   2,
				},
			},
		}
		ctx := context.Background()
		attDataRoot, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			t.Fatal(err)
		}
		retrievedAtt, err := db.Attestation(ctx, attDataRoot)
		if err != nil {
			t.Fatal(err)
		}
		if retrievedAtt != nil {
			t.Errorf("Expected nil attestation, received %v", retrievedAtt)
		}
		if err := db.SaveAttestation(ctx, att); err != nil {
			t.Fatal(err)
		}
	}
	// if indices are improperly deleted this test will then panic.
	for i := 0; i <= 100; i++ {
		startEpoch := i + 1
		wg.Add(1)
		go func() {
			att := &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Crosslink: &ethpb.Crosslink{
						Shard:      5,
						ParentRoot: []byte("parent"),
						StartEpoch: uint64(startEpoch),
						EndEpoch:   2,
					},
				},
			}
			ctx := context.Background()
			attDataRoot, err := ssz.HashTreeRoot(att.Data)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.DeleteAttestation(ctx, attDataRoot); err != nil {
				t.Fatal(err)
			}
			if db.HasAttestation(ctx, attDataRoot) {
				t.Error("Expected attestation to have been deleted from the db")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestStore_Attestations_FiltersCorrectly(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	someRoot := [32]byte{1, 2, 3}
	otherRoot := [32]byte{4, 5, 6}
	atts := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: someRoot[:],
				Source: &ethpb.Checkpoint{
					Root:  someRoot[:],
					Epoch: 5,
				},
				Target: &ethpb.Checkpoint{
					Root:  someRoot[:],
					Epoch: 7,
				},
			},
		},
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: someRoot[:],
				Source: &ethpb.Checkpoint{
					Root:  otherRoot[:],
					Epoch: 5,
				},
				Target: &ethpb.Checkpoint{
					Root:  otherRoot[:],
					Epoch: 7,
				},
			},
		},
		{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: otherRoot[:],
				Source: &ethpb.Checkpoint{
					Root:  someRoot[:],
					Epoch: 7,
				},
				Target: &ethpb.Checkpoint{
					Root:  someRoot[:],
					Epoch: 5,
				},
			},
		},
	}
	ctx := context.Background()
	if err := db.SaveAttestations(ctx, atts); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		filter         *filters.QueryFilter
		expectedNumAtt int
	}{
		{
			filter: filters.NewFilter().
				SetSourceEpoch(5),
			expectedNumAtt: 2,
		},
		{
			filter: filters.NewFilter().
				SetHeadBlockRoot(someRoot[:]),
			expectedNumAtt: 2,
		},
		{
			filter: filters.NewFilter().
				SetHeadBlockRoot(otherRoot[:]),
			expectedNumAtt: 1,
		},
		{
			filter:         filters.NewFilter().SetTargetEpoch(7),
			expectedNumAtt: 2,
		},
		{
			// Only two attestation in the list meet the composite filter criteria above.
			filter: filters.NewFilter().
				SetHeadBlockRoot(someRoot[:]).
				SetTargetEpoch(7),
			expectedNumAtt: 2,
		},
		{
			// No attestation meets the criteria below.
			filter: filters.NewFilter().
				SetTargetEpoch(1000),
			expectedNumAtt: 0,
		},
	}
	for _, tt := range tests {
		retrievedAtts, err := db.Attestations(ctx, tt.filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(retrievedAtts) != tt.expectedNumAtt {
			t.Errorf("Expected %d attestations, received %d", tt.expectedNumAtt, len(retrievedAtts))
		}
	}
}
