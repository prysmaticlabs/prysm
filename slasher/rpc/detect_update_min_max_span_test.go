package rpc

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/db"
)

type spanMapTestStruct struct {
	validatorIdx        uint64
	sourceEpoch         uint64
	targetEpoch         uint64
	slashingTargetEpoch uint64
	resultSpanMap       *ethpb.EpochSpanMap
}

var spanTestsMax []spanMapTestStruct
var spanTestsMin []spanMapTestStruct

func init() {
	// Test data following example of a max span by https://github.com/protolambda
	// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
	spanTestsMax = []spanMapTestStruct{
		{
			validatorIdx:        0,
			sourceEpoch:         3,
			targetEpoch:         6,
			slashingTargetEpoch: 0,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					4: {MinEpochSpan: 0, MaxEpochSpan: 2},
					5: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         8,
			targetEpoch:         18,
			slashingTargetEpoch: 0,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					4:  {MinEpochSpan: 0, MaxEpochSpan: 2},
					5:  {MinEpochSpan: 0, MaxEpochSpan: 1},
					9:  {MinEpochSpan: 0, MaxEpochSpan: 9},
					10: {MinEpochSpan: 0, MaxEpochSpan: 8},
					11: {MinEpochSpan: 0, MaxEpochSpan: 7},
					12: {MinEpochSpan: 0, MaxEpochSpan: 6},
					13: {MinEpochSpan: 0, MaxEpochSpan: 5},
					14: {MinEpochSpan: 0, MaxEpochSpan: 4},
					15: {MinEpochSpan: 0, MaxEpochSpan: 3},
					16: {MinEpochSpan: 0, MaxEpochSpan: 2},
					17: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         4,
			targetEpoch:         12,
			slashingTargetEpoch: 0,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					4:  {MinEpochSpan: 0, MaxEpochSpan: 2},
					5:  {MinEpochSpan: 0, MaxEpochSpan: 7},
					6:  {MinEpochSpan: 0, MaxEpochSpan: 6},
					7:  {MinEpochSpan: 0, MaxEpochSpan: 5},
					8:  {MinEpochSpan: 0, MaxEpochSpan: 4},
					9:  {MinEpochSpan: 0, MaxEpochSpan: 9},
					10: {MinEpochSpan: 0, MaxEpochSpan: 8},
					11: {MinEpochSpan: 0, MaxEpochSpan: 7},
					12: {MinEpochSpan: 0, MaxEpochSpan: 6},
					13: {MinEpochSpan: 0, MaxEpochSpan: 5},
					14: {MinEpochSpan: 0, MaxEpochSpan: 4},
					15: {MinEpochSpan: 0, MaxEpochSpan: 3},
					16: {MinEpochSpan: 0, MaxEpochSpan: 2},
					17: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         10,
			targetEpoch:         15,
			slashingTargetEpoch: 18,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					4:  {MinEpochSpan: 0, MaxEpochSpan: 2},
					5:  {MinEpochSpan: 0, MaxEpochSpan: 7},
					6:  {MinEpochSpan: 0, MaxEpochSpan: 6},
					7:  {MinEpochSpan: 0, MaxEpochSpan: 5},
					8:  {MinEpochSpan: 0, MaxEpochSpan: 4},
					9:  {MinEpochSpan: 0, MaxEpochSpan: 9},
					10: {MinEpochSpan: 0, MaxEpochSpan: 8},
					11: {MinEpochSpan: 0, MaxEpochSpan: 7},
					12: {MinEpochSpan: 0, MaxEpochSpan: 6},
					13: {MinEpochSpan: 0, MaxEpochSpan: 5},
					14: {MinEpochSpan: 0, MaxEpochSpan: 4},
					15: {MinEpochSpan: 0, MaxEpochSpan: 3},
					16: {MinEpochSpan: 0, MaxEpochSpan: 2},
					17: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
	}

	spanTestsMin = []spanMapTestStruct{
		{
			validatorIdx:        0,
			sourceEpoch:         4,
			targetEpoch:         6,
			slashingTargetEpoch: 0,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 5, MaxEpochSpan: 0},
					2: {MinEpochSpan: 4, MaxEpochSpan: 0},
					3: {MinEpochSpan: 3, MaxEpochSpan: 0},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         13,
			targetEpoch:         18,
			slashingTargetEpoch: 0,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1:  {MinEpochSpan: 5, MaxEpochSpan: 0},
					2:  {MinEpochSpan: 4, MaxEpochSpan: 0},
					3:  {MinEpochSpan: 3, MaxEpochSpan: 0},
					4:  {MinEpochSpan: 14, MaxEpochSpan: 0},
					5:  {MinEpochSpan: 13, MaxEpochSpan: 0},
					6:  {MinEpochSpan: 12, MaxEpochSpan: 0},
					7:  {MinEpochSpan: 11, MaxEpochSpan: 0},
					8:  {MinEpochSpan: 10, MaxEpochSpan: 0},
					9:  {MinEpochSpan: 9, MaxEpochSpan: 0},
					10: {MinEpochSpan: 8, MaxEpochSpan: 0},
					11: {MinEpochSpan: 7, MaxEpochSpan: 0},
					12: {MinEpochSpan: 6, MaxEpochSpan: 0},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         11,
			targetEpoch:         15,
			slashingTargetEpoch: 0,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1:  {MinEpochSpan: 5, MaxEpochSpan: 0},
					2:  {MinEpochSpan: 4, MaxEpochSpan: 0},
					3:  {MinEpochSpan: 3, MaxEpochSpan: 0},
					4:  {MinEpochSpan: 11, MaxEpochSpan: 0},
					5:  {MinEpochSpan: 10, MaxEpochSpan: 0},
					6:  {MinEpochSpan: 9, MaxEpochSpan: 0},
					7:  {MinEpochSpan: 8, MaxEpochSpan: 0},
					8:  {MinEpochSpan: 7, MaxEpochSpan: 0},
					9:  {MinEpochSpan: 6, MaxEpochSpan: 0},
					10: {MinEpochSpan: 5, MaxEpochSpan: 0},
					11: {MinEpochSpan: 7, MaxEpochSpan: 0},
					12: {MinEpochSpan: 6, MaxEpochSpan: 0},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         10,
			targetEpoch:         20,
			slashingTargetEpoch: 15,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
					1:  {MinEpochSpan: 5, MaxEpochSpan: 0},
					2:  {MinEpochSpan: 4, MaxEpochSpan: 0},
					3:  {MinEpochSpan: 3, MaxEpochSpan: 0},
					4:  {MinEpochSpan: 11, MaxEpochSpan: 0},
					5:  {MinEpochSpan: 10, MaxEpochSpan: 0},
					6:  {MinEpochSpan: 9, MaxEpochSpan: 0},
					7:  {MinEpochSpan: 8, MaxEpochSpan: 0},
					8:  {MinEpochSpan: 7, MaxEpochSpan: 0},
					9:  {MinEpochSpan: 6, MaxEpochSpan: 0},
					10: {MinEpochSpan: 5, MaxEpochSpan: 0},
					11: {MinEpochSpan: 7, MaxEpochSpan: 0},
					12: {MinEpochSpan: 6, MaxEpochSpan: 0},
				},
			},
		},
	}
}

func TestServer_UpdateMaxEpochSpan(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, tt := range spanTestsMax {
		st, err := slasherServer.DetectAndUpdateMaxEpochSpan(ctx, tt.sourceEpoch, tt.targetEpoch, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to update span: %v", err)
		}
		if st != tt.slashingTargetEpoch {
			t.Fatalf("Expected slashing target: %d got: %v", tt.slashingTargetEpoch, st)
		}
		sm, err := slasherServer.SlasherDB.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to retrieve span: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.resultSpanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.resultSpanMap, sm)
		}
	}
}

func TestServer_UpdateMinEpochSpan(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, tt := range spanTestsMin {
		st, err := slasherServer.DetectAndUpdateMinEpochSpan(ctx, tt.sourceEpoch, tt.targetEpoch, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to update span: %v", err)
		}
		if st != tt.slashingTargetEpoch {
			t.Fatalf("Expected slashing target: %v got: %v", tt.slashingTargetEpoch, st)
		}
		sm, err := slasherServer.SlasherDB.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to retrieve span: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.resultSpanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.resultSpanMap, sm)
		}
	}
}

func TestServer_FailToUpdate(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	spanTestsFail := spanMapTestStruct{
		validatorIdx:        0,
		sourceEpoch:         0,
		slashingTargetEpoch: 0,
		targetEpoch:         params.BeaconConfig().WeakSubjectivityPeriod + 1,
		resultSpanMap: &ethpb.EpochSpanMap{
			EpochSpanMap: map[uint64]*ethpb.MinMaxEpochSpan{
				4: {MinEpochSpan: 0, MaxEpochSpan: 2},
				5: {MinEpochSpan: 0, MaxEpochSpan: 1},
			},
		},
	}
	if _, err := slasherServer.DetectAndUpdateMinEpochSpan(ctx, spanTestsFail.sourceEpoch, spanTestsFail.targetEpoch, spanTestsFail.validatorIdx); err == nil {
		t.Fatalf("Update should not support diff greater then weak subjectivity period: %v ", params.BeaconConfig().WeakSubjectivityPeriod)
	}
	if _, err := slasherServer.DetectAndUpdateMaxEpochSpan(ctx, spanTestsFail.sourceEpoch, spanTestsFail.targetEpoch, spanTestsFail.validatorIdx); err == nil {
		t.Fatalf("Update should not support diff greater then weak subjectivity period: %v ", params.BeaconConfig().WeakSubjectivityPeriod)
	}

}
