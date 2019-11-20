package db

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

type spanMapTestStruct struct {
	validatorIdx uint64
	spanMap      *ethpb.EpochSpanMap
}

var spanTests []spanMapTestStruct

func init() {
	spanTests = []spanMapTestStruct{
		{
			validatorIdx: 1,
			spanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 10, MaxEpochSpan: 20},
					2: {MinEpochSpan: 11, MaxEpochSpan: 21},
					3: {MinEpochSpan: 12, MaxEpochSpan: 22},
				},
			},
		},
		{
			validatorIdx: 2,
			spanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 10, MaxEpochSpan: 20},
					2: {MinEpochSpan: 11, MaxEpochSpan: 21},
					3: {MinEpochSpan: 12, MaxEpochSpan: 22},
				},
			},
		},
		{
			validatorIdx: 3,
			spanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 10, MaxEpochSpan: 20},
					2: {MinEpochSpan: 11, MaxEpochSpan: 21},
					3: {MinEpochSpan: 12, MaxEpochSpan: 22},
				},
			},
		},
	}
}

func TestValidatorSpanMap_NilDB(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	validatorIdx := uint64(1)
	vsm, err := db.ValidatorSpansMap(validatorIdx)
	if err != nil {
		t.Fatalf("Nil ValidatorSpansMap should not return error: %v", err)
	}
	if !reflect.DeepEqual(vsm.EpochSpanMap, map[uint64]*ethpb.MinMaxEpochSpan{}) {
		t.Fatal("ValidatorSpansMap should return nil")
	}
}

func TestValidatorSpanMap_Save(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
		sm, err := db.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}

		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
	}
}

func TestValidatorSpanMap_Delete(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
	}

	for _, tt := range spanTests {
		sm, err := db.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
		err = db.DeleteValidatorSpanMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Delete validator span map error: %v", err)
		}
		sm, err = db.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(sm.EpochSpanMap, map[uint64]*ethpb.MinMaxEpochSpan{}) {
			t.Errorf("Expected validator span map to be deleted, received: %v", sm)
		}
	}
}
