package types_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
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
			spansLength: types.SpannerEncodedLength,
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
			spansLength: types.SpannerEncodedLength*300000 + types.SpannerEncodedLength,
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
			spansLength: types.SpannerEncodedLength*300000 + types.SpannerEncodedLength,
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
			spansLength: types.SpannerEncodedLength,
		},
	}
}

func TestStore_GetValidatorSpan(t *testing.T) {
	tooSmall, err := hex.DecodeString("000000")
	if err != nil {
		t.Fatal(err)
	}
	es, err := types.NewEpochStore(tooSmall)
	if err != types.ErrWrongSize {
		t.Error("expected error")
	}
	//nil es
	span, err := es.GetValidatorSpan(1)
	if !reflect.DeepEqual(span, types.Span{}) {
		t.Errorf("Expected empty span to be returned: %v", span)
	}
	tooBig, err := hex.DecodeString("0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	es, err = types.NewEpochStore(tooBig)
	if err != types.ErrWrongSize {
		t.Error("Expected error")
	}
	oneValidator, err := hex.DecodeString("01010101010101")
	if err != nil {
		t.Fatal(err)
	}
	es, err = types.NewEpochStore(oneValidator)
	if err != nil {
		t.Fatal(err)
	}
	span, err = es.GetValidatorSpan(0)
	if !reflect.DeepEqual(span, types.Span{MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true}) {
		t.Errorf("Expected Span{MinSpan: 1, MaxSpan: 1, SigBytes: [2]byte{1, 1}, HasAttested: true} to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	span, err = es.GetValidatorSpan(1)
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
	es, err = types.NewEpochStore(twoValidator)
	if err != nil {
		t.Fatal(err)
	}
	span, err = es.GetValidatorSpan(0)
	if !reflect.DeepEqual(span, types.Span{MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true}) {
		t.Errorf("Expected types.Span{MinSpan: 1, MaxSpan: 1, SigBytes: [2]byte{1, 1}, HasAttested: true} to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	span, err = es.GetValidatorSpan(1)
	if !reflect.DeepEqual(span, types.Span{MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true}) {
		t.Errorf("Expected types.Span{MinSpan: 1, MaxSpan: 1, SigBytes: [2]byte{1, 1}, HasAttested: true} to be returned: %v", span)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStore_SetValidatorSpan(t *testing.T) {
	for _, tt := range exampleSpansValues {
		t.Run(tt.name, func(t *testing.T) {
			oldSpans, err := hex.DecodeString(tt.oldSpans)
			if err != nil {
				t.Fatal(err)
			}
			es, err := types.NewEpochStore(oldSpans)
			if err != tt.err {
				t.Errorf("Expected error: %v got: %v", tt.err, err)
			}
			es, err = es.SetValidatorSpan(tt.validatorID, tt.validatorSpan)
			if err != nil {
				t.Fatal(err)
			}
			if es.HighestObservedIdx() != tt.validatorID {
				t.Fatalf("expected highest idx %d, received %d", tt.validatorID, es.HighestObservedIdx())
			}
			spans := es.Bytes()
			spanLen := len(spans)
			if uint64(spanLen) != tt.spansLength {
				t.Errorf("Expected spans length: %d got: %d", tt.spansLength, len(es.Bytes()))
			}
			span, err := es.GetValidatorSpan(tt.validatorID)
			if err != nil {
				t.Errorf("Got error while trying to get span from spans byte array: %v", err)
			}
			if !reflect.DeepEqual(span, tt.validatorSpan) {
				t.Errorf("Expected validator span: %v got: %v ", tt.validatorSpan, span)
			}
		})

	}
}

func BenchmarkEpochStore_Save(b *testing.B) {
	amount := uint64(100000)
	store, spansMap := generateEpochStore(b, amount)

	b.Run(fmt.Sprintf("%d old", amount), func(b *testing.B) {
		db := testDB.SetupSlasherDB(b, false)
		db.EnableSpanCache(false)
		b.ResetTimer()
		b.ReportAllocs()
		b.N = 5
		for i := 0; i < b.N; i++ {
			if err := db.SaveEpochSpansMap(context.Background(), 0, spansMap); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run(fmt.Sprintf("%d new", amount), func(b *testing.B) {
		db := testDB.SetupSlasherDB(b, false)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := db.SaveEpochSpans(context.Background(), 1, store); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func generateEpochStore(t testing.TB, n uint64) (*types.EpochStore, map[uint64]types.Span) {
	epochStore, err := types.NewEpochStore([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	spanMap := make(map[uint64]types.Span)
	for i := uint64(0); i < n; i++ {
		span := types.Span{
			MinSpan:     14,
			MaxSpan:     8,
			SigBytes:    [2]byte{5, 13},
			HasAttested: true,
		}
		spanMap[i] = span
		epochStore, err = epochStore.SetValidatorSpan(i, span)
		if err != nil {
			t.Fatal(err)
		}
	}
	return epochStore, spanMap
}
