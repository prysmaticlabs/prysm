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
	validatorIdx  uint64
	sourceEpoch   uint64
	targetEpoch   uint64
	resultSpanMap *ethpb.EpochSpanMap
}

var spanTestsMax []spanMapTestStruct
var spanTestsMin []spanMapTestStruct
var spanTestsFail []spanMapTestStruct

func init() {
	// Test data following example of a max span by https://github.com/protolambda
	// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
	spanTestsMax = []spanMapTestStruct{
		{
			validatorIdx: 0,
			sourceEpoch:  3,
			targetEpoch:  6,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					4: {MinSpan: 0, MaxSpan: 2},
					5: {MinSpan: 0, MaxSpan: 1},
				},
			},
		},
		{
			validatorIdx: 0,
			sourceEpoch:  8,
			targetEpoch:  18,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					4:  {MinSpan: 0, MaxSpan: 2},
					5:  {MinSpan: 0, MaxSpan: 1},
					9:  {MinSpan: 0, MaxSpan: 9},
					10: {MinSpan: 0, MaxSpan: 8},
					11: {MinSpan: 0, MaxSpan: 7},
					12: {MinSpan: 0, MaxSpan: 6},
					13: {MinSpan: 0, MaxSpan: 5},
					14: {MinSpan: 0, MaxSpan: 4},
					15: {MinSpan: 0, MaxSpan: 3},
					16: {MinSpan: 0, MaxSpan: 2},
					17: {MinSpan: 0, MaxSpan: 1},
				},
			},
		},
		{
			validatorIdx: 0,
			sourceEpoch:  4,
			targetEpoch:  12,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					4:  {MinSpan: 0, MaxSpan: 2},
					5:  {MinSpan: 0, MaxSpan: 7},
					6:  {MinSpan: 0, MaxSpan: 6},
					7:  {MinSpan: 0, MaxSpan: 5},
					8:  {MinSpan: 0, MaxSpan: 4},
					9:  {MinSpan: 0, MaxSpan: 9},
					10: {MinSpan: 0, MaxSpan: 8},
					11: {MinSpan: 0, MaxSpan: 7},
					12: {MinSpan: 0, MaxSpan: 6},
					13: {MinSpan: 0, MaxSpan: 5},
					14: {MinSpan: 0, MaxSpan: 4},
					15: {MinSpan: 0, MaxSpan: 3},
					16: {MinSpan: 0, MaxSpan: 2},
					17: {MinSpan: 0, MaxSpan: 1},
				},
			},
		},
	}

	spanTestsMin = []spanMapTestStruct{
		{
			validatorIdx: 0,
			sourceEpoch:  4,
			targetEpoch:  6,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					1: {MinSpan: 5, MaxSpan: 0},
					2: {MinSpan: 4, MaxSpan: 0},
					3: {MinSpan: 3, MaxSpan: 0},
				},
			},
		},
		{
			validatorIdx: 0,
			sourceEpoch:  13,
			targetEpoch:  18,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					1:  {MinSpan: 5, MaxSpan: 0},
					2:  {MinSpan: 4, MaxSpan: 0},
					3:  {MinSpan: 3, MaxSpan: 0},
					4:  {MinSpan: 14, MaxSpan: 0},
					5:  {MinSpan: 13, MaxSpan: 0},
					6:  {MinSpan: 12, MaxSpan: 0},
					7:  {MinSpan: 11, MaxSpan: 0},
					8:  {MinSpan: 10, MaxSpan: 0},
					9:  {MinSpan: 9, MaxSpan: 0},
					10: {MinSpan: 8, MaxSpan: 0},
					11: {MinSpan: 7, MaxSpan: 0},
					12: {MinSpan: 6, MaxSpan: 0},
				},
			},
		},
		{
			validatorIdx: 0,
			sourceEpoch:  11,
			targetEpoch:  15,
			resultSpanMap: &ethpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
					1:  {MinSpan: 5, MaxSpan: 0},
					2:  {MinSpan: 4, MaxSpan: 0},
					3:  {MinSpan: 3, MaxSpan: 0},
					4:  {MinSpan: 11, MaxSpan: 0},
					5:  {MinSpan: 10, MaxSpan: 0},
					6:  {MinSpan: 9, MaxSpan: 0},
					7:  {MinSpan: 8, MaxSpan: 0},
					8:  {MinSpan: 7, MaxSpan: 0},
					9:  {MinSpan: 6, MaxSpan: 0},
					10: {MinSpan: 5, MaxSpan: 0},
					11: {MinSpan: 7, MaxSpan: 0},
					12: {MinSpan: 6, MaxSpan: 0},
				},
			},
		},
	}

}

func TestServer_UpdateMaxSpan(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()

	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, tt := range spanTestsMax {
		if err := slasherServer.UpdateMaxSpan(ctx, tt.sourceEpoch, tt.targetEpoch, tt.validatorIdx); err != nil {
			t.Fatalf("Failed to update span: %v", err)
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

func TestServer_UpdateMinSpan(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	for _, tt := range spanTestsMin {
		if err := slasherServer.UpdateMinSpan(ctx, tt.sourceEpoch, tt.targetEpoch, tt.validatorIdx); err != nil {
			t.Fatalf("Failed to update span: %v", err)
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

		validatorIdx: 0,
		sourceEpoch:  0,
		targetEpoch:  params.BeaconConfig().WeakSubjectivityPeriod + 1,
		resultSpanMap: &ethpb.EpochSpanMap{
			EpochSpanMap: map[uint64]*ethpb.MinMaxSpan{
				4: {MinSpan: 0, MaxSpan: 2},
				5: {MinSpan: 0, MaxSpan: 1},
			},
		},
	}
	if err := slasherServer.UpdateMinSpan(ctx, spanTestsFail.sourceEpoch, spanTestsFail.targetEpoch, spanTestsFail.validatorIdx); err == nil {
		t.Fatalf("Update should not support diff greater then weak subjectivity period: %v ", params.BeaconConfig().WeakSubjectivityPeriod)
	}
	if err := slasherServer.UpdateMaxSpan(ctx, spanTestsFail.sourceEpoch, spanTestsFail.targetEpoch, spanTestsFail.validatorIdx); err == nil {
		t.Fatalf("Update should not support diff greater then weak subjectivity period: %v ", params.BeaconConfig().WeakSubjectivityPeriod)
	}

}
