package kv

import (
	"context"
	"encoding/hex"
	"flag"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/urfave/cli/v2"
)

type spansTestStruct struct {
	name           string
	epoch          uint64
	spansHex       string
	spansResultHex string
	validator1Span types.Span
	err            error
}
type spansValueTests struct {
	name          string
	validatorID   uint64
	oldSpans      string
	spansLength   uint64
	validatorSpan types.Span
	err           error
}

var exampleSpansValues []spansValueTests
var spanNewTests []spansTestStruct

func init() {
	spanNewTests = []spansTestStruct{
		{
			name:           "Span too small",
			epoch:          1,
			spansHex:       "00000000",
			spansResultHex: "",
			validator1Span: types.Span{},
			err:            errWrongSize,
		},
		{
			name:           "No validator 1 in spans",
			epoch:          2,
			spansHex:       "00000000000000",
			spansResultHex: "00000000000000",
			validator1Span: types.Span{},
			err:            nil,
		},
		{
			name:           "Validator 1 in spans",
			epoch:          3,
			spansHex:       "0000000000000001000000000000",
			spansResultHex: "0000000000000001000000000000",
			validator1Span: types.Span{MinSpan: 1},
			err:            nil,
		},
	}

	exampleSpansValues = []spansValueTests{
		{
			name: "Validator 0 first time",
			validatorSpan: types.Span{
				MinSpan:     1,
				MaxSpan:     2,
				SigBytes:    [2]byte{1, 1},
				HasAttested: false,
			},
			spansLength: spannerEncodedLength,
			validatorID: 0,
		},
		{
			name: "Validator 300000 first time",
			validatorSpan: types.Span{
				MinSpan:     256,
				MaxSpan:     677,
				SigBytes:    [2]byte{255, 250},
				HasAttested: true,
			},
			validatorID: 300000,
			spansLength: spannerEncodedLength*300000 + spannerEncodedLength,
		},
		{
			name: "Validator 1 with highestObservedValidatorIdx 300000",
			validatorSpan: types.Span{
				MinSpan:     54000,
				MaxSpan:     54001,
				SigBytes:    [2]byte{250, 255},
				HasAttested: true,
			},
			validatorID: 1,
			spansLength: spannerEncodedLength*300000 + spannerEncodedLength,
		},
		{
			name: "Validator 0 not with old spans(disregards the highestObservedValidatorIdx)",
			validatorSpan: types.Span{
				MinSpan:     65535,
				MaxSpan:     65535,
				SigBytes:    [2]byte{255, 255},
				HasAttested: true,
			},
			validatorID: 0,
			oldSpans:    "01000000000000",
			spansLength: spannerEncodedLength,
		},
	}

}

func TestValidatorSpans_NilDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	validatorIdx := uint64(1)
	vsm, err := db.EpochSpans(ctx, validatorIdx)
	if err != nil {
		t.Fatalf("Nil EpochSpansMap should not return error: %v", err)
	}
	if !reflect.DeepEqual(vsm, []byte{}) {
		t.Fatal("EpochSpans should return empty byte array if no record exists in the db")
	}
}

func TestStore_SaveReadEpochSpans(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	for _, tt := range spanNewTests {
		spans, err := hex.DecodeString(tt.spansHex)
		if err != nil {
			t.Fatal(err)
		}
		err = db.SaveEpochSpans(ctx, tt.epoch, spans)
		if err != tt.err {
			t.Fatalf("Failed to get the right error expected: %v got: %v", tt.err, err)
		}
		sm, err := db.EpochSpans(ctx, tt.epoch)
		if err != nil {
			t.Fatalf("Failed to get validator spans: %v", err)
		}
		spansResult, err := hex.DecodeString(tt.spansResultHex)
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil || !reflect.DeepEqual(sm, spansResult) {
			t.Fatalf("Get should return validator spans: %v got: %v", spansResult, sm)
		}
		es := EpochStore{Spans: sm}
		s, err := es.GetValidatorSpan(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get validator span for epoch 1: %v", err)
		}
		if !reflect.DeepEqual(s, tt.validator1Span) {
			t.Fatalf("Get should return validator span for validator 2: %v got: %v", tt.validator1Span, s)
		}
	}
}

func TestStore_GetValidatorSpan(t *testing.T) {
	ctx := context.Background()
	tooSmall, err := hex.DecodeString("000000")
	if err != nil {
		t.Fatal(err)
	}
	es := EpochStore{Spans: tooSmall}

	span, err := es.GetValidatorSpan(ctx, 1)
	if !reflect.DeepEqual(span, types.Span{}) {
		t.Errorf("Expected empty span to be returned: %v", span)
	}
	if err != errWrongSize {
		t.Error("expected error")
	}
	tooBig, err := hex.DecodeString("0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	es.Spans = tooBig
	span, err = es.GetValidatorSpan(ctx, 1)
	if !reflect.DeepEqual(span, types.Span{}) {
		t.Errorf("Expected empty span to be returned: %v", span)
	}
	if err != errWrongSize {
		t.Error("Expected error")
	}
	oneValidator, err := hex.DecodeString("01010101010101")
	if err != nil {
		t.Fatal(err)
	}
	es.Spans = oneValidator
	span, err = es.GetValidatorSpan(ctx, 0)
	if !reflect.DeepEqual(span, types.Span{MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true}) {
		t.Errorf("Expected types.Span{MinSpan: 1, MaxSpan: 1, SigBytes: [2]byte{1, 1}, HasAttested: true} to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	span, err = es.GetValidatorSpan(ctx, 1)
	if !reflect.DeepEqual(span, types.Span{}) {
		t.Errorf("Expected empty span to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	twoValidator, err := hex.DecodeString("0101010101010101010101010101")
	if err != nil {
		t.Fatal(err)
	}
	es.Spans = twoValidator
	span, err = es.GetValidatorSpan(ctx, 0)
	if !reflect.DeepEqual(span, types.Span{MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true}) {
		t.Errorf("Expected types.Span{MinSpan: 1, MaxSpan: 1, SigBytes: [2]byte{1, 1}, HasAttested: true} to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	span, err = es.GetValidatorSpan(ctx, 1)
	if !reflect.DeepEqual(span, types.Span{MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true}) {
		t.Errorf("Expected types.Span{MinSpan: 1, MaxSpan: 1, SigBytes: [2]byte{1, 1}, HasAttested: true} to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStore_SetValidatorSpan(t *testing.T) {
	ctx := context.Background()
	for _, tt := range exampleSpansValues {
		oldSpans, err := hex.DecodeString(tt.oldSpans)
		if err != nil {
			t.Fatal(err)
		}
		es := EpochStore{Spans: oldSpans}

		err = es.SetValidatorSpan(ctx, tt.validatorID, tt.validatorSpan)
		if err != tt.err {
			t.Errorf("Expected error: %v got: %v", tt.err, err)
		}
		if uint64(len(es.Spans)) != tt.spansLength {
			t.Errorf("Expected spans length: %d got: %d", tt.spansLength, len(es.Spans))
		}
		span, err := es.GetValidatorSpan(ctx, tt.validatorID)
		if err != nil {
			t.Errorf("Got error while trying to get span from spans byte array: %v", err)
		}
		if !reflect.DeepEqual(span, tt.validatorSpan) {
			t.Errorf("Expected validator span: %v got: %v ", tt.validatorSpan, span)
		}

	}

}
