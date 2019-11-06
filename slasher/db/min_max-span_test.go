package db

import (
	"github.com/gogo/protobuf/proto"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

type spanMapTestStruct struct {
	validatorID uint64
	spanMap     *ethpb.EpochSpanMap
}

var spanTests []spanMapTestStruct

func init() {
	spanTests = []spanMapTestStruct{
		{
			validatorID: 1,
			spanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					1: {MinSpan: 10, MaxSpan: 20},
					2: {MinSpan: 11, MaxSpan: 21},
					3: {MinSpan: 12, MaxSpan: 22},
				},
			},
		},
		{
			validatorID: 2,
			spanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					1: {MinSpan: 10, MaxSpan: 20},
					2: {MinSpan: 11, MaxSpan: 21},
					3: {MinSpan: 12, MaxSpan: 22},
				},
			},
		},
		{
			validatorID: 3,
			spanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					1: {MinSpan: 10, MaxSpan: 20},
					2: {MinSpan: 11, MaxSpan: 21},
					3: {MinSpan: 12, MaxSpan: 22},
				},
			},
		},
	}
}

func TestNilDBValidatorSpanMap(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	validatorID := uint64(1)
	vsm, err := db.ValidatorSpansMap(validatorID)
	if err != nil {
		t.Fatalf("Nil ValidatorSpansMap should not return error: %v", err)
	}
	if vsm.EpochSpanMap != nil {
		t.Fatal("ValidatorSpansMap should return nil")
	}
}

func TestSaveValidatorSpanMap(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(tt.validatorID, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
		sm, err := db.ValidatorSpansMap(tt.validatorID)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}

		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
	}
}

func TestDeleteValidatorSpanMap(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)

	for _, tt := range spanTests {

		err := db.SaveValidatorSpansMap(tt.validatorID, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
	}

	for _, tt := range spanTests {
		sm, err := db.ValidatorSpansMap(tt.validatorID)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v", sm)
		}
		err = db.DeleteValidatorSpanMap(tt.validatorID)
		if err != nil {
			t.Fatalf("Delete validator span map error: %v", err)
		}
		sm, err = db.ValidatorSpansMap(tt.validatorID)
		if err != nil {
			t.Fatal(err)
		}
		if sm.EpochSpanMap != nil {
			t.Errorf("Expected validator span map to be deleted, received: %v", sm)
		}
	}
}
