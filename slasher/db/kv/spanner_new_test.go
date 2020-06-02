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

var spanNewTests []spansTestStruct

func init() {
	spanNewTests = []spansTestStruct{
		{
			name:           "span too small",
			epoch:          1,
			spansHex:       "00000000",
			spansResultHex: "",
			validator1Span: types.Span{},
			err:            ErrWrongSize,
		},
		{
			name:           "no validator 1 in spans",
			epoch:          2,
			spansHex:       "00000000000000",
			spansResultHex: "00000000000000",
			validator1Span: types.Span{},
			err:            nil,
		},
		{
			name:           "validator 1 in spans",
			epoch:          3,
			spansHex:       "0000000000000001000000000000",
			spansResultHex: "0000000000000001000000000000",
			validator1Span: types.Span{MinSpan: 1},
			err:            nil,
		},
	}

}

func TestValidatorSpans_NilDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	validatorIdx := uint64(1)
	es, err := db.EpochSpans(ctx, validatorIdx)
	if err != nil {
		t.Fatalf("Nil EpochSpansMap should not return error: %v", err)
	}
	if !reflect.DeepEqual(es, EpochStore{}) {
		t.Fatal("EpochSpans should return empty byte array if no record exists in the db")
	}
}

func TestStore_SaveReadEpochSpans(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	for _, tt := range spanNewTests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := hex.DecodeString(tt.spansHex)
			if err != nil {
				t.Fatal(err)
			}
			es := EpochStore{}
			es = spans
			err = db.SaveEpochSpans(ctx, tt.epoch, es)
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
			esr := EpochStore{}
			esr = spansResult
			if !reflect.DeepEqual(sm, esr) {
				t.Fatalf("Get should return validator spans: %v got: %v", spansResult, sm)
			}

			s, err := es.GetValidatorSpan(ctx, 1)
			if err != tt.err {
				t.Fatalf("Failed to get validator 1 span: %v", err)
			}
			if !reflect.DeepEqual(s, tt.validator1Span) {
				t.Fatalf("Get should return validator span for validator 2: %v got: %v", tt.validator1Span, s)
			}
		})
	}
}
