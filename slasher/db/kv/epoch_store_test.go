package kv

import (
	"context"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

type spansValueTests struct {
	name          string
	validatorID   uint64
	oldSpans      string
	spansLength   uint64
	validatorSpan types.Span
	err           error
}

var exampleSpansValues []spansValueTests

func init() {
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

func TestStore_GetValidatorSpan(t *testing.T) {
	ctx := context.Background()
	tooSmall, err := hex.DecodeString("000000")
	if err != nil {
		t.Fatal(err)
	}
	es, err := NewEpochStore(tooSmall)
	if err != ErrWrongSize {
		t.Error("expected error")
	}
	//nil es
	span, err := es.GetValidatorSpan(ctx, 1)
	if !reflect.DeepEqual(span, types.Span{}) {
		t.Errorf("Expected empty span to be returned: %v", span)
	}
	tooBig, err := hex.DecodeString("0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	es = tooBig
	span, err = es.GetValidatorSpan(ctx, 1)
	if !reflect.DeepEqual(span, types.Span{}) {
		t.Errorf("Expected empty span to be returned: %v", span)
	}
	if err != ErrWrongSize {
		t.Error("Expected error")
	}
	oneValidator, err := hex.DecodeString("01010101010101")
	if err != nil {
		t.Fatal(err)
	}
	es = oneValidator
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
	es = twoValidator
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
		es, err := NewEpochStore(oldSpans)
		if err != tt.err {
			t.Errorf("Expected error: %v got: %v", tt.err, err)
		}
		err = es.SetValidatorSpan(ctx, tt.validatorID, tt.validatorSpan)
		if uint64(len(es)) != tt.spansLength {
			t.Errorf("Expected spans length: %d got: %d", tt.spansLength, len(es))
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
