package kv

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
)

func TestStore_AttestationCRUD(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	att := &ethpb.Attestation{
		Data:            &ethpb.AttestationData{Slot: 10},
		AggregationBits: bitfield.Bitlist{0b00000001, 0b1},
	}
	ctx := context.Background()
	attDataRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		t.Fatal(err)
	}
	retrievedAtts, err := db.AttestationsByDataRoot(ctx, attDataRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 0 {
		t.Errorf("Expected no attestations, received %v", retrievedAtts)
	}
	if err := db.SaveAttestation(ctx, att); err != nil {
		t.Fatal(err)
	}
	if !db.HasAttestation(ctx, attDataRoot) {
		t.Error("Expected attestation to exist in the db")
	}
	retrievedAtts, err = db.AttestationsByDataRoot(ctx, attDataRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(att, retrievedAtts[0]) {
		t.Errorf("Wanted %v, received %v", att, retrievedAtts[0])
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
	numAtts := 10
	totalAtts := make([]*ethpb.Attestation, numAtts)
	// We track the data roots for the even indexed attestations.
	attDataRoots := make([][32]byte, 0)
	oddAtts := make([]*ethpb.Attestation, 0)
	for i := 0; i < len(totalAtts); i++ {
		totalAtts[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: []byte("head"),
				Slot:            uint64(i),
			},
			AggregationBits: bitfield.Bitlist{0b00000001, 0b1},
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
	sort.Slice(retrieved, func(i, j int) bool {
		return retrieved[i].Data.Slot < retrieved[j].Data.Slot
	})
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
				Slot:   uint64(i),
				Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0b11},
		}
		ctx := context.Background()
		attDataRoot, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			t.Fatal(err)
		}
		retrievedAtts, err := db.AttestationsByDataRoot(ctx, attDataRoot)
		if err != nil {
			t.Fatal(err)
		}
		if len(retrievedAtts) != 0 {
			t.Errorf("Expected no attestation, received %v", retrievedAtts)
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
				Data:            &ethpb.AttestationData{Slot: uint64(startEpoch)},
				AggregationBits: bitfield.Bitlist{0b11},
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
			AggregationBits: bitfield.Bitlist{0b11},
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
			AggregationBits: bitfield.Bitlist{0b11},
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
			AggregationBits: bitfield.Bitlist{0b11},
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

func TestStore_DuplicatedAttestations_FiltersCorrectly(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	someRoot := [32]byte{1, 2, 3}
	att := &ethpb.Attestation{
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
		AggregationBits: bitfield.Bitlist{0b11},
	}
	atts := []*ethpb.Attestation{att, att, att}
	ctx := context.Background()
	if err := db.SaveAttestations(ctx, atts); err != nil {
		t.Fatal(err)
	}

	retrievedAtts, err := db.Attestations(ctx, filters.NewFilter().
		SetHeadBlockRoot(someRoot[:]))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 1 {
		t.Errorf("Expected %d attestations, received %d", 1, len(retrievedAtts))
	}

	att1 := proto.Clone(att).(*ethpb.Attestation)
	att1.Data.Source.Epoch = 6
	atts = []*ethpb.Attestation{att, att, att, att1, att1, att1}
	if err := db.SaveAttestations(ctx, atts); err != nil {
		t.Fatal(err)
	}

	retrievedAtts, err = db.Attestations(ctx, filters.NewFilter().
		SetHeadBlockRoot(someRoot[:]))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 2 {
		t.Errorf("Expected %d attestations, received %d", 1, len(retrievedAtts))
	}

	retrievedAtts, err = db.Attestations(ctx, filters.NewFilter().
		SetHeadBlockRoot(someRoot[:]).SetSourceEpoch(5))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 1 {
		t.Errorf("Expected %d attestations, received %d", 1, len(retrievedAtts))
	}

	retrievedAtts, err = db.Attestations(ctx, filters.NewFilter().
		SetHeadBlockRoot(someRoot[:]).SetSourceEpoch(6))
	if err != nil {
		t.Fatal(err)
	}
	if len(retrievedAtts) != 1 {
		t.Errorf("Expected %d attestations, received %d", 1, len(retrievedAtts))
	}
}

func TestStore_Attestations_BitfieldLogic(t *testing.T) {
	commonData := &ethpb.AttestationData{Slot: 10}

	tests := []struct {
		name   string
		input  []*ethpb.Attestation
		output []*ethpb.Attestation
	}{
		{
			name: "all distinct aggregation bitfields",
			input: []*ethpb.Attestation{
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000001},
				},
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000010},
				},
			},
			output: []*ethpb.Attestation{
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000001},
				},
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000010},
				},
			},
		},
		{
			name: "Incoming attestation is fully contained already",
			input: []*ethpb.Attestation{
				{
					Data:            commonData,
					AggregationBits: []byte{0b11111111},
				},
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000010},
				},
			},
			output: []*ethpb.Attestation{
				{
					Data:            commonData,
					AggregationBits: []byte{0b11111111},
				},
			},
		},
		{
			name: "Existing attestations are fully contained incoming attestation",
			input: []*ethpb.Attestation{
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000001},
				},
				{
					Data:            commonData,
					AggregationBits: []byte{0b10000010},
				},
				{
					Data:            commonData,
					AggregationBits: []byte{0b11111111},
				},
			},
			output: []*ethpb.Attestation{
				{
					Data:            commonData,
					AggregationBits: []byte{0b11111111},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			defer teardownDB(t, db)
			ctx := context.Background()
			if err := db.SaveAttestations(ctx, tt.input); err != nil {
				t.Fatal(err)
			}
			r, err := ssz.HashTreeRoot(tt.input[0].Data)
			if err != nil {
				t.Fatal(err)
			}
			output, err := db.AttestationsByDataRoot(ctx, r)
			if err != nil {
				t.Fatal(err)
			}
			if len(output) != len(tt.output) {
				t.Fatalf(
					"Wrong number of attestations returned. Got %d attestations but wanted %d",
					len(output),
					len(tt.output),
				)
			}
			sort.Slice(output, func(i, j int) bool {
				return output[i].AggregationBits.Bytes()[0] < output[j].AggregationBits.Bytes()[0]
			})
			sort.Slice(tt.output, func(i, j int) bool {
				return tt.output[i].AggregationBits.Bytes()[0] < tt.output[j].AggregationBits.Bytes()[0]
			})
			for i, att := range output {
				if !bytes.Equal(att.AggregationBits, tt.output[i].AggregationBits) {
					t.Errorf("Aggregation bits are not the same. Got %b, wanted %b", att.AggregationBits, tt.output[i].AggregationBits)
				}
			}
		})
	}
}
